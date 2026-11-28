package rules

import (
	"os"
	"path/filepath"
	"testing"

	"quality-health/internal/surfaces"
)

func TestTypecheckPlannerCoverageUsesUnitHealthPlanMap(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"scripts":{"type-check":"tsc --noEmit"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	rule, ok := ByID(RuleTypecheckPlannerCoverage)
	if !ok {
		t.Fatal("typecheck planner coverage rule is not registered")
	}
	base := EvalContext{Surface: surfaces.Surface{ID: "ui", Language: "typescript", RootPath: root}}
	if findings := rule.Evaluate(base); len(findings) != 0 {
		t.Fatalf("legacy script-only context should remain compatible: %+v", findings)
	}
	if findings := rule.Evaluate(EvalContext{Surface: base.Surface, TypecheckPlans: map[string]bool{"other": true}}); len(findings) != 1 {
		t.Fatalf("missing unit-health plan findings=%+v", findings)
	}
	if findings := rule.Evaluate(EvalContext{Surface: base.Surface, TypecheckPlans: map[string]bool{"ui": true}}); len(findings) != 0 {
		t.Fatalf("covered unit-health plan findings=%+v", findings)
	}
}

func TestTypecheckPlannerCoverageAcceptsPythonMypyDeclaration(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("[tool.mypy]\nfiles = ['.']\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rule, _ := ByID(RuleTypecheckPlannerCoverage)
	findings := rule.Evaluate(EvalContext{
		Surface:        surfaces.Surface{ID: "api", Language: "python", RootPath: root},
		TypecheckPlans: map[string]bool{"api": true},
	})
	if len(findings) != 0 {
		t.Fatalf("mypy-covered Python surface findings=%+v", findings)
	}
}
