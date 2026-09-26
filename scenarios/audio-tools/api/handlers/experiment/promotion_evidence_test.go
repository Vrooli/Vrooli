package experiment

import (
	"testing"

	intexp "audio-tools/internal/experiment"
	trustfloor "audio-tools/internal/qualification"

	"github.com/stretchr/testify/require"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
)

// [REQ:ATD-P0-002] Persisted evidence may accumulate only full realtime rungs.
func TestPromotionVerdictsForMeasurements_AccumulatesPersistedRealtimeRungs(t *testing.T) {
	verdicts := promotionVerdictsForMeasurements(replayMeasurements(&evalv1.EvalReport{PerStrategy: []*evalv1.StrategyReport{
		{EngineId: "kyutai", Strategy: "passthrough", Wer: 0.12, ReplayLane: "realtime", PerClip: []*evalv1.ClipReport{{AudioDurationMs: 30_000}}},
		{EngineId: "kyutai", Strategy: "passthrough", Wer: 0.08, ReplayLane: "realtime", PerClip: []*evalv1.ClipReport{{AudioDurationMs: 60_000}}},
		{EngineId: "kyutai", Strategy: "passthrough", Wer: 0.01, ReplayLane: "deterministic", PerClip: []*evalv1.ClipReport{{AudioDurationMs: 60 * 60 * 1000}}},
	}}), nil)
	require.Len(t, verdicts, 1)
	require.NotContains(t, verdicts[0].GetReasons(), "missing duration profile: 30_seconds")
	require.NotContains(t, verdicts[0].GetReasons(), "missing duration profile: 1_minute")
	require.Contains(t, verdicts[0].GetReasons(), "missing duration profile: 60_minutes")
}

// [REQ:ATD-P0-002] A persisted quality failure must remain visible after the
// report is reduced to promotion measurements; duration evidence cannot mask
// a threshold-sized dropped span.
func TestPromotionVerdictsForMeasurements_RejectsObservedSafetyFailure(t *testing.T) {
	verdicts := promotionVerdictsForMeasurements([]trustfloor.ReplayMeasurement{{
		EngineID:        "whisper-local",
		WER:             0.01,
		ReplayLane:      "realtime",
		ClipDurationsMS: []int64{60 * 60 * 1000},
		SafetyObserved:  true,
		SafetyPassed:    false,
	}}, nil)
	require.Len(t, verdicts, 1)
	require.Contains(t, verdicts[0].GetReasons(), "dropped-span rate exceeds trust threshold")
}

func TestPromotionVerdictsForMeasurements_SeparatesStrategiesAndRejectsLegacyRows(t *testing.T) {
	verdicts := promotionVerdictsForMeasurements(replayMeasurements(&evalv1.EvalReport{PerStrategy: []*evalv1.StrategyReport{
		{EngineId: "whisper-local", Strategy: "batch", Wer: 0.01, ReplayLane: "realtime", Safety: &evalv1.SafetyGateReport{Passed: false}},
		{EngineId: "whisper-local", Strategy: "vad_segment", Wer: 0.01, ReplayLane: "realtime", Safety: &evalv1.SafetyGateReport{Passed: true}},
		{EngineId: "whisper-local", Wer: 0.01, ReplayLane: "realtime", Safety: &evalv1.SafetyGateReport{Passed: false}},
	}}), nil)
	require.Len(t, verdicts, 2)
	byStrategy := map[string]*evalv1.PromotionVerdict{}
	for _, verdict := range verdicts {
		byStrategy[verdict.GetStrategy()] = verdict
	}
	require.Contains(t, byStrategy["batch"].GetReasons(), "dropped-span rate exceeds trust threshold")
	require.NotContains(t, byStrategy["vad_segment"].GetReasons(), "dropped-span rate exceeds trust threshold")
}

func TestSameQualificationMachine_RequiresStableHostAndArchitecture(t *testing.T) {
	require.True(t, sameQualificationMachine([]byte(`{"host":"runner","goos":"linux","goarch":"amd64"}`), []byte(`{"host":"runner","goos":"linux","goarch":"amd64"}`)))
	require.False(t, sameQualificationMachine([]byte(`{"host":"runner-a","goos":"linux","goarch":"amd64"}`), []byte(`{"host":"runner-b","goos":"linux","goarch":"amd64"}`)))
	require.False(t, sameQualificationMachine([]byte(`{}`), []byte(`{}`)))
}

