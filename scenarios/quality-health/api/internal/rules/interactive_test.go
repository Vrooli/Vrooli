package rules

import (
	"os"
	"path/filepath"
	"testing"

	"quality-health/internal/surfaces"
)

func TestInteractiveBoundaryRejectsTTYAndMaskedReads(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "prompt.go")
	if err := os.WriteFile(path, []byte("package main\nimport \"golang.org/x/term\"\nfunc prompt() { term.ReadPassword(0) }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings := evalScenarioInteractiveBoundary(EvalContext{Surface: surfaces.Surface{RootPath: root, Language: "go"}})
	if len(findings) != 1 || findings[0].FilePath != path {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestInteractiveBoundaryAllowsPipedInput(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "pipe.go"), []byte("package main\nimport (\"io\"; \"os\")\nfunc read() []byte { value, _ := io.ReadAll(os.Stdin); return value }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if findings := evalScenarioInteractiveBoundary(EvalContext{Surface: surfaces.Surface{RootPath: root, Language: "go"}}); len(findings) != 0 {
		t.Fatalf("findings = %+v, want none for piped input", findings)
	}
}

func TestInteractiveBoundaryExemptsSanctionedTrees(t *testing.T) {
	for _, path := range []string{"/repo/scenarios/vrooli-onboarding/cli/register.go", "/repo/scenarios/vrooli-bridge/cli/domains/onboard/password.go", "/repo/scenarios/vrooli-bridge/cli/domains/auth/handlers.go"} {
		if !interactiveSurfaceExempt(path) {
			t.Fatalf("%s was not recognized as sanctioned", path)
		}
	}
}

func TestInteractiveBoundaryDoesNotExemptUnrelatedBridgeCLI(t *testing.T) {
	if interactiveSurfaceExempt("/repo/scenarios/vrooli-bridge/cli/domains/status/register.go") {
		t.Fatal("unrelated Bridge CLI domains must remain covered by the boundary rule")
	}
}

func TestInteractiveBoundaryExemptsSanctionedFilesWithinBridgeCLIRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "scenarios", "vrooli-bridge", "cli")
	for _, item := range []struct {
		rel  string
		body string
	}{
		{rel: "domains/auth/handlers.go", body: "package auth\nfunc login() { term.ReadPassword(0) }\n"},
		{rel: "domains/onboard/password.go", body: "package onboard\nfunc prompt() { term.ReadPassword(0) }\n"},
		{rel: "domains/status/register.go", body: "package status\nfunc prompt() { term.ReadPassword(0) }\n"},
	} {
		path := filepath.Join(root, item.rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(item.body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	findings := evalScenarioInteractiveBoundary(EvalContext{Surface: surfaces.Surface{RootPath: root, Language: "go"}})
	if len(findings) != 1 || findings[0].FilePath != filepath.Join(root, "domains/status/register.go") {
		t.Fatalf("findings = %+v, want only the unrelated Bridge domain", findings)
	}
}
