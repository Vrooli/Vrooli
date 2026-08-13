package rules

import (
	"os"
	"path/filepath"
	"testing"

	"quality-health/internal/surfaces"
)

func TestScenarioPrivilegeBoundaryReportsExactRuntimeInvocation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	if err := os.WriteFile(path, []byte("package main\nimport \"os/exec\"\nfunc main() { exec.Command(\"sudo\", \"systemctl\", \"restart\", \"foo\").Run() }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings := evalScenarioPrivilegeBoundary(EvalContext{Surface: surfaces.Surface{RootPath: root, Kind: "api", Language: "go"}})
	if len(findings) != 1 {
		t.Fatalf("findings = %d, want 1: %+v", len(findings), findings)
	}
	if findings[0].FilePath != path || findings[0].Evidence != "line 3: sudo systemctl restart foo" {
		t.Fatalf("finding = %+v", findings[0])
	}
}

func TestScenarioPrivilegeBoundaryAllowsDeclaredGrant(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("repo marker\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	guard := filepath.Join(root, "internal", "safeguards", "example", "handler.go")
	if err := os.MkdirAll(filepath.Dir(guard), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(guard, []byte("const systemctlPath = \"/usr/bin/systemctl\"\nvar _ = \"%s restart cloudflared\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "scenario.go")
	if err := os.WriteFile(source, []byte("package main\nimport \"os/exec\"\nfunc main() { exec.Command(\"sudo\", \"systemctl\", \"restart\", \"cloudflared\").Run() }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings := evalScenarioPrivilegeBoundary(EvalContext{Surface: surfaces.Surface{RootPath: root, Kind: "api", Language: "go"}})
	if len(findings) != 0 {
		t.Fatalf("findings = %+v, want none", findings)
	}
}

func TestScenarioPrivilegeBoundaryExemptsSanctionedAndTestSources(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"internal/setup/setup.go", "internal/hostreqkit/run.go", "internal/safeguards/x/handler.go", "internal/privilegebroker/policy.go", "testdata/fixture.go", "ignored_test.go"} {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("package fixture\nimport \"os/exec\"\nfunc f() { exec.Command(\"sudo\", \"systemctl\", \"restart\", \"foo\").Run() }\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if findings := evalScenarioPrivilegeBoundary(EvalContext{Surface: surfaces.Surface{RootPath: root, Kind: "api", Language: "go"}}); len(findings) != 0 {
		t.Fatalf("findings = %+v, want none", findings)
	}
}
