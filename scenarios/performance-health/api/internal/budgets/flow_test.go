package budgets

import (
	"context"
	"path/filepath"
	"testing"
)

// fakeFlowSource implements both MeasurementSource and FlowMeasurementSource so
// CheckFlow can resolve a flow-tagged measurement.
type fakeFlowSource struct {
	scenarioM Measurement
	flowM     map[string]Measurement
}

func (f fakeFlowSource) Latest(context.Context, string) (Measurement, bool, error) {
	return f.scenarioM, true, nil
}

func (f fakeFlowSource) LatestFlow(_ context.Context, _, flow string) (Measurement, bool, error) {
	m, ok := f.flowM[flow]
	return m, ok, nil
}

// [REQ:PH-BUDGET-003] A per-flow budget gates a flow-tagged sample independently
// of the scenario aggregate; an over-budget flow axis is a violation.
func TestCheckFlowFlagsViolation(t *testing.T) {
	store := NewStore()
	if _, err := store.Set(context.Background(), Budget{
		Scenario: "demo",
		Flows:    map[string]FlowBudget{"scroll-list": {LCPMaxMs: 2000}},
	}, false); err != nil {
		t.Fatalf("Set: %v", err)
	}
	svc := NewService(store, WithMeasurementSource(fakeFlowSource{
		flowM: map[string]Measurement{"scroll-list": {LCPMs: 3200}},
	}))
	passed, violations, err := svc.CheckFlow(context.Background(), "demo", "scroll-list")
	if err != nil {
		t.Fatalf("CheckFlow: %v", err)
	}
	if passed || len(violations) != 1 || violations[0].Axis != "lcp" {
		t.Fatalf("expected one lcp violation, got passed=%v %#v", passed, violations)
	}
}

// [REQ:PH-BUDGET-003] Interaction flow budgets can fail on frame health even
// when React commit timing is inside budget.
func TestCheckFlowFlagsFrameHealthViolation(t *testing.T) {
	store := NewStore()
	if _, err := store.Set(context.Background(), Budget{
		Scenario: "demo",
		Flows: map[string]FlowBudget{"graph-pan": {
			ComponentCommitAvgMaxMs: 20,
			DrawnFPSMin:             45,
			DroppedFrameRateMax:     0.20,
			InputEventCountMin:      10,
		}},
	}, false); err != nil {
		t.Fatalf("Set: %v", err)
	}
	svc := NewService(store, WithMeasurementSource(fakeFlowSource{
		flowM: map[string]Measurement{"graph-pan": {
			ComponentCommitAvgMs: 8,
			DrawnFPS:             22,
			DroppedFrameRate:     0.58,
			InputEventCount:      30,
		}},
	}))
	passed, violations, err := svc.CheckFlow(context.Background(), "demo", "graph-pan")
	if err != nil {
		t.Fatalf("CheckFlow: %v", err)
	}
	if passed {
		t.Fatal("expected frame-health budget breach")
	}
	if !hasAxis(violations, "drawn_fps") || !hasAxis(violations, "dropped_frame_rate") {
		t.Fatalf("expected drawn_fps and dropped_frame_rate violations, got %#v", violations)
	}
}

// [REQ:PH-BUDGET-003] Gesture workflows fail closed when a captured sample lacks
// frame/input evidence; load-only flows opt out explicitly.
func TestCheckFlowMissingInteractionEvidenceFailsUnlessLoadOnly(t *testing.T) {
	store := NewStore()
	if _, err := store.Set(context.Background(), Budget{
		Scenario: "demo",
		Flows: map[string]FlowBudget{
			"graph-pan":  {DrawnFPSMin: 45, DroppedFrameRateMax: 0.20, InputEventCountMin: 10},
			"graph-load": {DrawnFPSMin: 45, DroppedFrameRateMax: 0.20, InputEventCountMin: 10, LoadOnly: true},
		},
	}, false); err != nil {
		t.Fatalf("Set: %v", err)
	}
	svc := NewService(store, WithMeasurementSource(fakeFlowSource{
		flowM: map[string]Measurement{
			"graph-pan":  {},
			"graph-load": {},
		},
	}))
	passed, violations, err := svc.CheckFlow(context.Background(), "demo", "graph-pan")
	if err != nil {
		t.Fatalf("CheckFlow graph-pan: %v", err)
	}
	if passed || !hasAxis(violations, "drawn_fps") || !hasAxis(violations, "input_event_count") {
		t.Fatalf("missing interaction evidence should fail closed, passed=%v violations=%#v", passed, violations)
	}
	passed, violations, err = svc.CheckFlow(context.Background(), "demo", "graph-load")
	if err != nil {
		t.Fatalf("CheckFlow graph-load: %v", err)
	}
	if !passed || len(violations) != 0 {
		t.Fatalf("load-only flow should not fail on missing interaction evidence, passed=%v violations=%#v", passed, violations)
	}
}

