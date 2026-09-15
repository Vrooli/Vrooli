package validation

import (
	"os"
	"path/filepath"
	"testing"

	"unit-health/internal/discovery"
)

func TestPlanNormalizationAndFilesystemLanguageBranches(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct{ value, file, want string }{{"golang", "", "go"}, {"ts", "", "typescript"}, {"js", "", "javascript"}, {"node", "", "node"}, {"py", "", "python"}, {"sh", "", "bash"}, {"rs", "", "rust"}, {"pwsh", "", "powershell"}, {"", "go.mod", "go"}} {
		path := root
		if tc.file != "" {
			path = t.TempDir()
			if err := os.WriteFile(filepath.Join(path, tc.file), []byte(""), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		if got := normalizeLanguage(tc.value, path); got != tc.want {
			t.Fatalf("normalizeLanguage(%q) = %q, want %q", tc.value, got, tc.want)
		}
	}
	for _, tc := range []struct{ file, want string }{{"requirements.txt", "python"}, {"run.sh", "bash"}, {"Cargo.toml", "rust"}, {"script.ps1", "powershell"}} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, tc.file), []byte(""), 0o600); err != nil {
			t.Fatal(err)
		}
		if got := normalizeLanguage("unknown", dir); got != tc.want {
			t.Fatalf("fallback %s = %q", tc.file, got)
		}
	}
	if got := normalizeLanguage("", t.TempDir()); got != "unknown" {
		t.Fatalf("empty fallback = %q", got)
	}
}

func TestPlanHandlesTargetKindsEmptySurfacesAndFastMode(t *testing.T) {
	root := t.TempDir()
	for _, kind := range []string{"package", "control-plane"} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		inv := normalizeTargetInventory(discovery.Inventory{TargetKind: kind, RootPath: dir, Surfaces: []discovery.Surface{{ID: "old", RootPath: dir}}})
		if len(inv.Surfaces) != 1 || inv.Surfaces[0].Kind != kind {
			t.Fatalf("normalized %s = %+v", kind, inv)
		}
	}
	if inv := normalizeTargetInventory(discovery.Inventory{TargetKind: "scenario"}); len(inv.Surfaces) != 0 {
		t.Fatal("scenario inventory changed")
	}
	_, _, plan, findings := buildPlan("demo", discovery.Inventory{TargetKind: "scenario"}, "now")
	if plan.Notes == "" || len(findings) == 0 {
		t.Fatalf("empty plan = %+v, %+v", plan, findings)
	}
	unsupported := unsupportedTargetFinding("demo", "resource", "now")
	if unsupported.Code != codeUnsupportedTargetKind || unsupported.Remediation == "" {
		t.Fatalf("unsupported finding = %+v", unsupported)
	}
	workspace := Workspace{ID: "api", Language: "go", Framework: "go test", Status: "ready", TestCommand: "go test", CoverageCommand: "go test -cover", TestExecutable: "go", CoverageExecutable: "go", TestArtifacts: []Artifact{{Label: "coverage"}}, RootPath: root, TimeoutSeconds: 0}
	if got := buildExecutionPlanForMode([]Workspace{workspace}, true); len(got.Commands) != 1 || len(got.Commands[0].Artifacts) != 0 || got.Notes == "" {
		t.Fatalf("fast plan = %+v", got)
	}
	if got := buildExecutionPlanForMode([]Workspace{{ID: "empty"}}, false); len(got.Commands) != 0 || got.Notes == "" {
		t.Fatalf("empty commands plan = %+v", got)
	}
}
