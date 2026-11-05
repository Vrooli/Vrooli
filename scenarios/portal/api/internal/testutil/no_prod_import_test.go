package testutil_test

import (
	"bufio"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoProductionImports walks every non-test .go file under ../ and
// asserts that none imports `<module>/internal/testutil/...`.
//
// This is the load-bearing guarantee that lets the testutil package
// freely depend on `testing`, mutate process-wide state in fakes, and
// expose concurrency knobs only tests need — production code can't see
// any of it.
//
// Failure modes the meta-test catches:
//   - Someone imports testutil from production by reflex copy-paste.
//   - A future refactor moves a test helper into a non-test file
//     without realising it's now leaking into builds.
//
// The test is opinionated: any import whose path begins with
// `<module>/internal/testutil` from a non-`_test.go` file fails.
func TestNoProductionImports(t *testing.T) {
	module := readModuleName(t)
	prefix := module + "/internal/testutil"
	violations := collectProductionImportViolations(t, "../", prefix)

	if len(violations) > 0 {
		t.Errorf("production code must not import %s/...", prefix)
		for _, v := range violations {
			t.Errorf("  %s", v)
		}
	}
}

func collectProductionImportViolations(t *testing.T, root, forbiddenPrefix string) []string {
	t.Helper()
	fset := token.NewFileSet()
	violations := []string{}
	walk(t, root, func(path string) {
		rel, ok := productionSourceRel(root, path)
		if !ok {
			return
		}
		for _, imported := range importPaths(t, fset, path) {
			if strings.HasPrefix(imported, forbiddenPrefix) {
				violations = append(violations, rel+" imports "+imported)
			}
		}
	})
	return violations
}

func productionSourceRel(root, path string) (string, bool) {
	if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
		return "", false
	}
	rel := strings.TrimPrefix(path, root)
	if strings.Contains(rel, "/vendor/") {
		return "", false
	}
	// mocks/ holds test-only fakes that lack the _test.go suffix (so sibling
	// _test.go files in other packages can import them); generated/ holds
	// temporal-model output (replay.go, runtime.go) that legitimately bridges
	// production transition functions into the modeltest harness. Both are test
	// scaffolding by directory-shape, exempt from the testutil-import rule.
	if pathHasDir(rel, "mocks") || pathHasDir(rel, "generated") {
		return "", false
	}
	return rel, true
}

func importPaths(t *testing.T, fset *token.FileSet, path string) []string {
	t.Helper()
	file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		t.Errorf("parse %s: %v", path, err)
		return nil
	}
	paths := make([]string, 0, len(file.Imports))
	for _, imp := range file.Imports {
		paths = append(paths, strings.Trim(imp.Path.Value, `"`))
	}
	return paths
}

// readModuleName returns the module path declared in the api root's
// go.mod. The walker is at ../../go.mod relative to this test file
// (api/internal/testutil/ is two levels deep).
func readModuleName(t *testing.T) string {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatalf("open go.mod: %v", err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(rest)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan go.mod: %v", err)
	}
	t.Fatal("module directive not found in go.mod")
	return ""
}

// walk iterates every .go file under root, calling fn for each. The
// testutil subtree itself is skipped because it legitimately holds
// internal references that would otherwise self-flag.
func walk(t *testing.T, root string, fn func(path string)) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "testutil" {
				return filepath.SkipDir
			}
			return nil
		}
		fn(path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
}

// pathHasDir reports whether any directory segment of rel (excluding the file
// name itself) equals name. It expresses the directory-shape exemptions for
// test-scaffolding trees that lack a `_test.go` suffix (mocks/, generated/):
// production code never imports them, and the TestNoProductionImports walker
// catches any production file that reaches for testutil directly.
func pathHasDir(rel, name string) bool {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, p := range parts[:len(parts)-1] {
		if p == name {
			return true
		}
	}
	return false
}
