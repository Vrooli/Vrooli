package repocontractcheck

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunPassesAgainstLiveRepo(t *testing.T) {
	report, err := Run(repoRoot(t))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !report.Success {
		t.Fatalf("report.Success = false, checks = %+v", report.Checks)
	}
	if len(report.Checks) == 0 {
		t.Fatal("expected checks to be populated")
	}
}

func TestRunRequiresRoot(t *testing.T) {
	if _, err := Run(""); err == nil {
		t.Fatal("expected error for empty root")
	}
}

func TestRunFailsWhenAgentGuidanceMissing(t *testing.T) {
	root := newValidationFixtureRepo(t)
	writeFixtureFile(t, root, "AGENTS.md", "# AGENTS.md\n")

	report, err := Run(root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	if !hasFailedCheck(report, "adoption_rules_alignment") {
		t.Fatalf("expected adoption_rules_alignment failure, got %+v", report.Checks)
	}
}

func TestRunFailsWhenUnexpectedAdoptionViolationAppears(t *testing.T) {
	root := newValidationFixtureRepo(t)
	writeFixtureFile(t, root, "scenarios/example/api/main.go", "package main\n\nimport \"path/filepath\"\n\nfunc getVrooliRoot() string {\n\thome := \"/tmp\"\n\treturn filepath.Join(home, \"Vrooli\")\n}\n")

	report, err := Run(root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	if !hasFailedCheck(report, "adoption_rules_alignment") {
		t.Fatalf("expected adoption_rules_alignment failure, got %+v", report.Checks)
	}
}

func TestRunFailsWhenGitMarkerRootProbeAppears(t *testing.T) {
	root := newValidationFixtureRepo(t)
	writeFixtureFile(t, root, "scenarios/example/api/main.go", "package main\n\nimport (\n\t\"os\"\n\t\"path/filepath\"\n)\n\nfunc resolveRepoRoot(path string) bool {\n\t_, err := os.Stat(filepath.Join(path, \".git\"))\n\treturn err == nil\n}\n")

	report, err := Run(root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	if !hasFailedCheck(report, "adoption_rules_alignment") {
		t.Fatalf("expected adoption_rules_alignment failure, got %+v", report.Checks)
	}
}

func TestRunFailsWhenPNPMWorkspaceRootProbeAppears(t *testing.T) {
	root := newValidationFixtureRepo(t)
	writeFixtureFile(t, root, "scenarios/example/api/main.go", "package main\n\nimport (\n\t\"os\"\n\t\"path/filepath\"\n)\n\nfunc resolveRepoRoot(path string) bool {\n\t_, err := os.Stat(filepath.Join(path, \"pnpm-workspace.yaml\"))\n\treturn err == nil\n}\n")

	report, err := Run(root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	if !hasFailedCheck(report, "adoption_rules_alignment") {
		t.Fatalf("expected adoption_rules_alignment failure, got %+v", report.Checks)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func hasFailedCheck(report Report, name string) bool {
	for _, check := range report.Checks {
		if check.Name == name && !check.Passed {
			return true
		}
	}
	return false
}

func newValidationFixtureRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	sourceRoot := repoRoot(t)
	for _, rel := range []string{
		filepath.Join(".vrooli", "repo-contract.json"),
		filepath.Join(".vrooli", "repo-contract-adoption-exceptions.json"),
		filepath.Join("docs", "repo-contract.md"),
		filepath.Join("docs", "CONTRIBUTING.md"),
		"AGENTS.md",
		filepath.Join("scenarios", "prompt-manager", "store", "skills", "packs", "core", "cross-platform-readiness", "SKILL.md"),
	} {
		data, err := os.ReadFile(filepath.Join(sourceRoot, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		writeFixtureFile(t, root, rel, string(data))
	}

	writeFixtureFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	for _, dir := range []string{"packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	writeFixtureFile(t, root, filepath.Join("scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha"}}`)
	writeFixtureFile(t, root, filepath.Join("resources", "redis", "resource.json"), `{"name":"redis"}`)

	return root
}

func writeFixtureFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(content, "\r\n", "\n")), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
