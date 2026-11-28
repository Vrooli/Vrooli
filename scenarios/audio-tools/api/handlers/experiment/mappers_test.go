//go:build audio_lab

package experiment

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	intexp "audio-tools/internal/experiment"

	"github.com/stretchr/testify/require"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	"google.golang.org/protobuf/proto"
)

func TestRunToProto_DecodesPersistedProviderCellIdentity(t *testing.T) {
	got := runToProto(intexp.Run{
		ID:            "run-1",
		ExperimentID:  "exp-1",
		Strategy:      "kyutai/passthrough/realtime",
		ConditionJSON: []byte(`{"engine_id":"kyutai","policy_profile":"","replay_lane":"realtime","fault_profile":""}`),
		CreatedAt:     time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC),
	})
	require.Equal(t, "kyutai", got.GetEngineId())
	require.Equal(t, experimentv1.ReplayLane_REPLAY_LANE_REALTIME, got.GetReplayLane())
	require.Empty(t, got.GetPolicyProfile())
	require.Empty(t, got.GetFaultProfile())
}

func TestQualificationEvidenceProviderTierRoundTripsAsTypedProvenance(t *testing.T) {
	input := &experimentv1.QualificationEvidence{
		EngineId: "same-provider", ModelId: "model", Strategy: "stream",
		ProviderTier: commonv1.ProviderTier_PROVIDER_TIER_LOCAL,
		Kind:         experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_BROWSER_PRODUCT_PATH,
		ArtifactRef:  "bas:route-tier", MachineJson: `{}`,
	}
	domain, err := qualificationEvidenceFromProto(input)
	require.NoError(t, err)
	require.Equal(t, "local", domain.ProviderTier)
	require.Equal(t, commonv1.ProviderTier_PROVIDER_TIER_LOCAL, qualificationEvidenceToProto(domain).GetProviderTier())
}

func TestQualificationEvidenceFromProto_RequiresKnownFaultAndArtifact(t *testing.T) {
	valid := &experimentv1.QualificationEvidence{
		EngineId: "kyutai", ModelId: "kyutai/stt-1b-en_fr", Strategy: "passthrough",
		Kind:         experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_FAULT,
		FaultProfile: "provider_busy", ArtifactRef: "bas:run-1", MachineJson: `{}`,
	}
	got, err := qualificationEvidenceFromProto(valid)
	require.NoError(t, err)
	require.Equal(t, "fault", got.Kind)

	valid.FaultProfile = "made_up"
	_, err = qualificationEvidenceFromProto(valid)
	require.ErrorContains(t, err, "not required")

	valid.FaultProfile = "provider_busy"
	valid.ArtifactRef = ""
	_, err = qualificationEvidenceFromProto(valid)
	require.ErrorContains(t, err, "artifact_ref is required")
}

func TestQualificationEvidenceFromProto_RetainsCorpusQualityKind(t *testing.T) {
	got, err := qualificationEvidenceFromProto(&experimentv1.QualificationEvidence{
		EngineId: "whisper-local", ModelId: "whisper-medium", Strategy: "vad_segment",
		Kind:        experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_CORPUS_QUALITY,
		ArtifactRef: "experiment-quality:test", MachineJson: `{}`,
	})
	require.NoError(t, err)
	require.Equal(t, "corpus_quality", got.Kind)
	require.Equal(t, experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_CORPUS_QUALITY, qualificationKindToProto(got.Kind))
}

func TestQualificationEvidenceModelMemoryKindRoundTrips(t *testing.T) {
	input := &experimentv1.QualificationEvidence{
		Id: "memory-run-1", EngineId: "sherpa-streaming", ModelId: "sherpa-zipformer-en-2023-06-26", Strategy: "streaming",
		Kind:   experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_MODEL_MEMORY,
		Passed: true, ArtifactRef: "audio-tools:model-memory:run-1", MachineJson: validModelMemoryJSON(),
	}
	domain, err := qualificationEvidenceFromProto(input)
	require.NoError(t, err)
	require.Equal(t, "model_memory", domain.Kind)
	require.Equal(t, input.MachineJson, string(domain.MachineJSON))
	require.Equal(t, input.GetKind(), qualificationEvidenceToProto(domain).GetKind())
	require.Equal(t, experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_MODEL_MEMORY, qualificationKindToProto(domain.Kind))
}

