package report

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	intcorpus "audio-tools/internal/corpus"
	inteval "audio-tools/internal/eval"
)

func qualityFixtureClips(n int) []intcorpus.Clip {
	clips := make([]intcorpus.Clip, 0, n)
	for i := 0; i < n; i++ {
		condition := intcorpus.NoiseClean
		snr := float64(0)
		if i%3 == 1 {
			condition, snr = intcorpus.NoiseNoisy, 20
		} else if i%3 == 2 {
			condition, snr = intcorpus.NoiseNoisy, 10
		}
		clips = append(clips, intcorpus.Clip{
			ID: fmt.Sprintf("clip-%d", i), ReferenceText: strings.Repeat("one two three four five six seven eight nine ten ", 5),
			CorpusRevision: "voice-pilot-en-v1", LicenseReference: "license-1",
			SpeakerPseudonym: fmt.Sprintf("speaker-%d", i%6), Split: intcorpus.SplitHeldOut,
			NoiseCondition: condition, SNRDB: snr, ContentSHA256: fmt.Sprintf("hash-%d", i),
		})
	}
	return clips
}

func qualityFixtureReport(clips []intcorpus.Clip) inteval.EvalReport {
	perClip := make([]inteval.ClipResult, 0, len(clips))
	for _, clip := range clips {
		perClip = append(perClip, inteval.ClipResult{ClipID: clip.ID, Reference: clip.ReferenceText, Hypothesis: clip.ReferenceText, WER: inteval.WERResult{RefWords: 20}})
	}
	return inteval.EvalReport{QualityMeasured: true, PerStrategy: []inteval.StrategyReport{{
		Strategy: sttchain.StrategyVADSegment, EngineID: "whisper-local", ModelID: "whisper-medium", PolicyProfile: "default", ReplayLane: "batch", PerClip: perClip,
		Safety: inteval.SafetyGateReport{Passed: true, RetractionFree: true, DroppedSpanFree: true, DroppedSpanThresholdWords: inteval.DefaultDroppedSpanThresholdWords},
	}}}
}

func TestQualityEvidenceForReportPassesOnlyCompleteQualifiedCell(t *testing.T) {
	clips := qualityFixtureClips(intcorpus.RequiredQualityClips)
	evidence := QualityEvidenceForReport(clips, qualityFixtureReport(clips), "build-1")
	require.Len(t, evidence, 1)
	require.True(t, evidence[0].Passed)
	var machine map[string]any
	require.NoError(t, json.Unmarshal(evidence[0].MachineJSON, &machine))
	acceptance := machine["acceptance"].(map[string]any)
	claim := acceptance["corpus-quality"].(map[string]any)
	require.Equal(t, true, claim["in_band"])
	require.Equal(t, "UNQUALIFIED", machine["device_profile"], "batch replay must not infer a native D2 profile")
}

func TestQualityEvidenceUsesCanonicalStrategyForExplicitCell(t *testing.T) {
	clips := qualityFixtureClips(intcorpus.RequiredQualityClips)
	report := qualityFixtureReport(clips)
	report.PerStrategy[0].Strategy = "whisper-local/vad_segment/realtime"
	report.PerStrategy[0].BaseStrategy = "vad_segment"
	report.PerStrategy[0].ReplayLane = "realtime"
	evidence := QualityEvidenceForReport(clips, report, "build-1")
	require.Len(t, evidence, 1)
	require.Equal(t, "vad_segment", evidence[0].Strategy)
	var machine map[string]any
	require.NoError(t, json.Unmarshal(evidence[0].MachineJSON, &machine))
	cell := machine["provider_cell"].(map[string]any)
	require.Equal(t, "vad_segment", cell["strategy"])
	require.Equal(t, "D1", machine["device_profile"])
}

func TestQualityEvidenceForReportScopesCohortToMeasuredClips(t *testing.T) {
	measured := qualityFixtureClips(intcorpus.RequiredQualityClips)
	legacy := intcorpus.Clip{ID: "legacy", ReferenceText: "legacy", CorpusRevision: "old"}
	all := append(append([]intcorpus.Clip{}, measured...), legacy)
	evidence := QualityEvidenceForReport(all, qualityFixtureReport(measured), "build-1")
	require.Len(t, evidence, 1)
	require.True(t, evidence[0].Passed, evidence[0].MachineJSON)
	var machine map[string]any
	require.NoError(t, json.Unmarshal(evidence[0].MachineJSON, &machine))
	quality := machine["quality_measurement"].(map[string]any)
	corpus := quality["corpus"].(map[string]any)
	require.Equal(t, float64(intcorpus.RequiredQualityClips), corpus["clip_count"])
}

func TestQualityEvidenceRetainsFailedReceiptForIncompleteCorpus(t *testing.T) {
	clips := qualityFixtureClips(1)
	evidence := QualityEvidenceForReport(clips, qualityFixtureReport(clips), "unknown")
	require.Len(t, evidence, 1)
	require.False(t, evidence[0].Passed)
	var machine map[string]any
	require.NoError(t, json.Unmarshal(evidence[0].MachineJSON, &machine))
	claim := machine["acceptance"].(map[string]any)["corpus-quality"].(map[string]any)
	require.Equal(t, false, claim["in_band"])
	require.NotEmpty(t, machine["quality_measurement"])
}

func TestQualityEvidenceRejectsSafetyFailureEvenWhenWERIsInBand(t *testing.T) {
	clips := qualityFixtureClips(intcorpus.RequiredQualityClips)
	report := qualityFixtureReport(clips)
	report.PerStrategy[0].Safety = inteval.SafetyGateReport{
		RetractionFree: true, DroppedSpanFree: false, MaxDroppedSpanWords: 27,
		DroppedSpanThresholdWords: inteval.DefaultDroppedSpanThresholdWords,
		Reasons: []string{"max contiguous dropped span is 27 words (threshold 4)"},
	}
	evidence := QualityEvidenceForReport(clips, report, "build-1")
	require.Len(t, evidence, 1)
	require.False(t, evidence[0].Passed)
	var machine map[string]any
	require.NoError(t, json.Unmarshal(evidence[0].MachineJSON, &machine))
	claim := machine["acceptance"].(map[string]any)["corpus-quality"].(map[string]any)
	require.Equal(t, false, claim["in_band"])
	require.Equal(t, false, claim["negative_controls_passed"])
	quality := machine["quality_measurement"].(map[string]any)
	require.Contains(t, quality["reasons"], "report safety gate failed")
}

func TestQualityEvidenceRejectsReportsWithoutSafetyObservation(t *testing.T) {
	clips := qualityFixtureClips(intcorpus.RequiredQualityClips)
	report := qualityFixtureReport(clips)
	report.PerStrategy[0].Safety = inteval.SafetyGateReport{}
	evidence := QualityEvidenceForReport(clips, report, "build-1")
	require.Len(t, evidence, 1)
	require.False(t, evidence[0].Passed)
	var machine map[string]any
	require.NoError(t, json.Unmarshal(evidence[0].MachineJSON, &machine))
	claim := machine["acceptance"].(map[string]any)["corpus-quality"].(map[string]any)
	require.Equal(t, false, claim["in_band"])
}
