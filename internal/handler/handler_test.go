package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestHandler(t *testing.T) {
	// Create a temporary post file
	tmpfile, err := os.CreateTemp("", "post.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	content := []byte("# Test Header\nTest content")
	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Create request
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// Call handler
	h := New(tmpfile.Name())
	h(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}

	if !strings.Contains(string(body), "<h1>Test Header</h1>") {
		t.Errorf("Response body does not contain rendered HTML")
	}
}

func TestHandlerNotFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/other", nil)
	w := httptest.NewRecorder()

	h := New("nonexistent_file.md")
	h(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status NotFound for wrong path, got %v", resp.Status)
	}
}

func TestHandlerMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("POST", "/", nil)
	w := httptest.NewRecorder()

	h := New("nonexistent_file.md")
	h(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status MethodNotAllowed for POST, got %v", resp.Status)
	}
}