// A flow with no declared per-flow budget passes vacuously.
func TestCheckFlowNoBudgetPasses(t *testing.T) {
	store := NewStore()
	_, _ = store.Set(context.Background(), Budget{Scenario: "demo", GoBuildMaxMs: 1}, false)
	svc := NewService(store, WithMeasurementSource(fakeFlowSource{}))
	passed, violations, err := svc.CheckFlow(context.Background(), "demo", "scroll-list")
	if err != nil || !passed || len(violations) != 0 {
		t.Fatalf("expected vacuous pass, got passed=%v %#v err=%v", passed, violations, err)
	}
}

// [REQ:PH-BUDGET-003] A per-flow budget may only tighten under the ratchet.
func TestRatchetRejectsPerFlowLoosening(t *testing.T) {
	store := NewStore()
	base := Budget{
		Scenario: "demo",
		Ratchet:  true,
		Flows:    map[string]FlowBudget{"scroll-list": {LCPMaxMs: 2000}},
	}
	if _, err := store.Set(context.Background(), base, false); err != nil {
		t.Fatalf("Set base: %v", err)
	}
	loosen := base
	loosen.Flows = map[string]FlowBudget{"scroll-list": {LCPMaxMs: 2500}}
	if _, err := store.Set(context.Background(), loosen, false); err == nil {
		t.Fatal("expected ratchet to reject a per-flow loosening")
	}
	tighten := base
	tighten.Flows = map[string]FlowBudget{"scroll-list": {LCPMaxMs: 1800}}
	if _, err := store.Set(context.Background(), tighten, false); err != nil {
		t.Fatalf("tightening a per-flow budget must be allowed: %v", err)
	}
}

// [REQ:PH-BUDGET-003] A flows block round-trips through the on-disk ConfigStore,
// preserving sibling scenario-level axes.
func TestConfigStoreFlowRoundTrip(t *testing.T) {
	root := t.TempDir()
	store := NewConfigStore("", func(string) (string, error) { return root, nil })
	in := Budget{
		Scenario:     "demo",
		GoBuildMaxMs: 90000,
		Flows: map[string]FlowBudget{
			"scroll-list": {LCPMaxMs: 2500, ComponentCommitAvgMaxMs: 8, DrawnFPSMin: 45, DroppedFrameRateMax: 0.2, InputEventCountMin: 10},
		},
	}
	if _, err := store.Set(context.Background(), in, false); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, declared, err := store.Get(context.Background(), "demo")
	if err != nil || !declared {
		t.Fatalf("Get: declared=%v err=%v", declared, err)
	}
	if got.GoBuildMaxMs != 90000 {
		t.Fatalf("scenario-level axis lost: %#v", got)
	}
	fb, ok := got.Flows["scroll-list"]
	if !ok || fb.LCPMaxMs != 2500 || fb.ComponentCommitAvgMaxMs != 8 || fb.DrawnFPSMin != 45 || fb.DroppedFrameRateMax != 0.2 || fb.InputEventCountMin != 10 {
		t.Fatalf("flow budget not round-tripped: %#v", got.Flows)
	}
	// The block lives under performance.budgets.flows in testing.json.
	if _, err := filepath.Abs(filepath.Join(root, TestingConfigRelPath)); err != nil {
		t.Fatal(err)
	}
}

func hasAxis(violations []Violation, axis string) bool {
	for _, v := range violations {
		if v.Axis == axis {
			return true
		}
	}
	return false
}

// FindingsForFlow tags the flow slug and keeps the PERF_BUDGET_BREACH_ prefix.
func TestFindingsForFlowTagsSlug(t *testing.T) {
	findings := FindingsForFlow("scroll-list", []Violation{{Axis: "lcp", Measured: 3200, Budget: 2000, Unit: "ms"}})
	if len(findings) != 1 {
		t.Fatalf("expected one finding, got %d", len(findings))
	}
	f := findings[0]
	if f.Severity != "error" {
		t.Fatalf("per-flow breach must be ERROR, got %s", f.Severity)
	}
	if want := "PERF_BUDGET_BREACH_SCROLL-LIST_LCP"; f.Code != want {
		t.Fatalf("code = %q, want %q", f.Code, want)
	}
}
