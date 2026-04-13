package testkitgo

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigratedCoreTestsDoNotImportLegacyWrappers(t *testing.T) {
	root := ProjectRoot(t)
	legacyImportPrefix := "github.com/vrooli/vrooli/internal/test"
	forbidden := []string{
		legacyImportPrefix + "fixture",
		legacyImportPrefix + "util",
	}
	scanRoots := []string{
		filepath.Join(root, "internal"),
		filepath.Join(root, "cmd"),
		filepath.Join(root, "packages"),
	}

	for _, scanRoot := range scanRoots {
		err := filepath.WalkDir(scanRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if path == filepath.Join(root, "packages", "testkit-go", "adoption_hygiene_test.go") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			contents := string(data)
			for _, forbiddenImport := range forbidden {
				if strings.Contains(contents, forbiddenImport) {
					t.Errorf("%s imports legacy wrapper %q", path, forbiddenImport)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", scanRoot, err)
		}
	}
}
