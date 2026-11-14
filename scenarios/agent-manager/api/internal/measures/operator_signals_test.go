package measures

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/interventions"

	measurelib "github.com/vrooli/measures-go"
)

type signalWindows struct {
	calls [][2]time.Time
	data  map[int][]interventions.SignalObservation // by call index: 0 current, 1 prior
}

func (s *signalWindows) SignalObservations(_ context.Context, _ string, from, to time.Time) ([]interventions.SignalObservation, error) {
	s.calls = append(s.calls, [2]time.Time{from, to})
	return s.data[len(s.calls)-1], nil
}

func signalObs(n int, label interventions.Signal, mode string) []interventions.SignalObservation {
	p := 0.9
	out := make([]interventions.SignalObservation, n)
	for i := range out {
		out[i] = interventions.SignalObservation{Status: interventions.StatusClassified, Signal: label, Probability: &p, Distribution: map[interventions.Signal]float64{label: p}, SignalModel: "winnow:e4b", ActualModel: "haiku"}
		if mode != "" {
			out[i].ModeID, out[i].ModeRelation = mode, "applied"
		}
	}
	return out
}

func runSignals(t *testing.T, source *signalWindows) []map[string]string {
	t.Helper()
	h := NewHandler(nil, func() time.Time { return time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC) })
	h.SetSignals(source)
	registry, err := h.Registry()
	if err != nil {
		t.Fatal(err)
	}
	served, err := registry.Execute(context.Background(), measurelib.MeasureRequest{Measure: UsageOperatorSignals, Params: map[string]string{"window": "last_7d"}})
	if err != nil {
		t.Fatal(err)
	}
	return served.Fields
}

func signalRow(rows []map[string]string, label string) map[string]string {
	for _, row := range rows {
		if row["signal"] == label {
			return row
		}
	}
	return nil
}

// Below the gate a label reports no count and no rate, only its measured
// accuracy and an unproven state: empty is unknown, never zero.
func TestOperatorSignalsMeasureWithholdsRatesBelowTheGate(t *testing.T) {
	originalGate, originalMeasured := interventions.ProvenSignalGate, interventions.SignalHoldout
	interventions.ProvenSignalGate = interventions.SignalGate{}
	interventions.SignalHoldout = interventions.SignalMeasurements{"winnow:e4b": {interventions.ReportedFailure: {TP: 6, Pred: 10, Gold: 12}}}
	defer func() { interventions.ProvenSignalGate, interventions.SignalHoldout = originalGate, originalMeasured }()
	rows := runSignals(t, &signalWindows{data: map[int][]interventions.SignalObservation{0: signalObs(40, interventions.ReportedFailure, "")}})
	failure := signalRow(rows, "reported_failure")
	if failure["state"] != "unproven" || failure["rate"] != "" || failure["count"] != "" || failure["measured_precision"] != "0.6000" || failure["measured_recall"] != "0.5000" || failure["note"] == "" || failure["coverage"] != "1.0000" || failure["classified"] != "40" {
		t.Fatalf("row = %v, want unproven with no rate and the measured precision", failure)
	}
}

func TestOperatorSignalsMeasureReportsRatesRangesAndTheOwnBaselineWhenProven(t *testing.T) {
	originalGate := interventions.ProvenSignalGate
	interventions.ProvenSignalGate = interventions.SignalGate{"winnow:e4b": {interventions.ReportedFailure: 0.5}}
	defer func() { interventions.ProvenSignalGate = originalGate }()
	current := append(signalObs(10, interventions.ReportedFailure, "um-1"), signalObs(30, interventions.Neutral, "um-1")...)
	prior := append(signalObs(2, interventions.ReportedFailure, ""), signalObs(48, interventions.Neutral, "")...)
	source := &signalWindows{data: map[int][]interventions.SignalObservation{0: current, 1: prior}}
	rows := runSignals(t, source)
	if len(source.calls) != 2 || !source.calls[1][1].Equal(source.calls[0][0]) || source.calls[1][1].Sub(source.calls[1][0]) != source.calls[0][1].Sub(source.calls[0][0]) {
		t.Fatalf("windows = %v, want the current window and the equal-length period just before it", source.calls)
	}
	failure := signalRow(rows, "reported_failure")
	if failure["state"] != "proven" || failure["mode"] != "um-1" || failure["model"] != "haiku" || failure["count"] != "10" || failure["rate"] != "0.2500" || failure["low"] == "" || failure["high"] == "" ||
		failure["baseline_rate"] != "0.0400" || failure["baseline_classified"] != "50" || failure["comparison"] != "higher" || failure["difference_low"] == "" || failure["classifier_version"] != interventions.SignalVersion {
		t.Fatalf("row = %v", failure)
	}
	if correction := signalRow(rows, "correction"); correction["state"] != "unproven" || correction["rate"] != "" {
		t.Fatalf("a label the gate does not prove must stay unproven: %v", correction)
	}
}

