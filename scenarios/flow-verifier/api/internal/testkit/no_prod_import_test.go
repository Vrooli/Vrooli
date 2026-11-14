package testkit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTestkitIsOnlyImportedByTests(t *testing.T) {
	root, err := moduleRoot()
	if err != nil {
		t.Fatal(err)
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), `"flow-verifier/internal/testkit"`) {
			t.Fatalf("production file imports internal/testkit: %s", filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func moduleRoot() (string, error) {
	start, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := filepath.Clean(start); ; dir = filepath.Dir(dir) {
		data, readErr := os.ReadFile(filepath.Join(dir, "go.mod"))
		if readErr == nil && strings.Contains(string(data), "module flow-verifier\n") {
			if info, statErr := os.Stat(filepath.Join(dir, "internal", "testkit")); statErr == nil && info.IsDir() {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("flow-verifier module root not found from %s", start)
		}
	}
}
