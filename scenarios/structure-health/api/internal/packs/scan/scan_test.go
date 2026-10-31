package scan

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBuildSkipsRuleDefinitionTree proves the conformance scan ignores the
// migrated rule-definition sources (which embed example-violation strings) so a
// scenario that ships rule packs does not flag itself. Files outside the
// rule-definition root are still scanned normally.
// [REQ:SH-PROF-003]
func TestBuildSkipsRuleDefinitionTree(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, root, "api/handlers/app.go", "package handlers\n")
	mustWrite(t, root, "api/internal/packs/configpack/hardcodedvalues/hardcodedvalues.go", "package hardcodedvalues // contains port 15000 example\n")
	mustWrite(t, root, "api/internal/packs/scan/scan.go", "package scan\n")

	ctx, err := Build("demo", root)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	for _, rel := range ctx.AllFiles {
		if isRuleDefinitionDir(filepath.Dir(rel)) || rel == "api/internal/packs" {
			t.Fatalf("rule-definition file leaked into scan: %s", rel)
		}
	}
	var sawHandler bool
	for _, rel := range ctx.AllFiles {
		if rel == "api/handlers/app.go" {
			sawHandler = true
		}
	}
	if !sawHandler {
		t.Fatalf("expected product file api/handlers/app.go in scan, got %v", ctx.AllFiles)
	}
	if got := len(ctx.FilesForTarget(TargetAPI)); got != 1 {
		t.Fatalf("expected exactly 1 api file (the handler), got %d: %v", got, ctx.FilesForTarget(TargetAPI))
	}
}

func TestIsRuleDefinitionDir(t *testing.T) {
	cases := map[string]bool{
		"api/internal/packs":                  true,
		"api/internal/packs/configpack/ports": true,
		"api/internal/packsuffix":             false,
		"api/internal":                        false,
		"api/internal/packers":                false,
	}
	for in, want := range cases {
		if got := isRuleDefinitionDir(in); got != want {
			t.Errorf("isRuleDefinitionDir(%q)=%v want %v", in, got, want)
		}
	}
}

func mustWrite(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