func TestOperatorSignalsMeasureSwitchedOff(t *testing.T) {
	t.Setenv(interventions.SignalsSwitchEnv, "off")
	rows := runSignals(t, &signalWindows{})
	if len(rows) != 1 || rows[0]["state"] != "off" {
		t.Fatalf("rows = %v, want a single off row and no read of the messages", rows)
	}
}

func TestOperatorSignalsMeasureIsDefined(t *testing.T) {
	if definitionFor(UsageOperatorSignals).SourceTable == "" {
		t.Fatal("usage.operator_signals needs a backend definition")
	}
}

type correctionSource struct {
	signalWindows
	details map[string]interventions.MessageDetail
}

func (s *correctionSource) MessageDetails(_ context.Context, ids []string) (map[string]interventions.MessageDetail, error) {
	out := map[string]interventions.MessageDetail{}
	for _, id := range ids {
		if detail, ok := s.details[id]; ok {
			out[id] = detail
		}
	}
	return out, nil
}

// The friction section shows the messages the proven signal gate admits as an
// explicit negative, newest first with their words and prior work, and the
// precision the gate was proven at. Messages under the gate never appear.
func TestCorrectionsSectionListsProvenExplicitNegativesNewestFirst(t *testing.T) {
	negative := func(id, at string, mass float64) interventions.SignalObservation {
		return interventions.SignalObservation{
			EventID: id, RunID: "run-" + id, OccurredAt: at, Status: interventions.StatusClassified, SignalModel: "winnow:e4b",
			Distribution: map[interventions.Signal]float64{interventions.Correction: mass, interventions.Neutral: 1 - mass},
		}
	}
	observations := []interventions.SignalObservation{
		negative("old", "2026-11-01T10:00:00Z", 0.9),
		negative("new", "2026-11-05T10:00:00Z", 0.6),
		negative("weak", "2026-11-04T10:00:00Z", 0.2),
		{EventID: "pending", Status: interventions.StatusUnclassified},
	}
	source := &correctionSource{
		signalWindows: signalWindows{data: map[int][]interventions.SignalObservation{0: observations}},
		details:       map[string]interventions.MessageDetail{"new": {Text: "too small: take the whole class", Work: interventions.Work{Requests: 4, ToolCalls: 9, Tokens: 1200}}},
	}
	gate := interventions.SignalGate{"winnow:e4b": {interventions.ExplicitNegative: 0.40}}
	measured := interventions.SignalMeasurements{"winnow:e4b": {interventions.ExplicitNegative: {TP: 33, Pred: 41, Gold: 35}}}
	section, err := correctionsSection(context.Background(), source, gate, measured, time.Time{}, time.Time{}, 10, 1)
	if err != nil {
		t.Fatal(err)
	}
	if section.GetCount() != 2 || section.GetClassified() != 3 || section.GetPerAgentHour() != 0.2 || section.GetPrecision() < 0.80 || section.GetRecall() < 0.94 {
		t.Fatalf("section = %+v, want 2 of 3 classified at the proven precision", section)
	}
	if len(section.GetExamples()) != 1 {
		t.Fatalf("examples = %+v, want the newest one within the limit", section.GetExamples())
	}
	example := section.GetExamples()[0]
	if example.GetEventId() != "new" || example.GetText() != "too small: take the whole class" || example.GetWorkToolCalls() != 9 || example.GetProbability() != 0.6 || example.GetCategory() != string(interventions.ExplicitNegative) {
		t.Fatalf("example = %+v", example)
	}
	if unproven, err := correctionsSection(context.Background(), source, interventions.SignalGate{}, measured, time.Time{}, time.Time{}, 10, 1); err != nil || unproven != nil {
		t.Fatalf("an unproven gate must show no corrections, got %+v, %v", unproven, err)
	}
	t.Setenv(interventions.SignalsSwitchEnv, "off")
	if off, err := correctionsSection(context.Background(), source, gate, measured, time.Time{}, time.Time{}, 10, 1); err != nil || off != nil {
		t.Fatalf("switched-off signals must show no corrections, got %+v, %v", off, err)
	}
}
