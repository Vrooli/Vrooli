//go:build linux

package maintenance

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestReadProcessOpenFilesResolvesThisProcessDescriptors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout-held.jsonl")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	targets, err := readProcessOpenFiles(os.Getpid())
	if err != nil || !slices.Contains(targets, path) {
		t.Fatalf("open files missing held transcript %s: err=%v", path, err)
	}
}
