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

	"github.com/dewitt/dewitt-blog/internal/render"
)

func main() {
	// Define flags
	var planFile string
	flag.StringVar(&planFile, "f", "plan.md", "Path to the plan file")
	flag.StringVar(&planFile, "file", "plan.md", "Path to the plan file")

	// Custom usage to show commands
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

	switch cmd {
	case "preview":
		if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
			os.Exit(1)
		}
		preview(planFile)
	case "build":
		if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
			os.Exit(1)
		}
		build(planFile)
	case "save":
		if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
			os.Exit(1)
		}
		save(planFile)
	case "publish":
		if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
			os.Exit(1)
		}
		publish(planFile)
	case "revert":
		if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
			os.Exit(1)
		}
		revert(planFile)
	case "rollback":
		if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
			os.Exit(1)
		}
		args := flag.CommandLine.Args()
		commit := ""
		if len(args) > 0 {
			commit = args[0]
		}
		rollback(planFile, commit)
	case "edit":
		if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
			os.Exit(1)
		}
		edit(planFile)
	case "-h", "--help":
		flag.Usage()
	default:
		if cmd[0] == '-' {
			fmt.Printf("Unknown command or invalid usage: %s\n", cmd)
			flag.Usage()
			os.Exit(1)
		}
		fmt.Printf("Unknown command: %s\n", cmd)
		flag.Usage()
		os.Exit(1)
	}
}

func preview(file string) {
	port := "8081"
	build(file)

	fs := http.FileServer(http.Dir("public"))
	http.Handle("/", fs)

	fmt.Printf("Starting preview server for static content at http://localhost:%s/index.html\n", port)

	go func() {
		time.Sleep(500 * time.Millisecond)
		openBrowser("http://localhost:" + port + "/index.html")
	}()

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func build(file string) {
	fmt.Printf("Building %s...\n", file)

	// 1. Build current version
	content, err := os.ReadFile(file)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}
	info, err := os.Stat(file)
	if err != nil {
		log.Fatalf("Failed to stat file: %v", err)
	}

	if err := renderAndWrite(content, info.ModTime(), "public/index.html"); err != nil {
		log.Fatalf("Failed to build current page: %v", err)
	}

	// 2. Build history
	if err := buildHistory(file); err != nil {
		// Log error but don't fail the whole build?
		// Or fail strictly? Let's fail strictly to be safe.
		// Use specific error message to help debugging
		log.Printf("Warning: Failed to build history (is this a git repo?): %v", err)
	}

	fmt.Println("Build complete.")
}

