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

	r := New(nil, "", false) // Use default config and template for testing
	body, err := r.RenderBody(input)
	if err != nil {
		t.Fatalf("RenderBody failed: %v", err)
	}

	output, err := r.Compose(body, now, now)
	if err != nil {
		t.Fatalf("Compose failed: %v", err)
	}

	if !strings.Contains(string(output), "Hello World") {
		t.Error("Output does not contain \"Hello World\" text")
	}
	if !bytes.Contains(output, []byte("<p>This is a test.</p>")) {
		t.Error("Output does not contain rendered HTML paragraph")
	}
	if !strings.Contains(string(output), "<!DOCTYPE html>") {
		t.Error("Output does not contain HTML5 doctype")
	}
}

func TestRender_AutoLink(t *testing.T) {
	input := []byte("Check out https://example.com for more info.")
	r := New(nil, "", false)
	body, err := r.RenderBody(input)
	if err != nil {
		t.Fatalf("RenderBody failed: %v", err)
	}

	expected := `<a href="https://example.com">https://example.com</a>`
	if !strings.Contains(string(body), expected) {
		t.Errorf("Expected auto-linked URL, got: %s", string(body))
	}
}