// [REQ:ATD-P0-002] Dedicated artifacts are the only way browser, recovery,
// interval, device, and fault gates may become promotable for a provider cell.
func TestPromotionVerdictsForMeasurements_UsesDedicatedQualificationArtifactsForExactCell(t *testing.T) {
	measurements := []trustfloor.ReplayMeasurement{{
		EngineID: "kyutai", Strategy: "passthrough", PolicyProfile: "dictation",
		WER: 0.01, ReplayLane: "realtime", SafetyObserved: true, SafetyPassed: true,
		ClipDurationsMS: []int64{60 * 60 * 1000},
	}}
	qualification := []trustfloor.QualificationMeasurement{
		{EngineID: "kyutai", Strategy: "passthrough", PolicyProfile: "dictation", Kind: trustfloor.QualificationIntervalAccounting, Passed: true, AllIntervalsAccounted: true},
		{EngineID: "kyutai", Strategy: "passthrough", PolicyProfile: "dictation", Kind: trustfloor.QualificationBoundedRecovery, Passed: true},
		{EngineID: "kyutai", Strategy: "passthrough", PolicyProfile: "dictation", Kind: trustfloor.QualificationBrowserProductPath, Passed: true},
		{EngineID: "kyutai", Strategy: "passthrough", PolicyProfile: "dictation", Kind: trustfloor.QualificationDevice, Passed: true},
	}
	for _, fault := range trustfloor.RequiredFaults {
		qualification = append(qualification, trustfloor.QualificationMeasurement{
			EngineID: "kyutai", Strategy: "passthrough", PolicyProfile: "dictation",
			Kind: trustfloor.QualificationFault, FaultProfile: fault, Passed: true,
		})
	}
	// A passing artifact for another strategy must not qualify this cell.
	qualification = append(qualification, trustfloor.QualificationMeasurement{
		EngineID: "kyutai", Strategy: "vad_segment", PolicyProfile: "dictation",
		Kind: trustfloor.QualificationFault, FaultProfile: "provider_busy", Passed: true,
	})

	verdicts := promotionVerdictsForMeasurements(measurements, qualification)
	require.Len(t, verdicts, 1)
	require.True(t, verdicts[0].GetStable(), verdicts[0].GetReasons())
}

func TestPromotionVerdictsForMeasurements_IntervalAccountingCannotImplicitlyClearDeliveryFailures(t *testing.T) {
	measurements := []trustfloor.ReplayMeasurement{{
		EngineID: "kyutai", Strategy: "passthrough", WER: 0.01,
	}}
	qualification := []trustfloor.QualificationMeasurement{{
		EngineID: "kyutai", Strategy: "passthrough", Kind: trustfloor.QualificationIntervalAccounting,
		Passed: true, AllIntervalsAccounted: true, DuplicateCommittedSegments: 1, SilentTerminalOutcomes: 1,
	}}
	verdicts := promotionVerdictsForMeasurements(measurements, qualification)
	require.Len(t, verdicts, 1)
	require.Contains(t, verdicts[0].GetReasons(), "duplicate committed segments were observed")
	require.Contains(t, verdicts[0].GetReasons(), "silent terminal outcomes were observed")
}

func TestPromotionVerdictsForMeasurements_ShowsFailedQualificationArtifact(t *testing.T) {
	verdicts := promotionVerdictsForMeasurements([]trustfloor.ReplayMeasurement{{
		EngineID: "kyutai", Strategy: "passthrough", WER: 0.01,
	}}, []trustfloor.QualificationMeasurement{{
		EngineID: "kyutai", Strategy: "passthrough", Kind: trustfloor.QualificationFault,
		FaultProfile: "provider_busy", Passed: false,
	}})
	require.Len(t, verdicts, 1)
	require.Contains(t, verdicts[0].GetReasons(), "qualification evidence failed: fault:provider_busy")
}

func TestPromotionVerdictsForMeasurements_SeparatesModelVersions(t *testing.T) {
	measurements := []trustfloor.ReplayMeasurement{
		{EngineID: "kyutai", ModelID: "kyutai/stt-1b-en_fr@0.2.0", Strategy: "passthrough", WER: 0.01},
		{EngineID: "kyutai", ModelID: "kyutai/stt-2b@0.3.0", Strategy: "passthrough", WER: 0.01},
	}
	qualification := []trustfloor.QualificationMeasurement{{
		EngineID: "kyutai", ModelID: "kyutai/stt-1b-en_fr@0.2.0", Strategy: "passthrough",
		Kind: trustfloor.QualificationFault, FaultProfile: "provider_busy", Passed: true,
	}}
	verdicts := promotionVerdictsForMeasurements(measurements, qualification)
	require.Len(t, verdicts, 2)
	byModel := map[string]*evalv1.PromotionVerdict{}
	for _, verdict := range verdicts {
		byModel[verdict.GetModelId()] = verdict
	}
	require.NotContains(t, byModel["kyutai/stt-1b-en_fr@0.2.0"].GetReasons(), "missing fault profile: provider_busy")
	require.Contains(t, byModel["kyutai/stt-2b@0.3.0"].GetReasons(), "missing fault profile: provider_busy")
}

func TestQualificationMeasurements_RejectsDifferentMachine(t *testing.T) {
	items := []intexp.QualificationEvidence{
		{EngineID: "kyutai", Strategy: "passthrough", Kind: trustfloor.QualificationFault, FaultProfile: "provider_busy", Passed: true, MachineJSON: []byte(`{"host":"runner-a","goos":"linux","goarch":"amd64"}`)},
		{EngineID: "kyutai", Strategy: "passthrough", Kind: trustfloor.QualificationFault, FaultProfile: "delayed_ready", Passed: true, MachineJSON: []byte(`{"host":"runner-b","goos":"linux","goarch":"amd64"}`)},
	}
	got := qualificationMeasurements([]byte(`{"host":"runner-a","goos":"linux","goarch":"amd64"}`), items)
	require.Len(t, got, 1)
	require.Equal(t, "provider_busy", got[0].FaultProfile)
}
