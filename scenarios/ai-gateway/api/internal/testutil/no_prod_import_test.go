package testutil_test

import (
	"go/parser"
	"go/token"
	"io/fs"
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

	root := "../"
	fset := token.NewFileSet()
	violations := []string{}

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel := strings.TrimPrefix(path, root)
		if skipTreeEntry(rel, entry) {
			return filepath.SkipDir
		}
		if entry.IsDir() || !isProductionGoFile(path) {
			return nil
		}
		// mocks/ holds test-only fakes that lack the _test.go suffix (so sibling
		// _test.go files in other packages can import them); generated/ holds
		// temporal-model output (replay.go, runtime.go) that legitimately bridges
		// production transition functions into the modeltest harness. Both are test
		// scaffolding by directory-shape, exempt from the testutil-import rule.
		if pathHasDir(rel, "mocks") || pathHasDir(rel, "generated") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("parse %s: %v", path, err)
			return nil
		}
		for _, imp := range file.Imports {
			ip := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(ip, prefix) {
				violations = append(violations, rel+" imports "+ip)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	requireNoViolations(t, prefix, violations)
}

// readModuleName returns the module path declared in the api root's
// go.mod. The walker is at ../../go.mod relative to this test file
// (api/internal/testutil/ is two levels deep).
func readModuleName(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(rest)
		}
	}
	t.Fatal("module directive not found in go.mod")
	return ""
}

func requireNoViolations(t *testing.T, prefix string, violations []string) {
	t.Helper()
	if len(violations) == 0 {
		return
	}
	t.Errorf("production code must not import %s/...", prefix)
	for _, v := range violations {
		t.Errorf("  %s", v)
	}
}

func isProductionGoFile(path string) bool {
	return strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
}

func skipTreeEntry(rel string, entry fs.DirEntry) bool {
	return entry.IsDir() && (entry.Name() == "testutil" || entry.Name() == "vendor" || rel == ".git")
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
