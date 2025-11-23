package render

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"time"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

//go:embed template.html
var templateHTML string

// Renderer handles the conversion of markdown to HTML with dynamic headers.
type Renderer struct {
	mdRenderer goldmark.Markdown
	loc        *time.Location
}

// New creates a new Renderer.
func New() *Renderer {
	// Initialize markdown renderer once
	md := goldmark.New(
		goldmark.WithExtensions(
			highlighting.NewHighlighting(
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(true),
				),
			),
		),
	)

	// Pre-load location
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		loc = time.UTC
	}

	return &Renderer{
		mdRenderer: md,
		loc:        loc,
	}
}

// RenderBody converts markdown content to an HTML fragment.
func (r *Renderer) RenderBody(md []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := r.mdRenderer.Convert(md, &buf); err != nil {
		return nil, fmt.Errorf("failed to convert markdown: %w", err)
	}
	return buf.Bytes(), nil
}

// Compose combines the pre-rendered HTML body with dynamic header information.
func (r *Renderer) Compose(bodyHTML []byte, created, updated time.Time) ([]byte, error) {
	// Calculate idle time
	idle := time.Since(updated)
	idleStr := fmt.Sprintf("%d:%02d", int(idle.Hours()), int(idle.Minutes())%60)

	// Format created time
	onSince := created.In(r.loc).Format("Mon Jan _2 15:04 (MST)")

	// Inject dynamic values into the template.
	outputStr := templateHTML
	outputStr = strings.Replace(outputStr, "{{onSince}}", onSince, 1)
	outputStr = strings.Replace(outputStr, "{{idleStr}}", idleStr, 1)

	// Inject Content
	parts := strings.Split(outputStr, "{{content}}")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid template: missing {{content}} marker")
	}

	// Pre-allocate buffer
	totalLen := len(parts[0]) + len(bodyHTML) + len(parts[1])
	finalBuf := bytes.NewBuffer(make([]byte, 0, totalLen))

	finalBuf.WriteString(parts[0])
	finalBuf.Write(bodyHTML)
	finalBuf.WriteString(parts[1])

	return finalBuf.Bytes(), nil
}

// Render converts markdown content to a full HTML page.
// It is kept for backward compatibility but using New() + RenderBody + Compose is preferred for performance.
func Render(md []byte, created, updated time.Time) ([]byte, error) {
	r := New()
	body, err := r.RenderBody(md)
	if err != nil {
		return nil, err
	}
	return r.Compose(body, created, updated)
}