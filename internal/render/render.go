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

// Render converts markdown content to a full HTML page.
func Render(md []byte, created, updated time.Time) ([]byte, error) {
	var buf bytes.Buffer

	mdRenderer := goldmark.New(
		goldmark.WithExtensions(
			highlighting.NewHighlighting(
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(true),
				),
			),
		),
	)

	if err := mdRenderer.Convert(md, &buf); err != nil {
		return nil, fmt.Errorf("failed to convert markdown: %w", err)
	}

	// Calculate idle time
	idle := time.Since(updated)
	idleStr := fmt.Sprintf("%d:%02d", int(idle.Hours()), int(idle.Minutes())%60)

	// Format created time: Sat Nov 22 06:33 (PST)
	onSince := created.Format("Mon Jan _2 15:04 (MST)")

	// Inject dynamic values into the template.
	// First, format the header variables.
	// Note: We use a simple string replace for the content body to avoid
	// allocating massive strings via Sprintf if the blog post is large.
	
	// 1. Inject Header Info
	// The template has %s placeholders for onSince and idleStr in the finger header.
	// However, because we moved the content placeholder {{content}} to the template
	// and that template might contain % characters (in CSS), using Sprintf on the 
	// whole template is risky. 
	// Instead, let's replace specific tokens for safety and performance.
	
	// We will assume the template.html uses specific tokens or we just formatted the
	// header part carefully. But wait, the template.html I just wrote uses %s.
	// Let's switch the strategy to use simple string replacement for EVERYTHING 
	// to be absolutely safe against CSS % signs and other Sprintf quirks.
	
	// Let's re-read the template.html I wrote. It has:
	// On since %s on ttys000,       idle %s
	// {{content}}
	
	// Using Sprintf on the whole file is dangerous because of CSS %.
	// So I will replace the %s manually or update the template to use tokens.
	// I'll update the template to use {{onSince}} and {{idleStr}} in the next step 
	// to be cleaner, but for now let's just do string replacement on the current template.

	outputStr := templateHTML
	outputStr = strings.Replace(outputStr, "{{onSince}}", onSince, 1)
	outputStr = strings.Replace(outputStr, "{{idleStr}}", idleStr, 1)
	
	// 2. Inject Content
	// We do this last.
	// Using bytes.Buffer to construct the final output to avoid large string copies.
	
	parts := strings.Split(outputStr, "{{content}}")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid template: missing {{content}} marker")
	}

	// Pre-allocate buffer
	totalLen := len(parts[0]) + buf.Len() + len(parts[1])
	finalBuf := bytes.NewBuffer(make([]byte, 0, totalLen))
	
	finalBuf.WriteString(parts[0])
	finalBuf.Write(buf.Bytes())
	finalBuf.WriteString(parts[1])

	return finalBuf.Bytes(), nil
}