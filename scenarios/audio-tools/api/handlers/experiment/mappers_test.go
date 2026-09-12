package experiment

import (
	"testing"
	"time"

	intexp "audio-tools/internal/experiment"

	"github.com/stretchr/testify/require"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
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
