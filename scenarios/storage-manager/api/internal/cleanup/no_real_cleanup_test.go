package cleanup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductionCodeDoesNotInstantiateCleanupSideEffectsDirectly(t *testing.T) {
	t.Parallel()

	root := filepath.Clean(filepath.Join(".."))
	forbidden := []string{
		"exec.Command(",
		"os.Remove(",
		"os.RemoveAll(",
		"docker system prune",
		"docker image prune",
		"journalctl --vacuum",
		"apt clean",
		"pnpm store prune",
		"npm cache clean",
		"pip cache purge",
	}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "testutil" || entry.Name() == "mocks" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(data)
		for _, needle := range forbidden {
			if strings.Contains(text, needle) {
				t.Errorf("%s contains forbidden cleanup side-effect constructor %q", path, needle)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk production code: %v", err)
	}
}
