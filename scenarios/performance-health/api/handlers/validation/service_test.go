package validation

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	"performance-health/internal/autofix"
	"performance-health/internal/budgets"
	"performance-health/internal/readiness"

	"github.com/vrooli/maturity-go/assessment"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/readiness"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

type fakeFacts struct{ facts readiness.Facts }

func (f fakeFacts) Describe(context.Context, string, string) (readiness.Facts, error) {
	return f.facts, nil
}

type fakeBudgetChecker struct {
	passed     bool
	violations []budgets.Violation
}

func (f fakeBudgetChecker) Check(context.Context, string) (bool, []budgets.Violation, error) {
	return f.passed, f.violations, nil
}

const budgetGateSpec = `{
  "provider": "performance-health",
  "phase": "performance",
  "version": "1.0.0",
  "levels": [
    {"id": "L0", "name": "exists", "description": "resolvable",
     "entry_criteria": ["a scenario is provided"], "exit_criteria": ["resolvable"]},
    {"id": "L1", "name": "within budget", "description": "no breach",
     "entry_criteria": ["resolvable"], "exit_criteria": ["within performance budget"]}
  ],
  "findings": {
    "PERF_BUDGET_BREACH_GO_BUILD": {
      "local_level_impact": "L1", "global_impact": "evolvability_gap",
      "dimension": "performance", "severity_default": "SEVERITY_ERROR", "clean_requirement": "required"
    }
  },
  "fallback": {
    "local_level_impact": "L1", "global_impact": "evolvability_gap",
    "dimension": "performance", "severity_default": "SEVERITY_ERROR", "clean_requirement": "required"
  }
}`

// [REQ:PH-BUDGET-002] A perf-budget breach folds into the validation assessment
// as an ERROR finding and drives the status to FAILED — the baseline-diff gate.
func TestBudgetBreachFailsValidation(t *testing.T) {
	spec, err := assessment.ParseSpec([]byte(budgetGateSpec))
	if err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	h := NewHandlerWithDeps(Deps{
		Readiness:    readiness.NewService(fakeFacts{facts: readiness.Facts{Scenario: "demo", Surfaces: []string{"ui"}, UIFramework: "react"}}),
		Autofix:      autofix.NewService(),
		MaturitySpec: spec,
		Budgets: fakeBudgetChecker{passed: false, violations: []budgets.Violation{
			{Axis: "go_build", Measured: 130000, Budget: 90000, Unit: "ms"},
		}},
	})
	resp, err := h.ValidateReadiness(context.Background(), connect.NewRequest(&readinessv1.ValidateReadinessRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ValidateReadiness: %v", err)
	}
	if resp.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		t.Fatalf("budget breach must fail validation, got %s", resp.Msg.GetStatus())
	}
}

// [REQ:PH-BUDGET-002] A within-budget run folds no budget findings and does not
// fail validation.
func TestBudgetWithinBudgetPassesValidation(t *testing.T) {
	spec, err := assessment.ParseSpec([]byte(budgetGateSpec))
	if err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	h := NewHandlerWithDeps(Deps{
		Readiness:    readiness.NewService(fakeFacts{facts: readiness.Facts{Scenario: "demo", Surfaces: []string{"ui"}, UIFramework: "react"}}),
		Autofix:      autofix.NewService(),
		MaturitySpec: spec,
		Budgets:      fakeBudgetChecker{passed: true},
	})
	resp, err := h.ValidateReadiness(context.Background(), connect.NewRequest(&readinessv1.ValidateReadinessRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ValidateReadiness: %v", err)
	}
	if resp.Msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		t.Fatal("a within-budget run must not fail validation")
	}
}

