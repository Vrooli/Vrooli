package budgets

import (
	"context"
	"strings"
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

// TestCLSBudgetGatesOnFractions: the CLS axis must trip on a fractional breach
// and report the real numbers. The int64 Measured/Budget pair rounds 0.03 to 0,
// so a renderer keying off it alone would print "measured=0 budget_max=0"; the
// *Value fields carry the magnitude.
func TestCLSBudgetGatesOnFractions(t *testing.T) {
	budget := Budget{Scenario: "demo", CLSMax: 0.1}

	// Within budget: no violation.
	if v := Evaluate(budget, Measurement{CLS: 0.05}); len(v) != 0 {
		t.Fatalf("0.05 under a 0.1 budget must pass, got %+v", v)
	}
	// Not measured: zero is never a violation on a max axis.
	if v := Evaluate(budget, Measurement{CLS: 0}); len(v) != 0 {
		t.Fatalf("an unmeasured CLS must not trip, got %+v", v)
	}

	got := Evaluate(budget, Measurement{CLS: 0.25})
	if len(got) != 1 {
		t.Fatalf("0.25 over a 0.1 budget must trip exactly once, got %+v", got)
	}
	v := got[0]
	if v.Axis != "cls" {
		t.Errorf("Axis = %q, want cls", v.Axis)
	}
	if v.MeasuredValue != 0.25 || v.BudgetValue != 0.1 {
		t.Errorf("precise pair = (%v, %v), want (0.25, 0.1)", v.MeasuredValue, v.BudgetValue)
	}
	if v.Unit != "" {
		t.Errorf("Unit = %q, want empty — CLS is unitless, and renderers key on that", v.Unit)
	}
	if v.Mode != "max" {
		t.Errorf("Mode = %q, want max", v.Mode)
	}
}

// TestCLSRatchetsTightenOnly: once declared, a CLS budget may only tighten,
// matching every other axis.
func TestCLSRatchetsTightenOnly(t *testing.T) {
	existing := Budget{Scenario: "demo", CLSMax: 0.1, Ratchet: true}
	if err := enforceRatchet(existing, Budget{Scenario: "demo", CLSMax: 0.05}); err != nil {
		t.Errorf("tightening 0.1 -> 0.05 must be allowed: %v", err)
	}
	err := enforceRatchet(existing, Budget{Scenario: "demo", CLSMax: 0.3})
	if err == nil {
		t.Fatal("loosening 0.1 -> 0.3 must be rejected under ratchet")
	}
	if !strings.Contains(err.Error(), "cls") {
		t.Errorf("ratchet error must name the axis, got %v", err)
	}
	// The message must report the threshold it is protecting. %.1f rounded a
	// sub-0.1 budget to "0.0", so the operator saw "was 0.0, requested 0.3".
	if !strings.Contains(err.Error(), "was 0.1") {
		t.Errorf("ratchet message must state the real prior value, got %v", err)
	}
}

// TestRatchetMessageKeepsFractionalPrecision: every fractional axis, not just
// cls, must report its real threshold. dropped_frame_rate is also a 0-1 ratio.
func TestRatchetMessageKeepsFractionalPrecision(t *testing.T) {
	existing := Budget{Scenario: "demo", DroppedFrameRateMax: 0.02, Ratchet: true}
	err := enforceRatchet(existing, Budget{Scenario: "demo", DroppedFrameRateMax: 0.5})
	if err == nil {
		t.Fatal("loosening 0.02 -> 0.5 must be rejected under ratchet")
	}
	if !strings.Contains(err.Error(), "was 0.02") {
		t.Errorf("ratchet message rounded away the threshold, got %v", err)
	}
}

// TestCLSCountsAsADeclaredBudget: a CLS-only budget is a real budget, so IsSet
// reports true and the axis is listed as continuously-measured rather than
// freshly gated.
func TestCLSCountsAsADeclaredBudget(t *testing.T) {
	b := Budget{Scenario: "demo", CLSMax: 0.1}
	if !b.IsSet() {
		t.Error("a CLS-only budget must count as declared")
	}
	if !contains(UngatedDeclaredAxes(b), "cls") {
		t.Errorf("cls is measured out-of-band by capture/sweep, so it must be reported ungated: %v", UngatedDeclaredAxes(b))
	}
	f := FlowBudget{CLSMax: 0.1}
	if !f.IsSet() {
		t.Error("a CLS-only flow budget must count as declared")
	}
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// TestNavigationBudgetAxes: each navigation phase gates independently, so a
// regression can be attributed to a phase (backend, parse, deferred scripts,
// total asset weight) rather than just "the page got slower".
func TestNavigationBudgetAxes(t *testing.T) {
	budget := Budget{
		Scenario:              "demo",
		ResponseEndMaxMs:      50,
		DOMInteractiveMaxMs:   100,
		DOMContentLoadedMaxMs: 200,
		LoadEventEndMaxMs:     400,
	}
	// A load comfortably inside every phase ceiling.
	if v := Evaluate(budget, Measurement{
		ResponseEndMs: 4, DOMInteractiveMs: 14, DOMContentLoadedMs: 101, LoadEventEndMs: 202,
	}); len(v) != 0 {
		t.Fatalf("a load inside every ceiling must pass, got %+v", v)
	}
	// Not measured is never a violation.
	if v := Evaluate(budget, Measurement{}); len(v) != 0 {
		t.Fatalf("an unmeasured navigation must not trip, got %+v", v)
	}

	// A backend regression trips only response_end.
	got := Evaluate(budget, Measurement{
		ResponseEndMs: 900, DOMInteractiveMs: 14, DOMContentLoadedMs: 101, LoadEventEndMs: 202,
	})
	if len(got) != 1 || got[0].Axis != "response_end" {
		t.Fatalf("a slow response must trip response_end alone, got %+v", got)
	}
	if got[0].Measured != 900 || got[0].Budget != 50 {
		t.Errorf("violation numbers = (%d, %d), want (900, 50)", got[0].Measured, got[0].Budget)
	}

	// A heavy page trips the later phases without implicating the backend.
	got = Evaluate(budget, Measurement{
		ResponseEndMs: 4, DOMInteractiveMs: 14, DOMContentLoadedMs: 900, LoadEventEndMs: 1800,
	})
	axes := map[string]bool{}
	for _, v := range got {
		axes[v.Axis] = true
	}
	if !axes["dom_content_loaded"] || !axes["load_event_end"] {
		t.Errorf("a heavy page must trip the later phases, got %v", axes)
	}
	if axes["response_end"] {
		t.Errorf("a fast backend must not be implicated by a heavy page, got %v", axes)
	}
}

// TestNavigationAxesAreDeclaredAndUngated: navigation budgets count as declared
// and are reported as continuously measured, since capture/sweep produces them
// out-of-band rather than the synchronous gate measuring them fresh.
func TestNavigationAxesAreDeclaredAndUngated(t *testing.T) {
	b := Budget{Scenario: "demo", LoadEventEndMaxMs: 400}
	if !b.IsSet() {
		t.Error("a navigation-only budget must count as declared")
	}
	if !contains(UngatedDeclaredAxes(b), "load_event_end") {
		t.Errorf("navigation axes are measured out-of-band, so must report ungated: %v", UngatedDeclaredAxes(b))
	}
}

// TestNavigationRatchetsTightenOnly: navigation ceilings ratchet like every
// other axis.
func TestNavigationRatchetsTightenOnly(t *testing.T) {
	existing := Budget{Scenario: "demo", LoadEventEndMaxMs: 400, Ratchet: true}
	if err := enforceRatchet(existing, Budget{Scenario: "demo", LoadEventEndMaxMs: 300}); err != nil {
		t.Errorf("tightening 400 -> 300 must be allowed: %v", err)
	}
	err := enforceRatchet(existing, Budget{Scenario: "demo", LoadEventEndMaxMs: 900})
	if err == nil {
		t.Fatal("loosening 400 -> 900 must be rejected")
	}
	if !strings.Contains(err.Error(), "load_event_end") {
		t.Errorf("ratchet error must name the axis, got %v", err)
	}
}

// TestFlowBudgetHasNoNavigationAxes documents the modelling decision: a targeted
// interaction flow reuses the same page load, so a per-flow navigation ceiling
// would gate the identical measurement twice. If navigation is ever added to
// FlowBudget this test should be deleted deliberately, not silently.
func TestFlowBudgetHasNoNavigationAxes(t *testing.T) {
	if (FlowBudget{}).IsSet() {
		t.Fatal("an empty flow budget must not be declared")
	}
	// A flow budget carrying only navigation-shaped intent has nothing to set,
	// which is the point: the axes do not exist on FlowBudget.
	f := FlowBudget{LCPMaxMs: 100}
	if !f.IsSet() {
		t.Error("a flow budget with LCP must still be declared")
	}
}
