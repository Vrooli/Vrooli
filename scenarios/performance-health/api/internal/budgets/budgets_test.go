package budgets

import (
	"context"
	"testing"
)

// [REQ:PH-BUDGET-002] Declared lcp/startup/component axes are flagged as
// continuously-gated (out-of-band), never synchronously; build+bundle axes are
// not — so a declared budget can't masquerade as synchronous protection.
func TestUngatedDeclaredAxesAndAdvisory(t *testing.T) {
	// Only synchronously-gateable axes => no advisory.
	if axes := UngatedDeclaredAxes(Budget{GoBuildMaxMs: 1, UIBuildMaxMs: 1, BundleMaxBytes: 1}); len(axes) != 0 {
		t.Fatalf("build+bundle axes are gated synchronously; expected none, got %v", axes)
	}
	b := Budget{LCPMaxMs: 2500, StartupMaxMs: 3000, ComponentCommitAvgMaxMs: 8, ComponentCommitMaxMs: 16}
	axes := UngatedDeclaredAxes(b)
	want := []string{"lcp", "startup", "component_commit_avg", "component_commit_max"}
	if len(axes) != len(want) {
		t.Fatalf("expected %v ungated axes, got %v", want, axes)
	}
	findings := AdvisoryFindings(b)
	if len(findings) != 1 || findings[0].Code != "PERF_BUDGET_AXIS_UNGATED" || findings[0].Severity != "info" {
		t.Fatalf("expected one INFO PERF_BUDGET_AXIS_UNGATED advisory, got %+v", findings)
	}
	if AdvisoryFindings(Budget{GoBuildMaxMs: 1}) != nil {
		t.Fatalf("no advisory expected when only synchronously-gated axes are declared")
	}
}

