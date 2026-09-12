package orchestration_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestNoLegacyDeclarationLayoutReaders enforces the no-fallback cutover at the
// source level: the retired .vrooli/agent-profiles/ and .vrooli/agent-workflows/
// directory names may appear only in the designated rejection code that flags
// them, never as an active reader path. If a new reference appears outside the
// allowlist, a reader for the old layout has crept back in and the test fails.
func TestNoLegacyDeclarationLayoutReaders(t *testing.T) {
	apiRoot := agentManagerAPIRoot(t)

	// Files that legitimately name the old directories to reject them.
	allow := map[string]bool{
		filepath.Join("internal", "orchestration", "declaration_reconcile.go"): true,
		filepath.Join("internal", "conformance", "service.go"):                 true,
	}
	legacyLiterals := []string{".vrooli/agent-profiles", ".vrooli/agent-workflows"}

	err := filepath.WalkDir(apiRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if name := entry.Name(); name == "vendor" || name == "node_modules" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, relErr := filepath.Rel(apiRoot, path)
		if relErr != nil {
			return relErr
		}
		if allow[rel] {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := string(data)
		for _, literal := range legacyLiterals {
			if strings.Contains(text, literal) {
				t.Errorf("%s references retired declaration directory %q; the old layout is not readable — only the rejection code may name it", rel, literal)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk agent-manager api tree: %v", err)
	}
}

func agentManagerAPIRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	// file = <api>/internal/orchestration/legacy_layout_guard_test.go
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
