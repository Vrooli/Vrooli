package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductionCodeDoesNotImportTestutil(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(data), "/internal/testutil") {
			t.Errorf("production file imports testutil: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
