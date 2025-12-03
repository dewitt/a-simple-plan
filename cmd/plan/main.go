package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/dewitt/a-simple-plan/internal/config"
	"github.com/dewitt/a-simple-plan/internal/render"
)

type PlanContext struct {
	PlanDir   string
	PlanFile  string // Relative to PlanDir
	OutputDir string
	Config    config.Config
	Template  string // Custom template content
	CreationTime time.Time
	LiveReload   bool
}

type Rss struct {
	XMLName xml.Name `xml:"rss"`

	Version string `xml:"version,attr"`

	ContentNs string `xml:"xmlns:content,attr"`

	Channel Channel `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Content     string `xml:"content:encoded"`
	PubDate     string `xml:"pubDate"`
	Guid        string `xml:"guid"`
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
		fmt.Fprintf(os.Stderr, "  debug    - Print debug information\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}
	
	// Manually parse args to handle flags before the subcommand
	// Standard flag.Parse() stops at the first non-flag argument (the subcommand)
	flag.Parse()

	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	cmd := flag.Arg(0)

	// Re-parse flags if they were placed after the command (legacy support / user convenience)
	// This is a bit tricky because flag.Parse() already consumed what it could.
	// But since we want to support `plan preview -f ...` and `plan -f ... preview`,
	// we can check if there are args after the command.
	if len(flag.Args()) > 1 {
		// Create a new flag set to parse the remaining arguments
		subFs := flag.NewFlagSet("subcommand", flag.ContinueOnError)
		subFs.StringVar(&inputPath, "f", inputPath, "Path to the plan file or directory")
		subFs.StringVar(&inputPath, "file", inputPath, "Path to the plan file or directory")
		subFs.Parse(flag.Args()[1:])
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
	case "debug":
		debugCmd(ctx)
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

func debugCmd(ctx *PlanContext) {
	fmt.Println("=== Plan Debug Info ===")
	
	// Build Info
	fmt.Println("\n-- Build Information --")
	if info, ok := debug.ReadBuildInfo(); ok {
		fmt.Printf("Go Version:   %s\n", info.GoVersion)
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				fmt.Printf("Git Revision: %s\n", setting.Value)
			}
			if setting.Key == "vcs.time" {
				fmt.Printf("Git Time:     %s\n", setting.Value)
			}
			if setting.Key == "vcs.modified" && setting.Value == "true" {
				fmt.Println("Git Status:   dirty")
			}
		}
	} else {
		fmt.Println("Build info not available.")
	}
	fmt.Printf("OS/Arch:      %s/%s\n", runtime.GOOS, runtime.GOARCH)

	// Context Info
	fmt.Println("\n-- Context --")
	cwd, _ := os.Getwd()
	fmt.Printf("Working Dir:  %s\n", cwd)
	fmt.Printf("Plan Dir:     %s\n", ctx.PlanDir)
	fmt.Printf("Plan File:    %s\n", ctx.PlanFile)
	fmt.Printf("Output Dir:   %s\n", ctx.OutputDir)
	fmt.Printf("Creation:     %s\n", ctx.CreationTime.Format(time.RFC3339))

	// Configuration
	fmt.Println("\n-- Configuration --")
	fmt.Printf("Username:     %s\n", ctx.Config.Username)
	fmt.Printf("FullName:     %s\n", ctx.Config.FullName)
	fmt.Printf("Title:        %s\n", ctx.Config.Title)
	fmt.Printf("Timezone:     %s\n", ctx.Config.Timezone)
	
	tmplStatus := "Default (Embedded)"
	if len(ctx.Template) > 0 {
		// Simple check to see if it matches default is hard since default is embedded in another package
		// But if initContext loaded it from disk, we know. 
		// Actually initContext sets ctx.Template to file content if found, else empty string?
		// Re-reading initContext: 
		// tmplContent is set ONLY if os.ReadFile succeeds. 
		// But render.New defaults if passed empty string.
		// So if ctx.Template is NOT empty, it was loaded from file.
		tmplStatus = "Custom (Loaded from file)"
	} else {
		// It might still be empty string in ctx, but renderer uses default.
		tmplStatus = "Default (Embedded)"
	}
	fmt.Printf("Template:     %s\n", tmplStatus)

	// Git Status of Plan
	fmt.Println("\n-- Plan Git Status --")
	if _, err := exec.LookPath("git"); err == nil {
		cmd := exec.Command("git", "status", "-s", ctx.PlanFile)
		cmd.Dir = ctx.PlanDir
		out, err := cmd.CombinedOutput()
		if err == nil {
			status := strings.TrimSpace(string(out))
			if status == "" {
				fmt.Println("Status:       Clean")
			} else {
				fmt.Printf("Status:       %s\n", status)
			}
		} else {
			fmt.Printf("Status:       Error checking git status (%v)\n", err)
		}

		cmdLog := exec.Command("git", "log", "-1", "--format=%h - %s (%an)", ctx.PlanFile)
		cmdLog.Dir = ctx.PlanDir
		outLog, errLog := cmdLog.CombinedOutput()
		if errLog == nil {
			fmt.Printf("Last Commit:  %s", string(outLog))
		} else {
			fmt.Println("Last Commit:  (None or not a git repo)")
		}
	} else {
		fmt.Println("Git not found in PATH")
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

	// Determine creation time
	creationTime := getCreationTime(planDir, planFile)
	if creationTime.IsZero() {
		// Fallback to file mod time if git fails or no commits
		if info, err := os.Stat(filepath.Join(planDir, planFile)); err == nil {
			creationTime = info.ModTime()
		} else {
			creationTime = time.Now()
		}
	}

	return &PlanContext{
		PlanDir:      planDir,
		PlanFile:     planFile,
		OutputDir:    filepath.Join(planDir, "public"),
		Config:       cfg,
		Template:     tmplContent,
		CreationTime: creationTime,
	}, nil
}

func getCreationTime(dir, file string) time.Time {
	cmd := exec.Command("git", "log", "--reverse", "--format=%aI", file)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) > 0 {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(lines[0]))
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

func preview(ctx *PlanContext) {
	ctx.LiveReload = true
	port := "8081"
	
	// Initial build
	build(ctx)

	// Setup SSE
	reloadCh := make(chan struct{})
	
	// Setup File Watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// Rebuild on write to plan file or template
				if event.Op&fsnotify.Write == fsnotify.Write {
					filename := filepath.Base(event.Name)
					if filename == ctx.PlanFile || filename == "template.html" || filename == "settings.json" {
						fmt.Printf("Change detected in %s, rebuilding...\n", filename)
						// Re-initialize context to catch settings/template changes
						// But for simplicity, we just rebuild mostly. 
						// To properly reload settings/template, we'd need to re-run init logic.
						// For now, let's just re-read the template if it changed? 
						// Simpler: Just call build. If template changed, we might need to reload it.
						// Let's do a partial context refresh if needed, but for now just Build.
						// Actually build() re-reads plan.md, but initContext loaded template. 
						// So if template.html changes, we need to update ctx.Template.
						
						if filename == "template.html" {
							if tmplBytes, err := os.ReadFile(filepath.Join(ctx.PlanDir, "template.html")); err == nil {
								ctx.Template = string(tmplBytes)
							}
						}

						build(ctx)
						
						// Notify clients
						// Non-blocking send
						go func() {
							// Give time for file system to settle / browser to be ready?
							// Send multiple signals?
							reloadCh <- struct{}{} 
						}()
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(ctx.PlanDir)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	
	// Re-implementing the handler logic properly
	
	broker := make(chan chan struct{})
	remove := make(chan chan struct{})
	broadcast := make(chan struct{})
	
	go func() {
		active := make(map[chan struct{}]bool)
		for {
			select {
			case c := <-broker:
				active[c] = true
			case c := <-remove:
				delete(active, c)
			case <-broadcast:
				for c := range active {
					select {
					case c <- struct{}{}:
					default:
					}
				}
			}
		}
	}()
	
	// Hook watcher to broadcast
	go func() {
		for range reloadCh {
			broadcast <- struct{}{}
		}
	}()

	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		ch := make(chan struct{}, 1)
		broker <- ch
		defer func() { remove <- ch }()

		// Heartbeat to keep connection alive
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-ch:
				fmt.Fprintf(w, "data: reload\n\n")
				flusher.Flush()
			case <-ticker.C:
				fmt.Fprintf(w, ": heartbeat\n\n")
				flusher.Flush()
			}
		}
	})

	fs := http.FileServer(http.Dir(ctx.OutputDir))
	mux.Handle("/", fs)

	fmt.Printf("Starting preview server at http://localhost:%s/index.html\n", port)
	fmt.Println("Watching for changes...")

	go func() {
		time.Sleep(500 * time.Millisecond)
		openBrowser("http://localhost:" + port + "/index.html")
	}()

	if err := http.ListenAndServe(":"+port, mux); err != nil {
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

	// Prepare RSS items
	var rssItems []Item

	// Add current post
	r := render.New(&ctx.Config, ctx.Template, false)
	bodyBytes, err := r.RenderBody(content)
	if err == nil {
		link := ctx.Config.BaseURL + "/"
		item := Item{
			Title:       ctx.Config.Title,
			Link:        link,
			Description: string(bodyBytes),
			Content:     string(bodyBytes),
			PubDate:     info.ModTime().Format(time.RFC1123Z),
			Guid:        link,
		}
		rssItems = append(rssItems, item)
	} else {
		log.Printf("Warning: Failed to render current content for RSS: %v", err)
	}

	historyItems, err := buildHistory(ctx)
	if err != nil {
		log.Printf("Warning: Failed to build history (is this a git repo?): %v", err)
	}
	rssItems = append(rssItems, historyItems...)

	// Generate RSS Feed
	rss := Rss{
		Version:   "2.0",
		ContentNs: "http://purl.org/rss/1.0/modules/content/",
		Channel: Channel{
			Title:       ctx.Config.Title,
			Link:        ctx.Config.BaseURL,
			Description: fmt.Sprintf("Updates for %s", ctx.Config.Title),
			Items:       rssItems,
		},
	}

	rssFile := filepath.Join(ctx.OutputDir, "rss.xml")
	f, err := os.Create(rssFile)
	if err != nil {
		log.Fatalf("Failed to create RSS file: %v", err)
	}
	defer f.Close()

	f.WriteString(xml.Header)
	enc := xml.NewEncoder(f)
	enc.Indent("", "  ")
	if err := enc.Encode(rss); err != nil {
		log.Fatalf("Failed to encode RSS: %v", err)
	}

	fmt.Println("Build complete.")
}

// buildHistory reconstructs the past versions of the plan file using git history.
// It iterates through unique dates in the git log, retrieves the file content for that date,
// and generates static pages for each day, as well as year and month index pages.
func buildHistory(ctx *PlanContext) ([]Item, error) {
	fmt.Println("Building history...")

	history, err := getGitHistory(ctx.PlanDir, ctx.PlanFile)
	if err != nil {
		return nil, err
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
	var rssItems []Item

	// Reuse renderer if config doesn't change per file
	r := render.New(&ctx.Config, ctx.Template, false)

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
			return nil, err
		}

		// Add to RSS
		relPath := fmt.Sprintf("/%s/%s/%s", year, month, day)
		link := ctx.Config.BaseURL + relPath

		// We need the body content. renderAndWrite does it but doesn't return it.
		// We'll just re-render body here.
		bodyBytes, err := r.RenderBody(content)
		if err == nil {
			item := Item{
				Title:       dateStr,
				Link:        link,
				Description: string(bodyBytes),
				Content:     string(bodyBytes),
				PubDate:     t.Format(time.RFC1123Z),
				Guid:        link,
			}
			rssItems = append(rssItems, item)
		}

		if tree[year] == nil {
			tree[year] = make(map[string][]dayEntry)
		}
		tree[year][month] = append(tree[year][month], dayEntry{
			DateStr: dateStr,
			Path:    relPath,
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
			return nil, err
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
				return nil, err
			}
		}
	}
	return rssItems, nil
}

func renderAndWrite(ctx *PlanContext, content []byte, modTime time.Time, outPath string) error {
	r := render.New(&ctx.Config, ctx.Template, ctx.LiveReload)

	body, err := r.RenderBody(content)
	if err != nil {
		return fmt.Errorf("rendering body: %w", err)
	}

	html, err := r.Compose(body, ctx.CreationTime, modTime)
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
	} else if runtime.GOOS == "windows" {
		cmd = exec.Command(editor, file)
	} else {
		// Use shell to execute the editor command string, handling args and quotes
		// e.g. EDITOR='emacsclient -t -a ""' -> sh -c 'emacsclient -t -a "" "/path/to/file"'
		// We quote the filename to handle spaces in paths
		fullCmd := fmt.Sprintf("%s %q", editor, file)
		cmd = exec.Command("sh", "-c", fullCmd)
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
