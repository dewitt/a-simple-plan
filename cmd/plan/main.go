package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/dewitt/dewitt-blog/internal/config"
	"github.com/dewitt/dewitt-blog/internal/render"
)

type PlanContext struct {
	PlanDir      string
	PlanFile     string // Relative to PlanDir
	OutputDir    string
	Config       config.Config
	Template     string // Custom template content
}

func main() {
	// Define flags
	var inputPath string
	flag.StringVar(&inputPath, "f", ".", "Path to the plan file or directory")
	flag.StringVar(&inputPath, "file", ".", "Path to the plan file or directory")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: plan [options] <command>\n")
		fmt.Fprintf(os.Stderr, "\nCommands:\n")
		fmt.Fprintf(os.Stderr, "  preview  - Render locally and open in browser\n")
		fmt.Fprintf(os.Stderr, "  build    - Generate static HTML in 'public' directory\n")
		fmt.Fprintf(os.Stderr, "  save     - Commit changes locally\n")
		fmt.Fprintf(os.Stderr, "  publish  - Commit and push to origin\n")
		fmt.Fprintf(os.Stderr, "  revert   - Discard local changes\n")
		fmt.Fprintf(os.Stderr, "  rollback - Revert to previous version and publish\n")
		fmt.Fprintf(os.Stderr, "  edit     - Open plan file in default editor\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	// Parse flags after command
	if len(os.Args) > 2 {
		if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
			os.Exit(1)
		}
	} else if cmd == "-h" || cmd == "--help" {
		flag.Usage()
		return
	}

	// Initialize Context
	ctx, err := initContext(inputPath)
	if err != nil {
		log.Fatalf("Initialization failed: %v", err)
	}

	switch cmd {
	case "preview":
		preview(ctx)
	case "build":
		build(ctx)
	case "save":
		save(ctx)
	case "publish":
		publish(ctx)
	case "revert":
		revert(ctx)
	case "rollback":
		args := flag.CommandLine.Args()
		commit := ""
		if len(args) > 0 {
			commit = args[0]
		}
		rollback(ctx, commit)
	case "edit":
		edit(ctx)
	case "-h", "--help":
		flag.Usage()
	default:
		if strings.HasPrefix(cmd, "-") {
			fmt.Printf("Unknown command or invalid usage: %s\n", cmd)
			flag.Usage()
			os.Exit(1)
		}
		fmt.Printf("Unknown command: %s\n", cmd)
		flag.Usage()
		os.Exit(1)
	}
}

// initContext resolves the plan directory and file, loads configuration, and reads any custom template.
// It handles both file paths (-f plan.md) and directory paths (-f ./my-plan).
func initContext(path string) (*PlanContext, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat input path: %w", err)
	}

	var planDir, planFile string
	if info.IsDir() {
		planDir = path
		planFile = "plan.md"
	} else {
		planDir = filepath.Dir(path)
		planFile = filepath.Base(path)
	}

	// Abs path for clarity
	absDir, err := filepath.Abs(planDir)
	if err != nil {
		return nil, err
	}
	planDir = absDir

	// Load Config
	cfg, err := config.Load(filepath.Join(planDir, "settings.json"))
	if err != nil {
		log.Printf("Warning: Failed to load settings.json, using defaults: %v", err)
		cfg = config.DefaultConfig()
	}

	// Load Template
	tmplContent := ""
	tmplPath := filepath.Join(planDir, "template.html")
	if tmplBytes, err := os.ReadFile(tmplPath); err == nil {
		tmplContent = string(tmplBytes)
	}

	return &PlanContext{
		PlanDir:   planDir,
		PlanFile:  planFile,
		OutputDir: filepath.Join(planDir, "public"),
		Config:    cfg,
		Template:  tmplContent,
	}, nil
}

