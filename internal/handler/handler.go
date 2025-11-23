package handler

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dewitt/dewitt-blog/internal/render"
)

// New creates a new HTTP handler for serving the blog post.
// It pre-renders the markdown content for performance.
func New(postFile string) http.HandlerFunc {
	// Initialize renderer
	r := render.New()

	// 1. Read the file (Once at startup)
	content, err := os.ReadFile(postFile)
	if err != nil {
		// If file doesn't exist at startup, we might want to panic or serve a "not found" page forever.
		// For a containerized app where the file is baked in, this is a fatal error.
		log.Printf("Warning: could not read %s: %v", postFile, err)
		// We'll proceed with empty content, or maybe a simple error message
		content = []byte("# No plan found\n")
	}

	// 2. Get file info for timestamps (Once at startup)
	info, err := os.Stat(postFile)
	var createTime, modTime time.Time
	if err == nil {
		modTime = info.ModTime()
		createTime = modTime
	} else {
		now := time.Now()
		modTime = now
		createTime = now
	}

	// 3. Pre-render the markdown body (Once at startup)
	bodyHTML, err := r.RenderBody(content)
	if err != nil {
		log.Printf("Error rendering markdown: %v", err)
		bodyHTML = []byte("<p>Error rendering content</p>")
	}

	// Return the handler
	return func(w http.ResponseWriter, req *http.Request) {
		// Only allow GET requests to the root
		if req.URL.Path != "/" {
			http.NotFound(w, req)
			return
		}
		if req.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check If-Modified-Since
		// Since modTime is fixed at startup, this is efficient.
		if t, err := time.Parse(http.TimeFormat, req.Header.Get("If-Modified-Since")); err == nil && modTime.Before(t.Add(1*time.Second)) {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		// Set caching headers
		w.Header().Set("Cache-Control", "public, max-age=60")
		w.Header().Set("Last-Modified", modTime.UTC().Format(http.TimeFormat))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		// 4. Compose the final HTML (Fast string/byte concatenation)
		// Note: 'modTime' is used for the idle calculation inside Compose.
		// Since modTime is fixed (startup time), the idle time will increase as the server runs.
		html, err := r.Compose(bodyHTML, createTime, modTime)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Error composing HTML: %v", err)
			return
		}

		w.Write(html)
	}
}