func buildHistory(file string) error {
	fmt.Println("Building history...")

	// Get history: Date -> CommitHash
	history, err := getGitHistory(file)
	if err != nil {
		return err
	}

	var dates []string
	for d := range history {
		dates = append(dates, d)
	}
	sort.Strings(dates) // Ascending order

	// Reverse to have newest first for iteration if needed,
	// but for "links" we probably want newest first.
	// Let's sort descending.
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	// Store dates for index generation: Year -> Month -> [Days]
	type dayEntry struct {
		DateStr string // YYYY-MM-DD
		Path    string
	}
	tree := make(map[string]map[string][]dayEntry)

	for _, dateStr := range dates {
		hash := history[dateStr]

		// Parse date components
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			log.Printf("Skipping invalid date %s: %v", dateStr, err)
			continue
		}
		year := t.Format("2006")
		month := t.Format("01")
		day := t.Format("02")

		// Retrieve content
		content, err := getGitContent(hash, file)
		if err != nil {
			log.Printf("Failed to get content for %s (%s): %v", dateStr, hash, err)
			continue
		}

		// Output path: public/YYYY/MM/DD/index.html
		outDir := filepath.Join("public", year, month, day)
		outPath := filepath.Join(outDir, "index.html")

		if err := os.MkdirAll(outDir, 0755); err != nil {
			return err
		}

		// Render
		// For historical posts, we use the commit date as the "modTime"
		// And also as "created" time for display purposes.
		if err := renderAndWrite(content, t, outPath); err != nil {
			return err
		}

		// Add to tree
		if tree[year] == nil {
			tree[year] = make(map[string][]dayEntry)
		}
		tree[year][month] = append(tree[year][month], dayEntry{
			DateStr: dateStr,
			Path:    fmt.Sprintf("/%s/%s/%s", year, month, day),
		})
	}

	// Generate Index Pages
	for year, months := range tree {
		// 1. Year Index: public/YYYY/index.html
		// Lists links to days (or months? Prompt says "links to any days")
		// Let's list all days in the year.
		var yearLinks []dayEntry
		for _, mDays := range months {
			yearLinks = append(yearLinks, mDays...)
		}
		// Sort yearLinks descending
		sort.Slice(yearLinks, func(i, j int) bool {
			return yearLinks[i].DateStr > yearLinks[j].DateStr
		})

		yearContent := fmt.Sprintf("# History for %s\n\n", year)
		for _, link := range yearLinks {
			yearContent += fmt.Sprintf("- [%s](%s)\n", link.DateStr, link.Path)
		}

		yearPath := filepath.Join("public", year, "index.html")
		if err := renderAndWrite([]byte(yearContent), time.Now(), yearPath); err != nil {
			return err
		}

		// 2. Month Indices: public/YYYY/MM/index.html
		for month, days := range months {
			// Sort days descending
			sort.Slice(days, func(i, j int) bool {
				return days[i].DateStr > days[j].DateStr
			})

			monthName := month // Could use time package to get "January", but "01" is safe
			t, _ := time.Parse("01", month)
			if !t.IsZero() {
				monthName = t.Format("January")
			}

			monthContent := fmt.Sprintf("# History for %s %s\n\n", monthName, year)
			for _, link := range days {
				monthContent += fmt.Sprintf("- [%s](%s)\n", link.DateStr, link.Path)
			}

			monthPath := filepath.Join("public", year, month, "index.html")
			if err := renderAndWrite([]byte(monthContent), time.Now(), monthPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func renderAndWrite(content []byte, modTime time.Time, outPath string) error {
	r := render.New()

	// Render body (Markdown -> HTML fragment)
	body, err := r.RenderBody(content)
	if err != nil {
		return fmt.Errorf("rendering body: %w", err)
	}

	// Compose full HTML
	html, err := r.Compose(body, modTime, modTime)
	if err != nil {
		return fmt.Errorf("composing html: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(outPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating dir %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(outPath, html, 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", outPath, err)
	}
	return nil
}

// getGitHistory returns a map of Date(YYYY-MM-DD) -> LatestCommitHash
func getGitHistory(file string) (map[string]string, error) {
	// git log --date=format:'%Y-%m-%d' --format="%H %ad" file
	cmd := exec.Command("git", "log", "--date=format:%Y-%m-%d", "--format=%H %ad", file)
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

		// Since git log is descending, the first time we see a date, it is the latest for that date.
		if _, exists := history[date]; !exists {
			history[date] = hash
		}
	}
	return history, nil
}

func getGitContent(hash, file string) ([]byte, error) {
	// git show hash:file
	// Note: file path in git show should be relative to repo root usually,
	// but if we are in root, it's fine.
	// git show hash:./file might work too.
	// Using generic "git show hash:path"
	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", hash, file))
	return cmd.Output()
}

func save(file string) {
	fmt.Printf("Saving %s...\n", file)

	if err := runCmd("git", "add", file); err != nil {
		log.Fatalf("Failed to add file: %v", err)
	}

	// Commit
	if err := runCmd("git", "commit", "-m", "Update plan"); err != nil {
		fmt.Println("Nothing to commit or commit failed.")
	} else {
		fmt.Println("Changes committed.")
	}
}

func publish(file string) {
	save(file)
	fmt.Println("Pushing to origin...")
	if err := runCmd("git", "push"); err != nil {
		log.Fatalf("Failed to push: %v", err)
	}
	fmt.Println("Successfully pushed to origin.")
}

func revert(file string) {
	fmt.Printf("Reverting %s...\n", file)
	if err := runCmd("git", "checkout", file); err != nil {
		log.Fatalf("Failed to revert: %v", err)
	}
	fmt.Println("Local changes discarded.")
}

func rollback(file, commit string) {
	target := commit
	if target == "" {
		target = "HEAD~1"
	}
	fmt.Printf("Rolling back %s to version %s...\n", file, target)
	if err := runCmd("git", "checkout", target, "--", file); err != nil {
		log.Fatalf("Failed to checkout version %s: %v", target, err)
	}
	fmt.Println("Committing rollback...")
	if err := runCmd("git", "commit", "-m", fmt.Sprintf("Rollback %s to %s", file, target)); err != nil {
		log.Fatalf("Failed to commit rollback: %v", err)
	}
	fmt.Println("Pushing to origin...")
	if err := runCmd("git", "push"); err != nil {
		log.Fatalf("Failed to push: %v", err)
	}
	fmt.Println("Rolled back and pushed.")
}

func edit(file string) {
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
		cmd = exec.Command("sh", "-c", editor+" \"$1\"", "editor_wrapper", file)
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

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
