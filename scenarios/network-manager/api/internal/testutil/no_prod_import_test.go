package testutil_test

import (
	"go/parser"
	"go/token"
	"io/fs"
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
	const prefix = "network-manager/internal/testutil"

	root := "../"
	violations := []string{}

	walk(t, root, func(path string) {
		rel := strings.TrimPrefix(path, root)
		if !isProductionGoFile(rel) {
			return
		}
		violations = append(violations, importViolations(t, path, rel, prefix)...)
	})

	reportImportViolations(t, prefix, violations)
}

func isProductionGoFile(rel string) bool {
	if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
		return false
	}
	if strings.Contains(rel, "/vendor/") {
		return false
	}
	// mocks/ holds test-only fakes that lack the _test.go suffix; generated/
	// holds temporal-model output that legitimately bridges production
	// transition functions into the modeltest harness.
	return !pathHasDir(rel, "mocks") && !pathHasDir(rel, "generated")
}

func importViolations(t *testing.T, path, rel, prefix string) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Errorf("parse %s: %v", path, err)
		return nil
	}
	violations := []string{}
	for _, imp := range file.Imports {
		ip := strings.Trim(imp.Path.Value, `"`)
		if strings.HasPrefix(ip, prefix) {
			violations = append(violations, rel+" imports "+ip)
		}
	}
	return violations
}

func reportImportViolations(t *testing.T, prefix string, violations []string) {
	t.Helper()
	if len(violations) == 0 {
		return
	}
	t.Errorf("production code must not import %s/...", prefix)
	for _, v := range violations {
		t.Errorf("  %s", v)
	}
}

// walk iterates every .go file under root, calling fn for each. The
// testutil subtree itself is skipped because it legitimately holds
// internal references that would otherwise self-flag.
func walk(t *testing.T, root string, fn func(path string)) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			t.Errorf("walk %s: %v", path, err)
			return nil
		}
		if d.IsDir() {
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
