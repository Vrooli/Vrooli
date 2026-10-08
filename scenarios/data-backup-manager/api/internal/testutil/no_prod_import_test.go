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
		if isInMocksDir(rel) {
			return
		}
		if isInGeneratedDir(rel) {
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
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("readdir %s: %v", root, err)
	}
	for _, e := range entries {
		full := filepath.Join(root, e.Name())
		if e.IsDir() {
			if filepath.Base(full) == "testutil" {
				continue
			}
			walk(t, full, fn)
			continue
		}
		fn(full)
	}
}

// isInMocksDir reports whether path lives inside a `mocks/` directory.
// Per-domain mocks/ folders hold test-only fakes that must lack the
// _test.go suffix (so sibling _test.go files in other packages can
// import them), but they're never linked into production binaries:
// production code never imports `<dom>/mocks`, and the
// TestNoProductionImports walker catches any production file that
// reaches for testutil directly. mocks files themselves are exempt
// from the testutil-import rule because they ARE test scaffolding,
// just sharing-shape rather than _test.go-shape.
func isInMocksDir(rel string) bool {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, p := range parts[:len(parts)-1] {
		if p == "mocks" {
			return true
		}
	}
	return false
}

// isInGeneratedDir reports whether path lives inside a `generated/`
// directory. Code emitted by the temporal-model generator (replay.go,
// runtime.go, etc.) lives under `<flow>/generated/` and legitimately
// imports the modeltest test-only helpers — it IS the bridge between
// production transition functions and the formal-model test harness,
// so it must reach into testutil. Generated files lack `_test.go`
// suffixes by convention, so this directory-shape check is the cleanest
// way to express the exemption.
func isInGeneratedDir(rel string) bool {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, p := range parts[:len(parts)-1] {
		if p == "generated" {
			return true
		}
	}
	return false
}