// [REQ:PH-BUDGET-001] Budgets set then get round-trip; an undeclared scenario
// reports declared=false with defaults.
func TestSetGetRoundTrip(t *testing.T) {
	svc := NewService(NewStore())
	if _, err := svc.Set(context.Background(), Budget{Scenario: "demo", GoBuildMaxMs: 90000}, false); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, declared, err := svc.Get(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !declared || got.GoBuildMaxMs != 90000 {
		t.Fatalf("unexpected budget: declared=%v %#v", declared, got)
	}
}

func TestGetUndeclared(t *testing.T) {
	svc := NewService(NewStore())
	_, declared, err := svc.Get(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if declared {
		t.Fatal("expected declared=false for an undeclared scenario")
	}
}

func TestSetDryRunDoesNotPersist(t *testing.T) {
	svc := NewService(NewStore())
	if _, err := svc.Set(context.Background(), Budget{Scenario: "demo", GoBuildMaxMs: 1}, true); err != nil {
		t.Fatalf("Set dry-run: %v", err)
	}
	if _, declared, _ := svc.Get(context.Background(), "demo"); declared {
		t.Fatal("dry-run should not persist")
	}
}

// [REQ:PH-BUDGET-001] Evaluate flags only axes whose measured value exceeds a
// positive budget; a zero measured value (not measured) never violates.
func TestEvaluateViolations(t *testing.T) {
	b := Budget{Scenario: "demo", GoBuildMaxMs: 90000, BundleMaxBytes: 500000}
	m := Measurement{GoBuildMs: 120000, BundleBytes: 400000}
	v := Evaluate(b, m)
	if len(v) != 1 || v[0].Axis != "go_build" {
		t.Fatalf("expected one go_build violation, got %#v", v)
	}
	if v[0].Measured != 120000 || v[0].Budget != 90000 {
		t.Fatalf("unexpected violation detail: %#v", v[0])
	}
}

func TestEvaluateNoBudgetNoViolation(t *testing.T) {
	b := Budget{Scenario: "demo"} // no budgets set
	v := Evaluate(b, Measurement{GoBuildMs: 999999})
	if len(v) != 0 {
		t.Fatalf("expected no violations when no budget set, got %#v", v)
	}
}

// [REQ:PH-BUDGET-001] Component-commit avg/max axes evaluate against the slowest
// component's readings.
func TestEvaluateComponentCommit(t *testing.T) {
	b := Budget{Scenario: "demo", ComponentCommitAvgMaxMs: 16, ComponentCommitMaxMs: 50}
	m := Measurement{ComponentCommitAvgMs: 24.3, ComponentCommitMaxMs: 40, SlowestComponent: "ProjectList"}
	v := Evaluate(b, m)
	if len(v) != 1 || v[0].Axis != "component_commit_avg" {
		t.Fatalf("expected one component_commit_avg violation, got %#v", v)
	}
	if v[0].Detail != "ProjectList" {
		t.Fatalf("expected slowest-component detail, got %q", v[0].Detail)
	}
}

// [REQ:PH-BUDGET-001] The ratchet rejects a loosening write but accepts a
// tightening one (and an unchanged one).
func TestRatchetTightenOnly(t *testing.T) {
	svc := NewService(NewStore())
	if _, err := svc.Set(context.Background(), Budget{Scenario: "demo", GoBuildMaxMs: 90000, Ratchet: true}, false); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// Loosening (higher max) is rejected.
	if _, err := svc.Set(context.Background(), Budget{Scenario: "demo", GoBuildMaxMs: 120000, Ratchet: true}, false); err == nil {
		t.Fatal("expected ratchet to reject a loosening write")
	}
	// Tightening (lower max) is accepted.
	if _, err := svc.Set(context.Background(), Budget{Scenario: "demo", GoBuildMaxMs: 80000, Ratchet: true}, false); err != nil {
		t.Fatalf("tightening should be accepted: %v", err)
	}
	got, _, _ := svc.Get(context.Background(), "demo")
	if got.GoBuildMaxMs != 80000 {
		t.Fatalf("expected tightened budget 80000, got %d", got.GoBuildMaxMs)
	}
}

func TestRatchetOffAllowsLoosening(t *testing.T) {
	svc := NewService(NewStore())
	if _, err := svc.Set(context.Background(), Budget{Scenario: "demo", GoBuildMaxMs: 90000}, false); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := svc.Set(context.Background(), Budget{Scenario: "demo", GoBuildMaxMs: 120000}, false); err != nil {
		t.Fatalf("without ratchet, loosening should be allowed: %v", err)
	}
}

// [REQ:PH-BUDGET-002] Check passes when no budget is declared.
func TestCheckPassesWithoutBudget(t *testing.T) {
	svc := NewService(NewStore())
	passed, violations, err := svc.Check(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !passed || len(violations) != 0 {
		t.Fatalf("expected pass with no budget, got passed=%v %#v", passed, violations)
	}
}

type fakeSource struct {
	m     Measurement
	found bool
}

func (f fakeSource) Latest(context.Context, string) (Measurement, bool, error) {
	return f.m, f.found, nil
}

// [REQ:PH-BUDGET-002] Check evaluates the latest measured sample against the
// declared budget and reports the breach.
func TestCheckFlagsViolationFromSource(t *testing.T) {
	store := NewStore()
	_, _ = store.Set(context.Background(), Budget{Scenario: "demo", GoBuildMaxMs: 90000}, false)
	svc := NewService(store, WithMeasurementSource(fakeSource{m: Measurement{GoBuildMs: 130000}, found: true}))
	passed, violations, err := svc.Check(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if passed || len(violations) != 1 || violations[0].Axis != "go_build" {
		t.Fatalf("expected one go_build breach, got passed=%v %#v", passed, violations)
	}
}

func TestCheckPassesWhenNoSampleYet(t *testing.T) {
	store := NewStore()
	_, _ = store.Set(context.Background(), Budget{Scenario: "demo", GoBuildMaxMs: 90000}, false)
	svc := NewService(store, WithMeasurementSource(fakeSource{found: false}))
	passed, violations, err := svc.Check(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !passed || len(violations) != 0 {
		t.Fatalf("expected pass with no measured sample, got passed=%v %#v", passed, violations)
	}
}
