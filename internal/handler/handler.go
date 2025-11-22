package handler

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dewitt/dewitt-blog/internal/render"
)

// New creates a new HTTP handler for serving the blog post.
func New(postFile string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only allow GET requests to the root
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		start := time.Now()
		content, err := os.ReadFile(postFile)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "No post found. Please create "+postFile, http.StatusNotFound)
				return
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Error reading file: %v", err)
			return
		}

		// Get file info for timestamps
		info, err := os.Stat(postFile)
		var createTime, modTime time.Time
		if err == nil {
			modTime = info.ModTime()
			createTime = modTime // Fallback if birthtime isn't available or easy to get portably
		} else {
			// If stat fails (unlikely if ReadFile worked), use current time
			now := time.Now()
			modTime = now
			createTime = now
		}

		html, err := render.Render(content, createTime, modTime)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Error rendering markdown: %v", err)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(html)
		log.Printf("Served request in %v", time.Since(start))
	}
}
