package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductionCodeDoesNotImportTestutil(t *testing.T) {
	t.Parallel()
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("locate CLI test workspace: %v", err)
	}
	cliRoot := filepath.Clean(filepath.Join(workingDir, "..", ".."))
	err = filepath.WalkDir(cliRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(contents), "/internal/testutil") {
			t.Errorf("production source imports test utility: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
