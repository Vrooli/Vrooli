package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

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