func preview(ctx *PlanContext) {
	port := "8081"
	build(ctx)

	fs := http.FileServer(http.Dir(ctx.OutputDir))
	http.Handle("/", fs)

	fmt.Printf("Starting preview server at http://localhost:%s/index.html\n", port)

	go func() {
		time.Sleep(500 * time.Millisecond)
		openBrowser("http://localhost:" + port + "/index.html")
	}()

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func build(ctx *PlanContext) {
	fullPath := filepath.Join(ctx.PlanDir, ctx.PlanFile)
	fmt.Printf("Building %s...\n", fullPath)

	content, err := os.ReadFile(fullPath)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}
	info, err := os.Stat(fullPath)
	if err != nil {
		log.Fatalf("Failed to stat file: %v", err)
	}

	if err := renderAndWrite(ctx, content, info.ModTime(), filepath.Join(ctx.OutputDir, "index.html")); err != nil {
		log.Fatalf("Failed to build current page: %v", err)
	}

	if err := buildHistory(ctx); err != nil {
		log.Printf("Warning: Failed to build history (is this a git repo?): %v", err)
	}

	fmt.Println("Build complete.")
}

// buildHistory reconstructs the past versions of the plan file using git history.
// It iterates through unique dates in the git log, retrieves the file content for that date,
// and generates static pages for each day, as well as year and month index pages.
func buildHistory(ctx *PlanContext) error {
	fmt.Println("Building history...")

	history, err := getGitHistory(ctx.PlanDir, ctx.PlanFile)
	if err != nil {
		return err
	}

	var dates []string
	for d := range history {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	type dayEntry struct {
		DateStr string
		Path    string
	}
	tree := make(map[string]map[string][]dayEntry)

	for _, dateStr := range dates {
		hash := history[dateStr]
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		year := t.Format("2006")
		month := t.Format("01")
		day := t.Format("02")

		content, err := getGitContent(ctx.PlanDir, hash, ctx.PlanFile)
		if err != nil {
			log.Printf("Failed to get content for %s: %v", dateStr, err)
			continue
		}

		outDir := filepath.Join(ctx.OutputDir, year, month, day)
		outPath := filepath.Join(outDir, "index.html")

		if err := renderAndWrite(ctx, content, t, outPath); err != nil {
			return err
		}

		if tree[year] == nil {
			tree[year] = make(map[string][]dayEntry)
		}
		tree[year][month] = append(tree[year][month], dayEntry{
			DateStr: dateStr,
			Path:    fmt.Sprintf("/%s/%s/%s", year, month, day),
		})
	}

	// Generate Indices
	for year, months := range tree {
		var yearLinks []dayEntry
		for _, mDays := range months {
			yearLinks = append(yearLinks, mDays...)
		}
		sort.Slice(yearLinks, func(i, j int) bool {
			return yearLinks[i].DateStr > yearLinks[j].DateStr
		})

		yearContent := fmt.Sprintf("# History for %s\n\n", year)
		for _, link := range yearLinks {
			yearContent += fmt.Sprintf("- [%s](%s)\n", link.DateStr, link.Path)
		}
		if err := renderAndWrite(ctx, []byte(yearContent), time.Now(), filepath.Join(ctx.OutputDir, year, "index.html")); err != nil {
			return err
		}

		for month, days := range months {
			sort.Slice(days, func(i, j int) bool {
				return days[i].DateStr > days[j].DateStr
			})
			monthName := month
			if t, _ := time.Parse("01", month); !t.IsZero() {
				monthName = t.Format("January")
			}
			monthContent := fmt.Sprintf("# History for %s %s\n\n", monthName, year)
			for _, link := range days {
				monthContent += fmt.Sprintf("- [%s](%s)\n", link.DateStr, link.Path)
			}
			if err := renderAndWrite(ctx, []byte(monthContent), time.Now(), filepath.Join(ctx.OutputDir, year, month, "index.html")); err != nil {
				return err
			}
		}
	}
	return nil
}

func renderAndWrite(ctx *PlanContext, content []byte, modTime time.Time, outPath string) error {
	r := render.New(&ctx.Config, ctx.Template)

	body, err := r.RenderBody(content)
	if err != nil {
		return fmt.Errorf("rendering body: %w", err)
	}

	html, err := r.Compose(body, modTime, modTime)
	if err != nil {
		return fmt.Errorf("composing html: %w", err)
	}

	dir := filepath.Dir(outPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating dir %s: %w", dir, err)
	}

	if err := os.WriteFile(outPath, html, 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", outPath, err)
	}
	return nil
}

func getGitHistory(dir, file string) (map[string]string, error) {
	cmd := exec.Command("git", "log", "--date=format:%Y-%m-%d", "--format=%H %ad", file)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	history := make(map[string]string)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		hash := parts[0]
		date := parts[1]
		if _, exists := history[date]; !exists {
			history[date] = hash
		}
	}
	return history, nil
}

