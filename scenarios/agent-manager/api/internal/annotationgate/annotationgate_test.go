package annotationgate

import (
	"testing"

	"agent-manager/internal/runannotations"
)

func gateRowsByKind(rows []map[string]string, kind string) []map[string]string {
	var out []map[string]string
	for _, row := range rows {
		if row["kind"] == kind {
			out = append(out, row)
		}
	}
	return out
}

func TestAnnotationGateNamesTheModelBehindTheRateAndCountsOnlyItsAnswers(t *testing.T) {
	rate := 0.5
	proven := runannotations.GateStatus{
		State: "proven", Model: "winnow:e4b", ResourceRole: "decide.local.accurate", Split: "holdout",
		Precision: 0.9, Recall: 0.8, Gold: 44, Positives: 9, Threshold: 0.6, Temperature: 1.4, ECE: 0.04, Brier: 0.11,
		MeasuredAt: "2026-11-20T10:00:00Z", MinGold: 40, TargetPrecision: 0.85, TargetRecall: 0.7,
	}
	untested := runannotations.GateStatus{State: "unmeasured", Model: "laya:en", ResourceRole: "decide.local", MinGold: 40, TargetPrecision: 0.85, TargetRecall: 0.7, Reason: "no measurement for this model and definition"}
	report := runannotations.CoverageResponse{
		Pack: runannotations.PackRef{Name: "default", Version: "7"}, EligibleRuns: 20, ObservedRuns: 10, Coverage: 0.5,
		Provenance: &runannotations.ProvenanceSummary{RunsHumanText: 3, HumanMessages: 30, RunsStale: 2},
		Labels: []runannotations.CoverageLabel{{
			ID: "operator.dissatisfied", Scope: runannotations.ScopeMessage, Group: "operator", State: "partial", Observed: 8, Eligible: 30, Positives: 4,
			ProvenObserved: 6, ProvenPositives: 3, Rate: &rate, Gate: proven, Friction: runannotations.Friction{Enabled: true},
			Population: &runannotations.Population{Basis: runannotations.PopulationHumanOperator, Unit: "message", Eligible: 30},
			Models: []runannotations.ModelCoverage{
				{Model: "winnow:e4b", ResourceRole: "decide.local.accurate", Observed: 6, Positives: 3, Proven: true, Gate: proven},
				{Model: "laya:en", ResourceRole: "decide.local", Observed: 2, Positives: 1, Gate: untested},
			},
		}},
	}
	rows := Rows(report)

	coverage := gateRowsByKind(rows, "coverage")
	if len(coverage) != 1 || coverage[0]["observed_runs"] != "10" || coverage[0]["human_messages"] != "30" || coverage[0]["stale_runs"] != "2" || coverage[0]["pack"] != "default" {
		t.Fatalf("coverage row = %+v", coverage)
	}
	labels := gateRowsByKind(rows, "label")
	if len(labels) != 1 {
		t.Fatalf("label rows = %+v", labels)
	}
	label := labels[0]
	want := map[string]string{
		"label": "operator.dissatisfied", "scope": "message", "state": "partial", "rate": "0.5",
		"proven_observed": "6", "observed": "8", "population": "human_operator", "population_unit": "message",
		"gate_state": "proven", "gate_model": "winnow:e4b", "gate_role": "decide.local.accurate", "split": "holdout",
		"precision": "0.9", "recall": "0.8", "gold": "44", "gold_positives": "9", "ece": "0.04", "brier": "0.11",
		"temperature": "1.4", "calibration": "calibrated", "measured_at": "2026-11-20T10:00:00Z",
	}
	for key, value := range want {
		if label[key] != value {
			t.Errorf("label[%q] = %q, want %q (row %+v)", key, label[key], value, label)
		}
	}
	models := gateRowsByKind(rows, "model")
	if len(models) != 2 {
		t.Fatalf("model rows = %+v", models)
	}
	if models[0]["proven"] != "true" || models[0]["model"] != "winnow:e4b" {
		t.Fatalf("proven model row = %+v", models[0])
	}
	other := models[1]
	if other["proven"] != "false" || other["gate_state"] != "unmeasured" || other["observed"] != "2" {
		t.Fatalf("unmeasured model row = %+v", other)
	}
	for _, key := range []string{"precision", "recall", "ece", "gold", "calibration", "gate_model"} {
		if _, present := other[key]; present {
			t.Errorf("an unmeasured model carries %q = %q: an empty cell must not read as zero", key, other[key])
		}
	}
}

