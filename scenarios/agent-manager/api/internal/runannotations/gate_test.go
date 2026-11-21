package runannotations

import (
	"context"
	"fmt"
	"strings"
	"testing"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

const hashA = "0000000000000000000000000000000000000000000000000000000000000001"
const hashB = "0000000000000000000000000000000000000000000000000000000000000002"

func measurement(hash, model, role, split string, gold int, precision, recall float64) Measurement {
	return Measurement{
		DefinitionHash: hash, Model: model, ResourceRole: role, Split: split, Gold: gold,
		Positives: gold / 4, Precision: precision, Recall: recall, Threshold: 0.5, Temperature: 1,
		MeasuredAt: "2026-11-20T10:00:00Z",
	}
}

func TestProofIsPerModelResourceRoleAndDefinitionAndOnlyFromAHoldout(t *testing.T) {
	gate := Gate{Precision: 0.8, Recall: 0.7, MinGold: 40, Measurements: []Measurement{
		measurement(hashA, "winnow:e4b", "decide.local.accurate", SplitHoldout, 60, 0.9, 0.8),
		measurement(hashA, "laya:en", "decide.local", SplitTuning, 60, 0.95, 0.95),
		measurement(hashA, "jev", "decide.jev.edenai", SplitHoldout, 20, 0.95, 0.95),
		measurement(hashA, "tiny", "decide.local.fast", SplitHoldout, 60, 0.5, 0.9),
	}}
	cases := []struct {
		name         string
		hash, model  string
		role         string
		want         string
		reasonSubstr string
	}{
		{"holdout that clears proves its own model", hashA, "winnow:e4b", "decide.local.accurate", GateProven, ""},
		{"same model on another resource role is not proven", hashA, "winnow:e4b", "decide.local", GateUnmeasured, "other models or resource roles"},
		{"a tuning measurement is never proof", hashA, "laya:en", "decide.local", GateTuningOnly, "fitted on"},
		{"a hold-out below min_gold is insufficient", hashA, "jev", "decide.jev.edenai", GateInsufficientGold, "needs 40"},
		{"a hold-out below target is unproven", hashA, "tiny", "decide.local.fast", GateUnproven, "below target"},
		{"another definition revision inherits nothing", hashB, "winnow:e4b", "decide.local.accurate", GateUnmeasured, "earlier definition revision"},
		{"an observation with no model cannot be proven", hashA, "", "", GateUnmeasured, "names no model"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status := gate.Status(tc.hash, tc.model, tc.role)
			if status.State != tc.want || !strings.Contains(status.Reason, tc.reasonSubstr) {
				t.Fatalf("status = %s (%q), want %s containing %q", status.State, status.Reason, tc.want, tc.reasonSubstr)
			}
			if gate.Proven(tc.hash, tc.model, tc.role) != (tc.want == GateProven) {
				t.Fatal("Proven disagrees with Status")
			}
		})
	}
}

func TestHoldoutOutranksTuningAndNewestWinsWithinASplit(t *testing.T) {
	older := measurement(hashA, "m", "r", SplitHoldout, 50, 0.6, 0.6)
	older.MeasuredAt = "2026-11-19T10:00:00Z"
	newer := measurement(hashA, "m", "r", SplitHoldout, 50, 0.9, 0.9)
	newer.Temperature = 2.5
	gate := Gate{Precision: 0.8, Recall: 0.7, MinGold: 10, Measurements: []Measurement{
		older, newer, measurement(hashA, "m", "r", SplitTuning, 50, 0.99, 0.99),
	}}
	if status := gate.Status(hashA, "m", "r"); status.State != GateProven || status.Precision != 0.9 {
		t.Fatalf("status = %+v, want the newest hold-out", status)
	}
	if temperature, threshold := gate.Calibration(hashA, "m", "r"); temperature != 2.5 || threshold != 0.5 {
		t.Fatalf("calibration = %v/%v, want the hold-out's temperature and threshold", temperature, threshold)
	}
	if temperature, _ := gate.Calibration(hashA, "other", "r"); temperature != 1 {
		t.Fatalf("an unmeasured model was calibrated with %v", temperature)
	}
}

func TestUpsertReplacesTheSameKeyAndSplitOnly(t *testing.T) {
	var gate Gate
	gate.Upsert(measurement(hashA, "m", "r", SplitTuning, 10, 0.5, 0.5))
	gate.Upsert(measurement(hashA, "m", "r", SplitHoldout, 10, 0.5, 0.5))
	gate.Upsert(measurement(hashA, "m", "r", SplitHoldout, 20, 0.9, 0.9))
	gate.Upsert(measurement(hashA, "n", "r", SplitHoldout, 20, 0.9, 0.9))
	if len(gate.Measurements) != 3 {
		t.Fatalf("measurements = %d, want tuning, one hold-out per model", len(gate.Measurements))
	}
	if got, _ := gate.forKey(hashA, "m", "r"); got.Gold != 20 {
		t.Fatalf("hold-out = %+v, want the replacement", got)
	}
}

