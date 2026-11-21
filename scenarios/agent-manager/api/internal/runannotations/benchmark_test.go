package runannotations

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// benchmarkFixture saves one stored noul answer per gold row from a single
// model whose raw scores are overconfident, and returns the gold rows.
func benchmarkFixture(t *testing.T, service *Service, store *Store, model, role string, scores map[string]float64, truth map[string]bool) []GoldRow {
	t.Helper()
	hash := service.Pack().LabelHashes["agent.stalled"]
	var rows []GoldRow
	var saved []Annotation
	for runID, score := range scores {
		item := annotation(runID, "agent.stalled", hash, time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC), floatPtr(score))
		item.Model, item.ModelVersion, item.Method = model, role, "DECISION_MODEL"
		saved = append(saved, item)
		rows = append(rows, GoldRow{RunID: runID, Positive: truth[runID]})
	}
	if err := store.Save(context.Background(), saved); err != nil {
		t.Fatal(err)
	}
	return rows
}

func benchmarkService(t *testing.T) (*Service, *Store) {
	t.Helper()
	return testService(t, fmt.Sprintf(frictionNoulLabel, ""), nil, nil, Options{})
}

// twoSplits returns tuning and hold-out score sets from the same overconfident
// model, with disjoint run ids.
func twoSplits() (map[string]float64, map[string]bool) {
	scores, truth := map[string]float64{}, map[string]bool{}
	for index, sample := range overconfidentSamples(240, 3, 7) {
		id := fmt.Sprintf("run-%03d", index)
		scores[id] = sample.Probs[1]
		truth[id] = sample.Positive
	}
	return scores, truth
}

func splitRows(rows []GoldRow) (tuning, holdout []GoldRow) {
	for index, row := range rows {
		if index%2 == 0 {
			tuning = append(tuning, row)
		} else {
			holdout = append(holdout, row)
		}
	}
	return tuning, holdout
}