func getGitContent(dir, hash, file string) ([]byte, error) {
	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", hash, file))
	cmd.Dir = dir
	return cmd.Output()
}

func runCmd(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func save(ctx *PlanContext) {
	fmt.Printf("Saving %s...\n", ctx.PlanFile)
	if err := runCmd(ctx.PlanDir, "git", "add", ctx.PlanFile); err != nil {
		log.Fatalf("Failed to add file: %v", err)
	}
	if err := runCmd(ctx.PlanDir, "git", "commit", "-m", "Update plan"); err != nil {
		fmt.Println("Nothing to commit or commit failed.")
	} else {
		fmt.Println("Changes committed.")
	}
}

func publish(ctx *PlanContext) {
	save(ctx)
	fmt.Println("Pushing to origin...")
	if err := runCmd(ctx.PlanDir, "git", "push"); err != nil {
		log.Fatalf("Failed to push: %v", err)
	}
	fmt.Println("Successfully pushed to origin.")
}

func revert(ctx *PlanContext) {
	fmt.Printf("Reverting %s...\n", ctx.PlanFile)
	if err := runCmd(ctx.PlanDir, "git", "checkout", ctx.PlanFile); err != nil {
		log.Fatalf("Failed to revert: %v", err)
	}
	fmt.Println("Local changes discarded.")
}

func rollback(ctx *PlanContext, commit string) {
	target := commit
	if target == "" {
		target = "HEAD~1"
	}
	fmt.Printf("Rolling back %s to version %s...\n", ctx.PlanFile, target)
	if err := runCmd(ctx.PlanDir, "git", "checkout", target, "--", ctx.PlanFile); err != nil {
		log.Fatalf("Failed to checkout version %s: %v", target, err)
	}
	fmt.Println("Committing rollback...")
	if err := runCmd(ctx.PlanDir, "git", "commit", "-m", fmt.Sprintf("Rollback %s to %s", ctx.PlanFile, target)); err != nil {
		log.Fatalf("Failed to commit rollback: %v", err)
	}
	fmt.Println("Pushing to origin...")
	if err := runCmd(ctx.PlanDir, "git", "push"); err != nil {
		log.Fatalf("Failed to push: %v", err)
	}
	fmt.Println("Rolled back and pushed.")
}

func edit(ctx *PlanContext) {
	file := filepath.Join(ctx.PlanDir, ctx.PlanFile)
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		switch runtime.GOOS {
		case "windows":
			editor = "notepad"
		case "darwin":
			editor = "open -t"
		default:
			editor = "vi"
		}
	}
	fmt.Printf("Opening %s with %s...\n", file, editor)

	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" && editor == "open -t" {
		cmd = exec.Command("open", "-t", file)
	} else if runtime.GOOS != "windows" {
		cmd = exec.Command("sh", "-c", editor+" \"\"", "editor_wrapper", file)
	} else {
		cmd = exec.Command(editor, file)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to open editor: %v", err)
	}
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

