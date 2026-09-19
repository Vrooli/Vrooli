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

	walk(t, root, func(path string) {
		if !strings.HasSuffix(path, ".go") {
			return
		}
		if strings.HasSuffix(path, "_test.go") {
			return
		}
		rel := strings.TrimPrefix(path, root)
		if strings.Contains(rel, "/vendor/") {
			return
		}
		// mocks/ holds test-only fakes that lack the _test.go suffix so sibling
		// _test.go files in other packages can import them.
		if pathHasDir(rel, "mocks") {
			return
		}

		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("parse %s: %v", path, err)
			return
		}
		for _, imp := range file.Imports {
			ip := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(ip, prefix) {
				violations = append(violations, rel+" imports "+ip)
			}
		}
	})

	if len(violations) > 0 {
		t.Errorf("production code must not import %s/...", prefix)
		for _, v := range violations {
			t.Errorf("  %s", v)
		}
	}
}

// readModuleName returns the module path declared in the api root's
// go.mod. The walker is at ../../go.mod relative to this test file
// (api/internal/testutil/ is two levels deep).
func readModuleName(t *testing.T) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	for _, line := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(rest)
		}
	}
	t.Fatal("module directive not found in go.mod")
	return ""
}

// walk iterates every .go file under root, calling fn for each. The
// testutil subtree itself is skipped because it legitimately holds
// internal references that would otherwise self-flag.
func walk(t *testing.T, root string, fn func(path string)) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if filepath.Base(path) == "testutil" {
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
// name itself) equals name. It expresses the directory-shape exemption for
// test-scaffolding trees that lack a `_test.go` suffix.
func pathHasDir(rel, name string) bool {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, p := range parts[:len(parts)-1] {
		if p == name {
			return true
		}
	}
	return false
}
