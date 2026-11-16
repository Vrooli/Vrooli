// [REQ:ATD-P0-002] Both engines must meet one provider-neutral trust floor.
package trustfloor

import (
	"math"
	"testing"
)

func TestEvaluateRejectsMissingRequiredEvidence(t *testing.T) {
	v := Evaluate(Evidence{EngineID: "kyutai", AllIntervalsAccounted: true, BoundedRecovery: true}, DefaultThresholds)
	if v.Stable || len(v.Reasons) == 0 {
		t.Fatalf("missing evidence must block promotion: %#v", v)
	}
}

func TestEvaluateAcceptsCompleteEvidence(t *testing.T) {
	durations := make(map[string]bool, len(DurationLadder))
	for _, profile := range DurationLadder {
		durations[profile.Name] = true
	}
	faults := make(map[string]bool, len(RequiredFaults))
	for _, fault := range RequiredFaults {
		faults[fault] = true
	}
	v := Evaluate(Evidence{
		EngineID:              "whisper-local",
		AllIntervalsAccounted: true,
		BoundedRecovery:       true,
		DurationProfiles:      durations,
		FaultProfiles:         faults,
		HasBrowserProductPath: true,
		HasDeviceEvidence:     true,
	}, DefaultThresholds)
	if !v.Stable || len(v.Reasons) != 0 {
		t.Fatalf("complete evidence must pass: %#v", v)
	}
}

func TestEvaluateReplayMeasurements_AccumulatesRealtimeDurationsOnly(t *testing.T) {
	verdicts := EvaluateReplayMeasurements([]ReplayMeasurement{
		{EngineID: "kyutai", WER: 0.12, ReplayLane: "realtime", ClipDurationsMS: []int64{30_000}},
		{EngineID: "kyutai", WER: 0.08, ReplayLane: "realtime", ClipDurationsMS: []int64{60_000}},
		{EngineID: "kyutai", WER: 0.01, ReplayLane: "deterministic", ClipDurationsMS: []int64{60 * 60 * 1000}},
	}, DefaultThresholds)
	if len(verdicts) != 1 {
		t.Fatalf("verdict count = %d, want 1", len(verdicts))
	}
	v := verdicts[0]
	if v.EngineID != "kyutai" {
		t.Fatalf("engine = %q, want kyutai", v.EngineID)
	}
	if contains(v.Verdict.Reasons, "missing duration profile: 30_seconds") || contains(v.Verdict.Reasons, "missing duration profile: 1_minute") {
		t.Fatalf("realtime duration evidence was not accumulated: %#v", v.Verdict.Reasons)
	}
	if !contains(v.Verdict.Reasons, "missing duration profile: 60_minutes") {
		t.Fatalf("deterministic duration incorrectly earned realtime evidence: %#v", v.Verdict.Reasons)
	}
	if v.Verdict.Reasons == nil || v.Verdict.Stable {
		t.Fatalf("missing non-duration gates must still block promotion: %#v", v.Verdict)
	}
}

func TestEvaluateReplayMeasurements_SeparatesProviderStrategyCells(t *testing.T) {
	measurements := []ReplayMeasurement{
		{
			EngineID:        "whisper-local",
			Strategy:        "batch",
			ReplayLane:      "realtime",
			WER:             0.01,
			ClipDurationsMS: []int64{60 * 60 * 1000},
			SafetyObserved:  true,
			SafetyPassed:    false,
		},
		{
			EngineID:        "whisper-local",
			Strategy:        "vad_segment",
			ReplayLane:      "realtime",
			WER:             0.01,
			ClipDurationsMS: []int64{60 * 60 * 1000},
			SafetyObserved:  true,
			SafetyPassed:    true,
		},
	}
	verdicts := EvaluateReplayMeasurements(measurements, DefaultThresholds)
	if len(verdicts) != 2 {
		t.Fatalf("got %d cell verdicts, want 2", len(verdicts))
	}
	byStrategy := map[string]Verdict{}
	for _, verdict := range verdicts {
		byStrategy[verdict.Strategy] = verdict.Verdict
	}
	if !contains(byStrategy["batch"].Reasons, "dropped-span rate exceeds trust threshold") {
		t.Fatalf("unsafe batch cell lost its safety failure: %#v", byStrategy["batch"])
	}
	if contains(byStrategy["vad_segment"].Reasons, "dropped-span rate exceeds trust threshold") {
		t.Fatalf("safe VAD cell inherited batch safety failure: %#v", byStrategy["vad_segment"])
	}
}

func TestEvaluateReplayMeasurements_DoesNotLetGoodWERHideFailedObservation(t *testing.T) {
	verdicts := EvaluateReplayMeasurements([]ReplayMeasurement{
		{EngineID: "kyutai", ModelID: "stt-1b", Strategy: "passthrough", PolicyProfile: "default", ReplayLane: "realtime", WER: 0.10, SafetyObserved: true, SafetyPassed: true},
		{EngineID: "kyutai", ModelID: "stt-1b", Strategy: "passthrough", PolicyProfile: "default", ReplayLane: "realtime", WER: 0.40, SafetyObserved: true, SafetyPassed: true},
	}, DefaultThresholds)
	if len(verdicts) != 1 {
		t.Fatalf("verdict count = %d, want 1", len(verdicts))
	}
	if verdicts[0].Verdict.Stable || !contains(verdicts[0].Verdict.Reasons, "WER exceeds trust threshold") {
		t.Fatalf("failed WER observation must block promotion: %#v", verdicts[0].Verdict)
	}
}

func TestEvaluateReplayMeasurementsRejectsNonFiniteWER(t *testing.T) {
	for _, badWER := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		verdicts := EvaluateReplayMeasurements([]ReplayMeasurement{{EngineID: "kyutai", WER: badWER}}, DefaultThresholds)
		if len(verdicts) != 1 || verdicts[0].Verdict.Stable || !contains(verdicts[0].Verdict.Reasons, "invalid WER observation") {
			t.Fatalf("non-finite WER %v did not fail closed: %#v", badWER, verdicts)
		}
	}
}

func TestEvaluateRejectsNonFiniteQualityAndThresholdValues(t *testing.T) {
	durations := make(map[string]bool, len(DurationLadder))
	for _, profile := range DurationLadder {
		durations[profile.Name] = true
	}
	faults := make(map[string]bool, len(RequiredFaults))
	for _, fault := range RequiredFaults {
		faults[fault] = true
	}
	valid := Evidence{EngineID: "kyutai", AllIntervalsAccounted: true, BoundedRecovery: true, DurationProfiles: durations, FaultProfiles: faults, HasBrowserProductPath: true, HasDeviceEvidence: true}
	for _, mutate := range []struct {
		name string
		want string
		run  func(*Evidence, *Thresholds)
	}{{"nan WER", "WER is invalid", func(e *Evidence, _ *Thresholds) { e.WER = math.NaN() }},
		{"infinite dropped span", "dropped-span rate is invalid", func(e *Evidence, _ *Thresholds) { e.DroppedSpanRate = math.Inf(1) }},
		{"nan threshold", "qualification thresholds are invalid", func(_ *Evidence, th *Thresholds) { th.MaxWER = math.NaN() }}} {
		t.Run(mutate.name, func(t *testing.T) {
			evidence, thresholds := valid, DefaultThresholds
			mutate.run(&evidence, &thresholds)
			verdict := Evaluate(evidence, thresholds)
			if verdict.Stable || !contains(verdict.Reasons, mutate.want) {
				t.Fatalf("invalid quality input was accepted: %#v", verdict)
			}
		})
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