func TestQualificationEvidenceModelMemoryRejectsInvalidMeasurements(t *testing.T) {
	base := &experimentv1.QualificationEvidence{
		EngineId: "sherpa-streaming", ModelId: "sherpa-zipformer-en-2023-06-26", Strategy: "streaming",
		Kind:   experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_MODEL_MEMORY,
		Passed: true, ArtifactRef: "audio-tools:model-memory:run-1", MachineJson: validModelMemoryJSON(),
	}
	tests := []struct {
		name string
		edit func(map[string]any)
		want string
	}{
		{name: "malformed json", want: "must be valid JSON"},
		{name: "missing peak", edit: func(v map[string]any) { delete(v["execution"].(map[string]any), "peak_memory_bytes") }, want: "positive budget and measured peak"},
		{name: "zero peak", edit: func(v map[string]any) { v["execution"].(map[string]any)["peak_memory_bytes"] = 0 }, want: "positive budget and measured peak"},
		{name: "over budget", edit: func(v map[string]any) {
			v["execution"].(map[string]any)["peak_memory_bytes"] = int64(3 * 1024 * 1024 * 1024)
		}, want: "in-budget measured peak"},
		{name: "missing integrity verdict", edit: func(v map[string]any) {
			delete(v["acceptance"].(map[string]any)["memory-budget"].(map[string]any), "measurement_integrity_passed")
		}, want: "passing integrity acceptance"},
		{name: "uppercase digest", edit: func(v map[string]any) { v["model"].(map[string]any)["artifact_sha256"] = strings.Repeat("A", 64) }, want: "lowercase 64-character"},
		{name: "unknown field", edit: func(v map[string]any) { v["unexpected"] = true }, want: "unknown field"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := proto.Clone(base).(*experimentv1.QualificationEvidence)
			if tt.name == "malformed json" {
				candidate.MachineJson = `{broken`
			} else {
				var machine map[string]any
				require.NoError(t, json.Unmarshal([]byte(candidate.MachineJson), &machine))
				tt.edit(machine)
				raw, err := json.Marshal(machine)
				require.NoError(t, err)
				candidate.MachineJson = string(raw)
			}
			_, err := qualificationEvidenceFromProto(candidate)
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func validModelMemoryJSON() string {
	return `{"schema_version":1,"provider_cell":{"engine_id":"sherpa-streaming","model_id":"sherpa-zipformer-en-2023-06-26","capability":"stt","strategy":"streaming"},"model":{"runtime":"sherpa-onnx","runtime_model_id":"sherpa-onnx/streaming-zipformer-en-2023-06-26","artifact_resource":"sherpa-onnx","artifact_name":"streaming-zipformer-en-2023-06-26","artifact_sha256":"` + strings.Repeat("a", 64) + `","runtime_sha256":"` + strings.Repeat("b", 64) + `"},"execution":{"lane":"cpu","os":"linux","arch":"amd64","accelerator":"none","memory_budget_bytes":2147483648,"peak_memory_bytes":123456789,"peak_memory_source":"cgroup-v2-memory.peak","inference_passed":true},"acceptance":{"memory-budget":{"in_band":true,"measured":true,"measurement_integrity_passed":true}}}`
}

func TestQualificationEvidenceFromProto_AcceptanceCompositeNeedsNoIntervalMetrics(t *testing.T) {
	got, err := qualificationEvidenceFromProto(&experimentv1.QualificationEvidence{
		EngineId: "acceptance-join", ModelId: "portable-voice-v2", Strategy: "composite",
		Kind:   experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_ACCEPTANCE_COMPOSITE,
		Passed: true, ArtifactRef: "audio-tools:acceptance-join:test", MachineJson: `{"target_revision":"portable-voice-v2","method":"audio-tools-acceptance-v2","acceptance":{}}`,
	})
	require.NoError(t, err)
	require.Equal(t, "acceptance_composite", got.Kind)
	require.Equal(t, experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_ACCEPTANCE_COMPOSITE, qualificationKindToProto(got.Kind))
}

func TestQualificationEvidenceFromProto_RequiresExplicitDeliveryAccounting(t *testing.T) {
	valid := &experimentv1.QualificationEvidence{
		EngineId: "kyutai", ModelId: "kyutai/stt-1b-en_fr", Strategy: "passthrough",
		Kind:   experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_INTERVAL_ACCOUNTING,
		Passed: true, ArtifactRef: "bas:coverage", MachineJson: `{"host":"runner","goos":"linux","goarch":"amd64","all_intervals_accounted":true,"duplicate_committed_segments":0,"silent_terminal_outcomes":0}`,
	}
	_, err := qualificationEvidenceFromProto(valid)
	require.NoError(t, err)

	valid.MachineJson = `{"host":"runner","goos":"linux","goarch":"amd64"}`
	_, err = qualificationEvidenceFromProto(valid)
	require.ErrorContains(t, err, "all_intervals_accounted")

	valid.MachineJson = `{"host":"runner","goos":"linux","goarch":"amd64","all_intervals_accounted":true,"duplicate_committed_segments":1,"silent_terminal_outcomes":0}`
	_, err = qualificationEvidenceFromProto(valid)
	require.ErrorContains(t, err, "zero duplicate commits")
}

func TestQualificationEvidenceFromProto_ValidatesMachineAndFaultPolicy(t *testing.T) {
	base := &experimentv1.QualificationEvidence{
		EngineId: "kyutai", ModelId: "kyutai/stt-1b-en_fr", Strategy: "passthrough",
		Kind:        experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_BOUNDED_RECOVERY,
		ArtifactRef: "bas:run-1", MachineJson: `{}`,
	}

	badJSON := proto.Clone(base).(*experimentv1.QualificationEvidence)
	badJSON.MachineJson = `{invalid}`
	_, err := qualificationEvidenceFromProto(badJSON)
	require.EqualError(t, err, "qualification evidence machine_json must be valid JSON")

	wrongKindFault := proto.Clone(base).(*experimentv1.QualificationEvidence)
	wrongKindFault.FaultProfile = "provider_busy"
	_, err = qualificationEvidenceFromProto(wrongKindFault)
	require.EqualError(t, err, "fault_profile is valid only for fault qualification evidence")
}
