package testutil

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestStartHTTPServer(t *testing.T) {
	server := StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ready")
	}))
	response, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("GET test server: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read test server response: %v", err)
	}
	if response.StatusCode != http.StatusOK || string(body) != "ready" {
		t.Fatalf("response = %d %q, want 200 %q", response.StatusCode, body, "ready")
	}
}

func TestCopyFilesCopiesNestedSourcesAndReturnsDestinations(t *testing.T) {
	sourceRoot := t.TempDir()
	targetRoot := t.TempDir()
	WriteFiles(t, sourceRoot, map[string][]byte{
		"api/owner.go":                []byte("package owner\n"),
		"docs/internal/contract.json": []byte("{}\n"),
	})

	paths := []string{"api/owner.go", "docs/internal/contract.json"}
	destinations := CopyFiles(t, sourceRoot, targetRoot, paths)
	if len(destinations) != len(paths) {
		t.Fatalf("destination count = %d, want %d", len(destinations), len(paths))
	}

	for _, relative := range paths {
		destination, ok := destinations[relative]
		if !ok {
			t.Fatalf("missing destination for %s", relative)
		}
		wantPath := filepath.Join(targetRoot, filepath.FromSlash(relative))
		if destination != wantPath {
			t.Errorf("destination for %s = %q, want %q", relative, destination, wantPath)
		}

		got, err := os.ReadFile(destination)
		if err != nil {
			t.Fatalf("read copied fixture %s: %v", relative, err)
		}
		want, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatalf("read source fixture %s: %v", relative, err)
		}
		if string(got) != string(want) {
			t.Errorf("copied fixture %s = %q, want %q", relative, got, want)
		}
	}
}
