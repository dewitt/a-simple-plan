package render

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"time"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/dewitt/a-simple-plan/internal/config"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/renderer/html"
)

//go:embed template.html
var defaultTemplateHTML string

// Renderer handles the conversion of markdown to HTML with dynamic headers.
type Renderer struct {
	mdRenderer   goldmark.Markdown
	loc          *time.Location
	config       *config.Config
	templateHTML string
}

// New creates a new Renderer.
func New(cfg *config.Config, customTemplate string) *Renderer {
	// Initialize markdown renderer once
	md := goldmark.New(
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
		goldmark.WithExtensions(
			highlighting.NewHighlighting(
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(true),
				),
			),
		),
	)

	tz := "America/Los_Angeles"
	if cfg != nil && cfg.Timezone != "" {
		tz = cfg.Timezone
	}

	// Pre-load location
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}

	tmpl := defaultTemplateHTML
	if customTemplate != "" {
		tmpl = customTemplate
	}

	return &Renderer{
		mdRenderer:   md,
		loc:          loc,
		config:       cfg,
		templateHTML: tmpl,
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
	// Format created time
	onSince := created.In(r.loc).Format("Mon Jan _2 15:04 (MST)")
	modTimeUnix := fmt.Sprintf("%d", updated.Unix())

	// Inject dynamic values into the template.
	outputStr := r.templateHTML
	outputStr = strings.ReplaceAll(outputStr, "{{onSince}}", onSince)
	outputStr = strings.ReplaceAll(outputStr, "{{modTimeUnix}}", modTimeUnix)

	if r.config != nil {
		outputStr = strings.ReplaceAll(outputStr, "{{username}}", r.config.Username)
		outputStr = strings.ReplaceAll(outputStr, "{{fullname}}", r.config.FullName)
		outputStr = strings.ReplaceAll(outputStr, "{{directory}}", r.config.Directory)
		outputStr = strings.ReplaceAll(outputStr, "{{shell}}", r.config.Shell)
		outputStr = strings.ReplaceAll(outputStr, "{{title}}", r.config.Title)
	}

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