func TestBenchmarkTuningFitsAndRecordsATuningMeasurementForOneModel(t *testing.T) {
	service, store := benchmarkService(t)
	scores, truth := twoSplits()
	rows := benchmarkFixture(t, service, store, "laya:en", "decide.local", scores, truth)
	result, err := service.Benchmark(context.Background(), BenchmarkRequest{Label: "agent.stalled", Rows: rows, GoldFile: "gold/agent.stalled.tuning.jsonl"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Refusal != "" || result.Measurement == nil {
		t.Fatalf("result = %+v", result)
	}
	m := result.Measurement
	if m.Split != SplitTuning || m.Model != "laya:en" || m.ResourceRole != "decide.local" || m.GoldFile != "gold/agent.stalled.tuning.jsonl" {
		t.Fatalf("measurement = %+v", m)
	}
	if m.DefinitionHash != service.Pack().LabelHashes["agent.stalled"] || m.Gold != 240 || m.Positives == 0 {
		t.Fatalf("measurement = %+v", m)
	}
	if !result.Fitted || m.Temperature < 2 || m.Temperature > 4.5 {
		t.Fatalf("temperature = %v (fitted %v), want the overconfidence recovered", m.Temperature, result.Fitted)
	}
	if result.ECE >= result.RawECE || result.Brier >= result.RawBrier {
		t.Fatalf("calibration did not improve ECE %.4f->%.4f Brier %.4f->%.4f", result.RawECE, result.ECE, result.RawBrier, result.Brier)
	}
	if len(result.Reliability) != ReliabilityBins || len(result.RawReliability) != ReliabilityBins {
		t.Fatal("reliability tables are not reported")
	}
	if m.Threshold <= 0 || m.Threshold > 1 || m.MeasuredAt == "" {
		t.Fatalf("measurement = %+v", m)
	}
}

func TestBenchmarkHoldoutEvaluatesTheTuningFitWithoutRefitting(t *testing.T) {
	service, store := benchmarkService(t)
	scores, truth := twoSplits()
	all := benchmarkFixture(t, service, store, "laya:en", "decide.local", scores, truth)
	tuning, holdout := splitRows(all)
	ctx := context.Background()
	fit, err := service.Benchmark(ctx, BenchmarkRequest{Label: "agent.stalled", Rows: tuning})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Benchmark(ctx, BenchmarkRequest{Label: "agent.stalled", Split: SplitHoldout, Rows: holdout, FitRows: tuning})
	if err != nil {
		t.Fatal(err)
	}
	if result.Measurement == nil || result.Measurement.Split != SplitHoldout {
		t.Fatalf("result = %+v", result)
	}
	if result.Measurement.Temperature != fit.Measurement.Temperature || result.Measurement.Threshold != fit.Measurement.Threshold {
		t.Fatalf("hold-out used T=%v thr=%v, want the tuning fit T=%v thr=%v",
			result.Measurement.Temperature, result.Measurement.Threshold, fit.Measurement.Temperature, fit.Measurement.Threshold)
	}
	if result.Gold != len(holdout) || result.FitRows != len(tuning) {
		t.Fatalf("gold = %d fit rows = %d, want %d/%d", result.Gold, result.FitRows, len(holdout), len(tuning))
	}
}

func TestBenchmarkHoldoutRefusesWithoutTuningRowsAndWhenTheSplitsOverlap(t *testing.T) {
	service, store := benchmarkService(t)
	scores, truth := twoSplits()
	all := benchmarkFixture(t, service, store, "laya:en", "decide.local", scores, truth)
	tuning, holdout := splitRows(all)
	ctx := context.Background()
	result, err := service.Benchmark(ctx, BenchmarkRequest{Label: "agent.stalled", Split: SplitHoldout, Rows: holdout})
	if err != nil || result.Measurement != nil || !strings.Contains(result.Refusal, "--fit-from") {
		t.Fatalf("result = %+v (%v), want a refusal naming --fit-from", result, err)
	}
	overlap := append(append([]GoldRow(nil), holdout...), tuning[0])
	_, err = service.Benchmark(ctx, BenchmarkRequest{Label: "agent.stalled", Split: SplitHoldout, Rows: overlap, FitRows: tuning})
	if !errors.Is(err, ErrInvalidRequest) || !strings.Contains(err.Error(), "disjoint") {
		t.Fatalf("err = %v, want an overlap refusal", err)
	}
	_, err = service.Benchmark(ctx, BenchmarkRequest{Label: "agent.stalled", Split: SplitTuning, Rows: holdout, FitRows: tuning})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("err = %v, want fit rows refused for a tuning measurement", err)
	}
}

func TestBenchmarkRefusesRowsSpanningMoreThanOneModelAndScoresAFilteredOne(t *testing.T) {
	service, store := benchmarkService(t)
	scores, truth := twoSplits()
	delete(scores, "run-100")
	rows := benchmarkFixture(t, service, store, "laya:en", "decide.local", scores, truth)
	other := benchmarkFixture(t, service, store, "winnow:e4b", "decide.local.accurate", map[string]float64{"other-1": 0.8}, map[string]bool{"other-1": true})
	ctx := context.Background()
	mixed := append(append([]GoldRow(nil), rows...), other...)
	result, err := service.Benchmark(ctx, BenchmarkRequest{Label: "agent.stalled", Rows: mixed})
	if err != nil || result.Measurement != nil || !strings.Contains(result.Refusal, "more than one") || len(result.Groups) != 2 {
		t.Fatalf("result = %+v (%v), want a refusal naming both groups", result, err)
	}
	scoped, err := service.Benchmark(ctx, BenchmarkRequest{Label: "agent.stalled", Rows: mixed, Model: "laya:en", ResourceRole: "decide.local"})
	if err != nil || scoped.Measurement == nil || scoped.Measurement.Model != "laya:en" {
		t.Fatalf("scoped = %+v (%v)", scoped, err)
	}
	if scoped.Unobserved != 1 || len(scoped.UnobservedSubjects) != 1 || scoped.UnobservedSubjects[0] != "other-1" {
		t.Fatalf("unobserved = %d %v, want the other model's row excluded and named, not scored", scoped.Unobserved, scoped.UnobservedSubjects)
	}
}

func TestBenchmarkReportsRowsWithoutAStoredAnswerAsUnobservedNotNegative(t *testing.T) {
	service, store := benchmarkService(t)
	rows := benchmarkFixture(t, service, store, "laya:en", "decide.local",
		map[string]float64{"a": 0.9, "b": 0.1, "c": 0.8, "d": 0.2}, map[string]bool{"a": true, "c": true})
	rows = append(rows, GoldRow{RunID: "never-annotated", Positive: true})
	result, err := service.Benchmark(context.Background(), BenchmarkRequest{Label: "agent.stalled", Rows: rows})
	if err != nil {
		t.Fatal(err)
	}
	if result.Labelled != 5 || result.Gold != 4 || result.Unobserved != 1 || result.Unavailable != 0 {
		t.Fatalf("labelled/gold/unobserved/unavailable = %d/%d/%d/%d", result.Labelled, result.Gold, result.Unobserved, result.Unavailable)
	}
	if len(result.UnobservedSubjects) != 1 || result.UnobservedSubjects[0] != "never-annotated" {
		t.Fatalf("unobserved subjects = %v", result.UnobservedSubjects)
	}
	if result.Precision != 1 || result.Recall != 1 {
		t.Fatalf("precision %.2f recall %.2f over a separable set", result.Precision, result.Recall)
	}
}

func TestBenchmarkOfARuleLabelIsAFixedMeasurementUnderTheRuleModel(t *testing.T) {
	labels := `[{
	  "id":"agent.stalled","group":"agent","scope":"run","method":"deterministic",
	  "rule":"tool_calls >= 1 ? 'yes' : 'no'","evidence":[{"source":"tool_summary","field":"calls"}],
	  "gate":{"precision":0.7,"recall":0.5,"min_gold":2},
	  "search":{"facet":"label:agent.stalled","min_probability":0.6},
	  "friction":{"enabled":true,"min_probability":0.6,"cost_unit":"run","importance":"high"},
	  "compatibility":{"change":"breaking"}
	}]`
	service, store := testService(t, labels, nil, nil, Options{})
	hash := service.Pack().LabelHashes["agent.stalled"]
	one := 1.0
	var saved []Annotation
	for _, id := range []string{"hit-1", "hit-2", "miss"} {
		row := annotation(id, "agent.stalled", hash, time.Now().UTC(), &one)
		row.Model, row.ModelVersion, row.Method = "", "", "deterministic"
		saved = append(saved, row)
	}
	if err := store.Save(context.Background(), saved); err != nil {
		t.Fatal(err)
	}
	rows := []GoldRow{{RunID: "hit-1", Positive: true}, {RunID: "hit-2", Positive: true}, {RunID: "miss"}}
	result, err := service.Benchmark(context.Background(), BenchmarkRequest{Label: "agent.stalled", Split: SplitHoldout, Rows: rows})
	if err != nil {
		t.Fatal(err)
	}
	m := result.Measurement
	if m == nil || m.Model != RuleModel || m.ResourceRole != "" || m.Temperature != 1 || result.Fitted {
		t.Fatalf("measurement = %+v fitted %v, want a rule measurement with no calibration", m, result.Fitted)
	}
	if m.Threshold != 0.6 || result.FP != 1 || result.TP != 2 {
		t.Fatalf("threshold %v TP %d FP %d, want the declared threshold and the miss counted", m.Threshold, result.TP, result.FP)
	}
}

func TestBenchmarkPinnedSourceReadsOnlyTheTrialRows(t *testing.T) {
	service, store := benchmarkService(t)
	hash := service.Pack().LabelHashes["agent.stalled"]
	ctx := context.Background()
	production := annotation("run-1", "agent.stalled", hash, time.Now().UTC(), floatPtr(0.1))
	production.Method = "DECISION_MODEL"
	trial := annotation("run-1", "agent.stalled", hash, time.Now().UTC(), floatPtr(0.9))
	trial.Model, trial.ModelVersion, trial.Method = "candidate", "decide.local.candidate", "DECISION_MODEL"
	if err := store.Save(ctx, []Annotation{production}); err != nil {
		t.Fatal(err)
	}
	if err := store.Trials().Save(ctx, []Annotation{trial}); err != nil {
		t.Fatal(err)
	}
	rows := []GoldRow{{RunID: "run-1", Positive: true}}
	pinned, err := service.Benchmark(ctx, BenchmarkRequest{Label: "agent.stalled", Rows: rows, Source: SourcePinned})
	if err != nil || pinned.Measurement == nil || pinned.Measurement.Model != "candidate" || pinned.TP != 1 {
		t.Fatalf("pinned = %+v (%v)", pinned, err)
	}
	prod, err := service.Benchmark(ctx, BenchmarkRequest{Label: "agent.stalled", Rows: rows})
	if err != nil || prod.Measurement == nil || prod.Measurement.Model != "model-a" || prod.FN != 1 {
		t.Fatalf("production = %+v (%v), want the production answer, untouched by the trial", prod, err)
	}
}
