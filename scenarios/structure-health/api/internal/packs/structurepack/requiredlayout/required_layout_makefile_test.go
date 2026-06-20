package requiredlayout

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRequiredLayoutAcceptsCanonicalMakefile(t *testing.T) {
	root := t.TempDir()
	writeRequiredLayoutFixture(t, root, canonicalScenarioMakefile())

	violations, err := Check(requiredLayoutPayload(t), root, "demo")
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %d: %+v", len(violations), violations)
	}
}

func TestRequiredLayoutRejectsNonCanonicalMakefile(t *testing.T) {
	root := t.TempDir()
	writeRequiredLayoutFixture(t, root, ".PHONY: help\nhelp:\n\t@echo bad\n")

	violations, err := Check(requiredLayoutPayload(t), root, "demo")
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %+v", len(violations), violations)
	}
	if got := violations[0].Message; got != "Makefile must provide the standard scenario lifecycle wrapper targets" {
		t.Fatalf("unexpected violation message: %q", got)
	}
}

func TestRequiredLayoutAcceptsExpandedGovernanceMakefile(t *testing.T) {
	root := t.TempDir()
	makefile, err := os.ReadFile(filepath.Join(repoRootForRequiredLayoutTest(t), "scenarios", "agent-manager", "Makefile"))
	if err != nil {
		t.Fatalf("read expanded Makefile: %v", err)
	}
	writeRequiredLayoutFixture(t, root, string(makefile))

	violations, err := Check(requiredLayoutPayload(t), root, "demo")
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %d: %+v", len(violations), violations)
	}
}

func TestRequiredLayoutAcceptsCanonicalMakefileWithCustomTargets(t *testing.T) {
	root := t.TempDir()
	makefile := canonicalScenarioMakefile() + `

.PHONY: fmt lint check-local

fmt:
	@pnpm fmt

lint:
	@pnpm lint

check-local:
	@pnpm lint && pnpm type-check
`
	writeRequiredLayoutFixture(t, root, makefile)

	violations, err := Check(requiredLayoutPayload(t), root, "demo")
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %d: %+v", len(violations), violations)
	}
}

func TestRequiredLayoutRejectsCanonicalTargetRecipeDrift(t *testing.T) {
	root := t.TempDir()
	makefile := strings.Replace(
		canonicalScenarioMakefile(),
		`	@vrooli scenario logs "$(SCENARIO_NAME)"`,
		`	@vrooli scenario logs "$(SCENARIO_NAME)" --tail 10`,
		1,
	)
	writeRequiredLayoutFixture(t, root, makefile)

	violations, err := Check(requiredLayoutPayload(t), root, "demo")
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %+v", len(violations), violations)
	}
}

func TestRequiredLayoutAcceptsLegacyLifecycleWrapper(t *testing.T) {
	root := t.TempDir()
	writeRequiredLayoutFixture(t, root, canonicalScenarioMakefileFallback())

	violations, err := Check(requiredLayoutPayload(t), root, "demo")
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %d: %+v", len(violations), violations)
	}
}

func writeRequiredLayoutFixture(t *testing.T, root, makefile string) {
	t.Helper()
	mustWriteFile(t, filepath.Join(root, ".vrooli", "service.json"), "{}\n")
	mustWriteFile(t, filepath.Join(root, "README.md"), "# Demo\n")
	mustWriteFile(t, filepath.Join(root, "PRD.md"), "# Demo PRD\n")
	mustWriteFile(t, filepath.Join(root, "Makefile"), makefile)
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func requiredLayoutPayload(t *testing.T) string {
	t.Helper()
	payload := struct {
		Scenario string   `json:"scenario"`
		Files    []string `json:"files"`
	}{
		Scenario: "demo",
		Files: []string{
			".vrooli/service.json",
			"README.md",
			"PRD.md",
			"Makefile",
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal payload: %v", err)
	}
	return string(data)
}

func repoRootForRequiredLayoutTest(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// .../scenarios/structure-health/api/internal/packs/structurepack/requiredlayout/<file>
	// repo root is 7 directories up from this file's directory.
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "..", "..", ".."))
}
