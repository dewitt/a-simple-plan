package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
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
		fmt.Fprintf(os.Stderr, "  edit     - Open plan file in default editor\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}

	// Parse flags before commands
	// Note: standard flag package expects flags before non-flag arguments.
	// e.g., `plan -f myplan.md preview`
	// To support `plan preview -f myplan.md`, we'd need to parse manually or use a sub-command library.
	// For simplicity/standard Go, let's assume `plan [flags] command`.
	// However, a common UX expectation is `plan command [flags]`.
	// Let's try to support `plan command [flags]` by parsing flags *after* the command if possible,
	// or better, defining subcommands.

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	// Simple subcommand parsing
	cmd := os.Args[1]

	// Create a FlagSet for each command if we want command-specific flags,
	// or just use global flags if they apply to all.
	// Since the user asked for a flag to override the post being previewed OR published,
	// it applies to both.

	// Let's handle the case where the user might type `plan preview -f foo.md`
	// We can shift os.Args if the first arg is a known command.

	switch cmd {
	case "preview":
		// Parse flags from args[2:]
		if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
			os.Exit(1)
		}
		preview(planFile)
	case "build":
		// Parse flags from args[2:]
		if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
			os.Exit(1)
		}
		build(planFile)
	case "save":
		// Parse flags from args[2:]
		if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
			os.Exit(1)
		}
		save(planFile)
	case "publish":
		// Parse flags from args[2:]
		if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
			os.Exit(1)
		}
		publish(planFile)
	case "revert":
		// Parse flags from args[2:]
		if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
			os.Exit(1)
		}
		revert(planFile)
	case "edit":
		// Parse flags from args[2:]
		if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
			os.Exit(1)
		}
		edit(planFile)
	case "-h", "--help":
		flag.Usage()
	default:
		// Check if the first arg is actually a flag (starts with -)
		if cmd[0] == '-' {
			// It's a flag, presumably before the command? Or no command provided?
			// If they ran `plan -f foo.md preview`, flag.Parse() would handle it
			// if we called it globally. But we want to support `plan preview`.

			// Let's stick to the robust subcommand pattern.
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
	port := "8081" // Use a different default port for preview to avoid conflict

	// Ensure the static site is built before previewing
	build(file)

	fs := http.FileServer(http.Dir("public"))
	http.Handle("/", fs)

	fmt.Printf("Starting preview server for static content at http://localhost:%s/index.html\n", port)

	// Launch browser in a goroutine
	go func() {
		// Give the server a moment to start
		time.Sleep(500 * time.Millisecond)
		openBrowser("http://localhost:" + port + "/index.html")
	}()

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func build(file string) {
	fmt.Printf("Building %s to public/index.html...\n", file)

	// 1. Read file
	content, err := os.ReadFile(file)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	// 2. Get stats
	info, err := os.Stat(file)
	if err != nil {
		log.Fatalf("Failed to stat file: %v", err)
	}
	modTime := info.ModTime()

	// 3. Render
	r := render.New()
	body, err := r.RenderBody(content)
	if err != nil {
		log.Fatalf("Failed to render body: %v", err)
	}

	html, err := r.Compose(body, modTime, modTime)
	if err != nil {
		log.Fatalf("Failed to compose HTML: %v", err)
	}

	// 4. Write
	if err := os.MkdirAll("public", 0755); err != nil {
		log.Fatalf("Failed to create public dir: %v", err)
	}
	if err := os.WriteFile("public/index.html", html, 0644); err != nil {
		log.Fatalf("Failed to write HTML: %v", err)
	}

	fmt.Println("Build complete.")
}

func save(file string) {
	fmt.Printf("Saving %s...\n", file)

	if err := runCmd("git", "add", file); err != nil {
		log.Fatalf("Failed to add file: %v", err)
	}

	// Commit. If no changes, this might fail, which is okay-ish, but let's handle it.
	if err := runCmd("git", "commit", "-m", "Update plan"); err != nil {
		fmt.Println("Nothing to commit or commit failed (maybe no changes?).")
	} else {
		fmt.Println("Changes committed.")
	}
}

func publish(file string) {
	// 1. Save (Commit)
	save(file)

	// 2. Push to git
	fmt.Println("Pushing to origin...")
	if err := runCmd("git", "push"); err != nil {
		log.Fatalf("Failed to push: %v", err)
	}

	fmt.Println("Successfully pushed to origin. Cloudflare Pages deployment should trigger shortly.")
}

func revert(file string) {
	fmt.Printf("Reverting %s...\n", file)
	// Use git checkout to discard changes to the file
	if err := runCmd("git", "checkout", file); err != nil {
		log.Fatalf("Failed to revert: %v", err)
	}
	fmt.Println("Local changes discarded.")
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
			editor = "open -t" // 'open -t' opens with default text editor on macOS
		default: // Linux and others
			editor = "vi"
		}
	}

	fmt.Printf("Opening %s with %s...\n", file, editor)

	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" && editor == "open -t" {
		cmd = exec.Command("open", "-t", file)
	} else if runtime.GOOS != "windows" {
		// Use sh -c to handle complex EDITOR strings (e.g., "emacsclient -t -a \"\"")
		// We pass the editor command string as a script, and the file as $1
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
