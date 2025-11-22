package render

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRender(t *testing.T) {
	input := []byte("# Hello World\nThis is a test.")
	now := time.Now()
	output, err := Render(input, now, now)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !bytes.Contains(output, []byte("<h1>Hello World</h1>")) {
		t.Error("Output does not contain rendered HTML header")
	}
	if !bytes.Contains(output, []byte("<p>This is a test.</p>")) {
		t.Error("Output does not contain rendered HTML paragraph")
	}
	if !strings.Contains(string(output), "<!DOCTYPE html>") {
		t.Error("Output does not contain HTML5 doctype")
	}
}
