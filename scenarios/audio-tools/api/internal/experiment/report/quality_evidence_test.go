//go:build audio_lab

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
			Tags:           []string{"accent_" + []string{"uk", "us", "south_african"}[(i/3)%3]},
			CorpusRevision: "voice-pilot-en-v1", LicenseReference: "license-1",
			SpeakerPseudonym: fmt.Sprintf("speaker-%d", i%6), Split: intcorpus.SplitHeldOut,
			NoiseCondition: condition, SNRDB: snr, ContentSHA256: fmt.Sprintf("hash-%d", i),
			ReferenceReview: &intcorpus.ReferenceReview{Reviewer: "auditor", Protocol: "listen-v1", ReviewedAt: "2026-06-29T12:00:00Z", ReferenceSHA256: intcorpus.ReferenceDigest(strings.Repeat("one two three four five six seven eight nine ten ", 5))},
		})
	}
	return clips
}

func TestQualityEvidenceFailsClosedForUnreviewedReferences(t *testing.T) {
	clips := qualityFixtureClips(intcorpus.RequiredQualityClips)
	clips[0].ReferenceReview = nil
	evidence := QualityEvidenceForReport(clips, qualityFixtureReport(clips), "build-1")
	require.Len(t, evidence, 1)
	require.False(t, evidence[0].Passed)
	require.Contains(t, string(evidence[0].MachineJSON), "has no human-reviewed reference")
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
	quality := machine["quality_measurement"].(map[string]any)
	require.Equal(t, qualityMeasurementSchema, quality["schema_version"])
	metrics := quality["metrics"].([]any)
	require.Len(t, metrics, 12)
	names := make([]string, 0, len(metrics))
	for _, raw := range metrics {
		names = append(names, raw.(map[string]any)["name"].(string))
	}
	require.Equal(t, []string{
		"clean", "noisy_20db", "noisy_10db",
		"accent_noise:uk:clean", "accent_noise:uk:20db", "accent_noise:uk:10db",
		"accent_noise:us:clean", "accent_noise:us:20db", "accent_noise:us:10db",
		"accent_noise:south_african:clean", "accent_noise:south_african:20db", "accent_noise:south_african:10db",
	}, names)
}

func TestQualityEvidenceFailsAccentSubgroupWhileAggregateNoisePasses(t *testing.T) {
	clips := qualityFixtureClips(intcorpus.RequiredQualityClips)
	report := qualityFixtureReport(clips)
	for i, clip := range clips {
		if (i/3)%3 == 0 && qualityConditionGroup(clip) == "clean" {
			report.PerStrategy[0].PerClip[i].WER.EditCounts.Substitutions = 15
			break
		}
	}
	evidence := QualityEvidenceForReport(clips, report, "build-1")
	require.Len(t, evidence, 1)
	require.False(t, evidence[0].Passed)
	var machine map[string]any
	require.NoError(t, json.Unmarshal(evidence[0].MachineJSON, &machine))
	quality := machine["quality_measurement"].(map[string]any)
	metrics := quality["metrics"].([]any)
	byName := map[string]map[string]any{}
	for _, raw := range metrics {
		metric := raw.(map[string]any)
		byName[metric["name"].(string)] = metric
	}
	require.Less(t, byName["clean"]["wer"].(float64), maxQualityCleanWER)
	require.Greater(t, byName["accent_noise:uk:clean"]["wer"].(float64), maxQualityCleanWER)
	require.Contains(t, quality["reasons"], "accent_noise:uk:clean WER 0.1071 exceeds 0.1000")
}

func TestQualityEvidenceFailsClosedWhenAccentNoiseSubgroupIsMissing(t *testing.T) {
	clips := qualityFixtureClips(intcorpus.RequiredQualityClips)
	for i := range clips {
		if (i/3)%3 == 0 && qualityConditionGroup(clips[i]) == "clean" {
			clips[i].Tags = nil
		}
	}
	evidence := QualityEvidenceForReport(clips, qualityFixtureReport(clips), "build-1")
	require.Len(t, evidence, 1)
	require.False(t, evidence[0].Passed)
	var machine map[string]any
	require.NoError(t, json.Unmarshal(evidence[0].MachineJSON, &machine))
	quality := machine["quality_measurement"].(map[string]any)
	require.Contains(t, quality["reasons"], "report has no measured accent_noise:uk:clean subgroup")
	metrics := quality["metrics"].([]any)
	for _, raw := range metrics {
		metric := raw.(map[string]any)
		if metric["name"] == "accent_noise:uk:clean" {
			require.Equal(t, false, metric["available"])
			require.NotContains(t, metric, "wer")
		}
	}
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

func TestQualityEvidenceResolvesCanonicalModelFromRuntimeID(t *testing.T) {
	clips := qualityFixtureClips(intcorpus.RequiredQualityClips)
	report := qualityFixtureReport(clips)
	report.PerStrategy[0].EngineID = "sherpa-streaming"
	report.PerStrategy[0].ModelID = "sherpa-onnx/streaming-zipformer-en-2023-06-26"
	evidence := QualityEvidenceForReport(clips, report, "build-1")
	require.Len(t, evidence, 1)
	require.True(t, evidence[0].Passed, evidence[0].MachineJSON)
	require.Equal(t, "sherpa-zipformer-en-2023-06-26", evidence[0].ModelID)
	var machine map[string]any
	require.NoError(t, json.Unmarshal(evidence[0].MachineJSON, &machine))
	cell := machine["provider_cell"].(map[string]any)
	require.Equal(t, "sherpa-zipformer-en-2023-06-26", cell["model_id"])
	require.NotContains(t, string(evidence[0].MachineJSON), "model id is not declared in the model catalog")
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
		Reasons:                   []string{"max contiguous dropped span is 27 words (threshold 4)"},
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