func TestAnnotationGateNeverGivesAnUnprovenLabelARate(t *testing.T) {
	tuning := runannotations.GateStatus{State: "tuning_only", Model: "laya:en", ResourceRole: "decide.local", Split: "tuning", Precision: 1, Recall: 1, Gold: 47, Temperature: 1, MinGold: 40, Reason: "measured on the rows the calibration was fitted on"}
	report := runannotations.CoverageResponse{Labels: []runannotations.CoverageLabel{{
		ID: "work.blocked_by_tool", Group: "work", State: "unproven", Observed: 5, Eligible: 5, Gate: tuning,
		Models: []runannotations.ModelCoverage{{Model: "laya:en", ResourceRole: "decide.local", Observed: 5, Gate: tuning}},
	}}}
	label := gateRowsByKind(Rows(report), "label")[0]
	if _, has := label["rate"]; has {
		t.Fatalf("an unproven label carries a rate: %+v", label)
	}
	if label["state"] != "unproven" || label["gate_state"] != "tuning_only" || label["split"] != "tuning" || label["calibration"] != "raw" {
		t.Fatalf("label = %+v", label)
	}
	if label["gate_reason"] == "" {
		t.Fatal("an unproven label says nothing about why")
	}
}

func TestAnnotationGateMarksRuleLabelsAndLeavesUnmeasuredCellsOut(t *testing.T) {
	rule := runannotations.GateStatus{State: "proven", Model: runannotations.RuleModel, Split: "holdout", Precision: 1, Recall: 1, Gold: 50, Temperature: 1}
	report := runannotations.CoverageResponse{Labels: []runannotations.CoverageLabel{
		{ID: "run.type", Group: "run", State: "proven", Observed: 4, Eligible: 4, Gate: rule},
		{ID: "agent.discovery_slow", Group: "agent", State: "not_observed", Gate: runannotations.GateStatus{State: "unmeasured", MinGold: 40}},
	}}
	labels := gateRowsByKind(Rows(report), "label")
	if labels[0]["calibration"] != "rule" || labels[0]["gate_model"] != "rule" {
		t.Fatalf("rule label = %+v", labels[0])
	}
	if _, has := labels[1]["calibration"]; has {
		t.Fatalf("an unmeasured label names a calibration: %+v", labels[1])
	}
	if _, has := labels[1]["gate_model"]; has {
		t.Fatalf("an unmeasured label names a model: %+v", labels[1])
	}
	unmeasured := runannotations.GateStatus{State: "unmeasured", Model: "laya:en", ResourceRole: "decide.local", MinGold: 40}
	row := labelGateRow(runannotations.CoverageLabel{ID: "run.type", State: "unproven", Observed: 3, Gate: unmeasured})
	if _, has := row["gate_model"]; has {
		t.Fatalf("a gate with no measurement names the model it was asked about as if it were measured: %+v", row)
	}
}

func TestAnnotationGateEmptyWindowIsUnknownNotZero(t *testing.T) {
	rows := Rows(runannotations.CoverageResponse{Notes: []string{"no annotations recorded in this window: coverage is unknown, not zero friction"}})
	if len(rows) != 1 || rows[0]["kind"] != "coverage" || rows[0]["note"] == "" {
		t.Fatalf("rows = %+v", rows)
	}
}
