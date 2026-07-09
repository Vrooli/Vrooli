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
// `template-manager/cli`) means the test works both pre-substitution
// (against the template) and post-substitution (in any generated
// scenario, regardless of its ID).
func readModuleName(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(rest)
		}
	}
	t.Fatal("module directive not found in go.mod")
	return ""
}

// walk iterates every file under root, calling fn for each. The
// testutil/ subtree is skipped because its own intra-package references
// would self-flag the guardrail.
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