// [REQ:PH-VALIDATION-001] The native ReadinessService validates a scenario and
// reports its reachable tier.
func TestValidateReadinessReportsTier(t *testing.T) {
	h := NewHandlerWithDeps(Deps{
		Readiness: readiness.NewService(fakeFacts{facts: readiness.Facts{Scenario: "demo", Surfaces: []string{"ui"}, UIFramework: "react"}}),
		Autofix:   autofix.NewService(),
	})
	resp, err := h.ValidateReadiness(context.Background(), connect.NewRequest(&readinessv1.ValidateReadinessRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ValidateReadiness: %v", err)
	}
	if resp.Msg.GetTier() != readinessv1.CaptureTier_CAPTURE_TIER_1 {
		t.Fatalf("tier = %v, want Tier1", resp.Msg.GetTier())
	}
}

func TestValidateReadinessRequiresTarget(t *testing.T) {
	h := NewHandlerWithDeps(Deps{Readiness: readiness.NewService(fakeFacts{}), Autofix: autofix.NewService()})
	if _, err := h.ValidateReadiness(context.Background(), connect.NewRequest(&readinessv1.ValidateReadinessRequest{})); err == nil {
		t.Fatal("expected invalid-argument error for empty target")
	}
}

// [REQ:PH-TIER-003] Preview readiness fix returns no candidates for an empty
// tree (nothing mechanically fixable) without error.
func TestPreviewReadinessFix(t *testing.T) {
	h := NewHandlerWithDeps(Deps{
		Readiness: readiness.NewService(fakeFacts{}),
		Autofix:   autofix.NewService(),
		RepoRoot:  t.TempDir(),
	})
	resp, err := h.PreviewReadinessFix(context.Background(), connect.NewRequest(&readinessv1.ReadinessFixRequest{Path: t.TempDir()}))
	if err != nil {
		t.Fatalf("PreviewReadinessFix: %v", err)
	}
	if resp.Msg.GetApplied() {
		t.Fatal("preview must not report applied")
	}
}

// [REQ:PH-TIER-002][REQ:PH-TIER-003] End-to-end: a divergent react-vite fixture
// surfaces the four missing-infra findings + an autofixable_count of 4 through
// ValidateReadiness, and PreviewReadinessFix returns the four fix candidates.
func TestValidateAndPreviewSurfaceDivergentInfra(t *testing.T) {
	root := writeBareReactViteFixture(t)
	h := NewHandlerWithDeps(Deps{
		Readiness: readiness.NewService(fakeFacts{facts: readiness.Facts{
			Scenario: "bare", Surfaces: []string{"ui"}, UIFramework: "react-vite", RootPath: root,
		}}),
		Autofix: autofix.NewService(),
	})

	val, err := h.ValidateReadiness(context.Background(), connect.NewRequest(&readinessv1.ValidateReadinessRequest{Scenario: "bare", Path: root}))
	if err != nil {
		t.Fatalf("ValidateReadiness: %v", err)
	}
	if val.Msg.GetTier() != readinessv1.CaptureTier_CAPTURE_TIER_1 {
		t.Fatalf("tier = %v, want Tier1 reachable", val.Msg.GetTier())
	}
	if val.Msg.GetAutofixableCount() != 4 {
		t.Fatalf("autofixable_count = %d, want 4", val.Msg.GetAutofixableCount())
	}

	prev, err := h.PreviewReadinessFix(context.Background(), connect.NewRequest(&readinessv1.ReadinessFixRequest{Scenario: "bare", Path: root}))
	if err != nil {
		t.Fatalf("PreviewReadinessFix: %v", err)
	}
	if got := len(prev.Msg.GetCandidates()); got != 4 {
		t.Fatalf("preview candidates = %d, want 4", got)
	}
	if prev.Msg.GetApplied() {
		t.Fatal("preview must not report applied")
	}
}

func writeBareReactViteFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	ui := filepath.Join(root, "ui")
	if err := os.MkdirAll(filepath.Join(ui, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"ui/package.json":   `{"name":"bare","scripts":{"build":"vite build"},"dependencies":{"react":"^19.0.0"},"devDependencies":{"vite":"^6.0.0"}}` + "\n",
		"ui/vite.config.ts": "import { defineConfig } from \"vite\";\nexport default defineConfig(() => ({ plugins: [] }));\n",
		"ui/src/main.tsx":   "import App from \"./App\";\nrender(<App />);\n",
	}
	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(root, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
