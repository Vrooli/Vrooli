package testutil_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoProductionImports walks every non-test .go file under
// ../../internal/ and asserts that none imports
// `workspace-sandbox/internal/testutil/...`.
//
// This is the load-bearing guarantee that lets the testutil package
// freely depend on `testing`, mutate process-wide state in fakes,
// and exercise concurrency knobs only tests need: production code
// can't see any of it.
//
// Failure modes the meta-test catches:
//   - Someone imports testutil from production by reflex copy-paste.
//   - A future refactor moves a test helper into a non-test file
//     without realizing it's now leaking into builds.
//
// The test is opinionated: any import whose path begins with
// `workspace-sandbox/internal/testutil` from a non-`_test.go` file
// fails.
func TestNoProductionImports(t *testing.T) {
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
		// Skip vendored, generated, or build-artifact directories
		// (none today, but inexpensive to guard against).
		rel := strings.TrimPrefix(path, root)
		if strings.Contains(rel, "/vendor/") {
			return
		}

		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("parse %s: %v", path, err)
			return
		}
		for _, imp := range file.Imports {
			// imp.Path.Value is quoted ("foo/bar"). Strip quotes
			// before comparing.
			ip := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(ip, "workspace-sandbox/internal/testutil") {
				violations = append(violations, rel+" imports "+ip)
			}
		}
	})

	if len(violations) > 0 {
		t.Errorf("production code must not import internal/testutil/...")
		for _, v := range violations {
			t.Errorf("  %s", v)
		}
	}
}

// walk iterates every .go file under root, calling fn for each.
// Hand-rolled rather than filepath.Walk so the test stays focused
// on what we actually need (extension filter happens in fn).
func walk(t *testing.T, root string, fn func(path string)) {
	t.Helper()
	entries := readDir(t, root)
	for _, e := range entries {
		full := filepath.Join(root, e.name)
		if e.isDir {
			// Recurse, but skip the testutil subtree itself: it's
			// the only place that legitimately holds testutil
			// imports (in its own package files), and the test
			// only inspects non-test files anyway.
			if filepath.Base(full) == "testutil" {
				continue
			}
			walk(t, full, fn)
			continue
		}
		fn(full)
	}
}
