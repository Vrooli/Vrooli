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
