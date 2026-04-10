package experiment

import (
	"encoding/json"
	"math"
	"testing"
)

func makeOutcome(variantID string, classification string, hadFixup bool, durationSecs float64) json.RawMessage {
	data := OutcomeDataV1{
		ExecutionID:    "exec-" + variantID,
		Classification: classification,
		BacklogKind:    "fix",
		BacklogName:    "test-item",
		Purpose:        "process",
		HadFixup:       hadFixup,
		DurationSecs:   durationSecs,
	}
	dataBytes, _ := json.Marshal(data)
	envelope := map[string]any{
		"variantId":     variantID,
		"source":        "swarm-manager",
		"schemaVersion": OutcomeSchemaVersion,
		"recordedAt":    "2026-04-06T14:00:00Z",
		"data":          json.RawMessage(dataBytes),
	}
	raw, _ := json.Marshal(envelope)
	return raw
}

func TestAnalyze_BasicStats(t *testing.T) {
	outcomes := []json.RawMessage{
		makeOutcome("control", "ready", false, 300),
		makeOutcome("control", "ready_with_notes", false, 400),
		makeOutcome("control", "needs_work", true, 500),
		makeOutcome("v1", "ready", false, 200),
		makeOutcome("v1", "ready", false, 250),
		makeOutcome("v1", "needs_work", false, 600),
	}

	results, err := Analyze(outcomes)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if results.TotalOutcomes != 6 {
		t.Errorf("expected 6 total outcomes, got %d", results.TotalOutcomes)
	}

	if len(results.Variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(results.Variants))
	}

	// Find each variant
	var control, v1 *VariantStats
	for i := range results.Variants {
		switch results.Variants[i].VariantID {
		case "control":
			control = &results.Variants[i]
		case "v1":
			v1 = &results.Variants[i]
		}
	}

	if control == nil || v1 == nil {
		t.Fatal("missing expected variant in results")
	}

	// Control: 3 total, 2 ready (ready + ready_with_notes), 1 needs_work, 1 fixup
	if control.TotalRuns != 3 {
		t.Errorf("control TotalRuns: expected 3, got %d", control.TotalRuns)
	}
	if control.ReadyCount != 2 {
		t.Errorf("control ReadyCount: expected 2, got %d", control.ReadyCount)
	}
	if control.NeedsWorkCount != 1 {
		t.Errorf("control NeedsWorkCount: expected 1, got %d", control.NeedsWorkCount)
	}
	if math.Abs(control.FixupRate-1.0/3.0) > 0.01 {
		t.Errorf("control FixupRate: expected ~0.33, got %f", control.FixupRate)
	}
	if math.Abs(control.AvgDurationSecs-400) > 0.01 {
		t.Errorf("control AvgDuration: expected 400, got %f", control.AvgDurationSecs)
	}

	// V1: 3 total, 2 ready, 1 needs_work, 0 fixup
	if v1.TotalRuns != 3 {
		t.Errorf("v1 TotalRuns: expected 3, got %d", v1.TotalRuns)
	}
	if v1.ReadyCount != 2 {
		t.Errorf("v1 ReadyCount: expected 2, got %d", v1.ReadyCount)
	}
	if v1.FixupRate != 0 {
		t.Errorf("v1 FixupRate: expected 0, got %f", v1.FixupRate)
	}
}

func TestAnalyze_Empty(t *testing.T) {
	results, err := Analyze(nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if results.TotalOutcomes != 0 {
		t.Errorf("expected 0 outcomes, got %d", results.TotalOutcomes)
	}
}

func TestAnalyze_UnknownSchemaVersion(t *testing.T) {
	// Unknown schema version should be counted but not deeply analyzed
	envelope := map[string]any{
		"variantId":     "v1",
		"source":        "unknown-app",
		"schemaVersion": 99,
		"recordedAt":    "2026-04-06T14:00:00Z",
		"data":          json.RawMessage(`{"custom":"data"}`),
	}
	raw, _ := json.Marshal(envelope)

	results, err := Analyze([]json.RawMessage{raw})
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if results.TotalOutcomes != 1 {
		t.Errorf("expected 1 outcome, got %d", results.TotalOutcomes)
	}
	if len(results.Variants) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(results.Variants))
	}
	if results.Variants[0].TotalRuns != 1 {
		t.Errorf("expected 1 total run, got %d", results.Variants[0].TotalRuns)
	}
	// No classification or duration tracked for unknown schema
	if results.Variants[0].ReadyCount != 0 {
		t.Errorf("expected 0 ready, got %d", results.Variants[0].ReadyCount)
	}
}
