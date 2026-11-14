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

// TestNoProductionImports walks every non-test .go file under cli/ and
// asserts that none imports `<module>/internal/testutil/...`.
//
// This mirrors the API-side guardrail at
// api/internal/testutil/no_prod_import_test.go. The contract: testutil
// can use `testing`, atomics, process-wide state, real sockets — none
// of which belong in a production binary. The walker enforces that.
//
// Failure modes the meta-test catches:
//   - Reflex copy-paste of a fake into a non-test file.
//   - A refactor that moves a helper into a non-test file without
//     noticing it now leaks into builds.
//
// If this test fires:
//   - ✅ Move the helper out of testutil into a non-test package.
//   - ❌ Don't add a `// nolint` — the production binary will then carry
//     the test-only dep on every build.
func TestNoProductionImports(t *testing.T) {
	module := readModuleName(t)
	prefix := module + "/internal/testutil"

	root := "../../"
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

// readModuleName returns the module path declared in cli/go.mod.
// The walker test file is at cli/internal/testutil/, so go.mod is two
// levels up. Reading dynamically (rather than hardcoding the literal
// `ai-gateway/cli`) means the test works both pre-substitution
// (against the template) and post-substitution (in any generated
// scenario, regardless of its ID).
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

// walk iterates every file under root, calling fn for each. The
// testutil/ subtree is skipped because its own intra-package references
// would self-flag the guardrail.
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