func TestBestSummarizesTheStrongestModelWithoutHidingTheOthers(t *testing.T) {
	gate := Gate{Precision: 0.8, Recall: 0.7, MinGold: 10, Measurements: []Measurement{
		measurement(hashA, "good", "r", SplitHoldout, 20, 0.9, 0.9),
		measurement(hashA, "weak", "r", SplitHoldout, 20, 0.3, 0.9),
	}}
	best := gate.Best(hashA, []ModelKey{{Model: "weak", ResourceRole: "r"}, {Model: "good", ResourceRole: "r"}})
	if best.State != GateProven || best.Model != "good" {
		t.Fatalf("best = %+v, want the proven model", best)
	}
	if only := gate.Best(hashA, []ModelKey{{Model: "weak", ResourceRole: "r"}}); only.State != GateUnproven {
		t.Fatalf("a label observed only with the weak model is %s, want unproven", only.State)
	}
	if none := (Gate{MinGold: 10}).Best(hashA, nil); none.State != GateUnmeasured {
		t.Fatalf("an unmeasured label is %s", none.State)
	}
}

const frictionNoulLabel = `[{
  "id":"agent.stalled","group":"agent","scope":"run","method":"decision",
  "question":{"type":"noul","instructions":"spinning?","true_description":"yes","false_description":"no"},
  "evidence":[{"source":"tool_summary","field":"calls"}],
  "gate":{"precision":0.7,"recall":0.5,"min_gold":10%s},
  "search":{"facet":"label:agent.stalled","min_probability":0.6},
  "friction":{"enabled":true,"min_probability":0.6,"cost_unit":"run","importance":"high"},
  "compatibility":{"change":"breaking"}
}]`

func TestRateCountsOnlyObservationsWhoseOwnModelIsProven(t *testing.T) {
	plain := fmt.Sprintf(frictionNoulLabel, "")
	pack, err := ParsePack([]byte(packWithLabels(plain)))
	if err != nil {
		t.Fatal(err)
	}
	hash, err := LabelDefinitionHash(pack.Labels[0])
	if err != nil {
		t.Fatal(err)
	}
	proof := fmt.Sprintf(`,"measurements":[{"definition_hash":%q,"model":"winnow","resource_role":"decide.local","precision":0.9,"recall":0.8,"gold":20,"positives":8,"threshold":0.6,"temperature":1,"split":"holdout"}]`, hash)
	labels := fmt.Sprintf(frictionNoulLabel, proof)
	first, second := runFact("run-1"), runFact("run-2")
	first.ToolSummary.ToolCalls, second.ToolSummary.ToolCalls = 3, 3
	model := "winnow"
	decision := &fakeDecision{resolve: func(*inferencev1.DecideBatchRequest) (string, string) { return model, "decide.local" }}
	service, _ := testService(t, labels, []RunFacts{first, second}, decision, Options{})
	ctx := context.Background()
	if _, err := service.Annotate(ctx, "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	// A second model answers the next run. Its answer is also a friction positive,
	// but nothing measured it, so it must not enter the proven rate.
	model = "unmeasured-model"
	view, err := service.Annotate(ctx, "run-2", AnnotateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if view.Labels[0].Proven || view.Labels[0].GateState != GateUnmeasured {
		t.Fatalf("run-2 view = %+v, want its own model unproven", view.Labels[0].Gate)
	}
	coverage, err := service.Coverage(ctx, "7d")
	if err != nil {
		t.Fatal(err)
	}
	label := coverage.Labels[0]
	if label.Observed != 2 || label.Positives != 2 || label.ProvenObserved != 1 || label.ProvenPositives != 1 {
		t.Fatalf("label = observed %d positives %d proven %d/%d", label.Observed, label.Positives, label.ProvenPositives, label.ProvenObserved)
	}
	if label.Rate == nil || *label.Rate != 1 || label.State != "partial" {
		t.Fatalf("rate = %v state = %s, want a rate over the proven model alone and a partial state", label.Rate, label.State)
	}
	if len(label.Models) != 2 {
		t.Fatalf("models = %+v, want one row per (model, resource role)", label.Models)
	}
	for _, row := range label.Models {
		if row.Proven != (row.Model == "winnow") {
			t.Fatalf("model row %+v has the wrong proof", row)
		}
	}
}

func TestLabelWithNoProvenModelHasNoRateAtAll(t *testing.T) {
	labels := fmt.Sprintf(frictionNoulLabel, "")
	fact := runFact("run-1")
	fact.ToolSummary.ToolCalls = 3
	service, _ := testService(t, labels, []RunFacts{fact}, &fakeDecision{}, Options{})
	if _, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	coverage, err := service.Coverage(context.Background(), "7d")
	if err != nil {
		t.Fatal(err)
	}
	if label := coverage.Labels[0]; label.Rate != nil || label.State != "unproven" || label.Positives != 1 {
		t.Fatalf("label = %+v, want positives shown as evidence and no rate", label)
	}
}
