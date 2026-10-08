package hostinventory

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestSingleDisplayManagerVocabulary prevents the scenario compatibility
// layer from growing a second, subtly different display-manager list.
func TestSingleDisplayManagerVocabulary(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	const declaration = "DisplayManagerNames = []string{"
	count := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
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
		count += strings.Count(string(data), declaration)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("found %d display-manager vocabulary declarations; want exactly one", count)
	}
	for _, line := range strings.Split(string(mustReadFile(t, filepath.Join(filepath.Dir(file), "types.go"))), "\n") {
		if strings.Contains(line, declaration) && strings.Contains(line, "xrdp") {
			t.Fatal("xrdp must not be classified as a display manager")
		}
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
