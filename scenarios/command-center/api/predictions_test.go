package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestJoinPredictionsKeepsPendingTargetAndMaturesVerdict(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "predictions.json")
	now := time.Now().UTC()
	rows := []PredictionRecord{
		{MetricID: "pending", Direction: "increase", Target: 42, Horizon: now.Add(time.Hour), Verdict: EmpiricalPending},
		{MetricID: "hit", Direction: "decrease", Target: 3, Horizon: now.Add(-time.Hour), Verdict: EmpiricalHit},
	}
	payload, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COMMAND_CENTER_PREDICTIONS_PATH", path)
	server := &Server{}
	readings := []MetricEntry{{ID: "pending"}, {ID: "hit"}}
	server.joinPredictions(readings)
	if readings[0].Empirical != EmpiricalPending || readings[0].Prediction == nil || readings[0].Prediction.Target != float64(42) {
		t.Fatalf("pending prediction was not joined: %+v", readings[0])
	}
	if readings[1].Empirical != EmpiricalHit || readings[1].Prediction != nil {
		t.Fatalf("matured prediction was not joined: %+v", readings[1])
	}
}

func TestPredictionFindingsRouteUnmeasurableAndUnknownMaturedRows(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "predictions.json")
	rows := []PredictionRecord{
		{MetricID: "known", Horizon: time.Now().Add(-time.Hour), Verdict: EmpiricalUnmeasurable, Reason: "no sensor"},
		{MetricID: "unknown", Horizon: time.Now().Add(-time.Hour), Verdict: EmpiricalNone},
	}
	payload, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COMMAND_CENTER_PREDICTIONS_PATH", path)
	findings := predictionFindings(&Registry{Metrics: []MetricEntry{{ID: "known"}}})
	if len(findings) != 2 {
		t.Fatalf("expected two open-loop findings, got %+v", findings)
	}
}
