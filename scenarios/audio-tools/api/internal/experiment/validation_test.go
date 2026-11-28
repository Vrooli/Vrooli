//go:build audio_lab

package experiment

import (
	"errors"
	"testing"

	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
)

func TestValidateRecipeReturnsSpeakerProfilePolicyError(t *testing.T) {
	err := ValidateRecipe(&experimentv1.ExperimentRecipe{Speaker: &experimentv1.SpeakerExperimentRecipe{
		ExtractionEnabled: true,
	}})
	if !errors.Is(err, ErrSpeakerProfileRequired) {
		t.Fatalf("ValidateRecipe() error = %v, want ErrSpeakerProfileRequired", err)
	}
}

func TestValidateRecipeRejectsRealtimeTailEvidence(t *testing.T) {
	err := ValidateRecipe(&experimentv1.ExperimentRecipe{
		LatencyTailSeconds: 1,
		Cells: []*experimentv1.EvaluationCell{{
			EngineId: "engine", Strategy: "batch", ReplayLane: experimentv1.ReplayLane_REPLAY_LANE_REALTIME, RepeatCount: 1,
		}},
	})
	if err == nil {
		t.Fatal("ValidateRecipe() unexpectedly accepted realtime tail evidence")
	}
}

func TestValidateRecipeAcceptsOnlyDeclaredReportRetentionModes(t *testing.T) {
	for _, mode := range []experimentv1.ReportRetentionMode{
		experimentv1.ReportRetentionMode_REPORT_RETENTION_MODE_UNSPECIFIED,
		experimentv1.ReportRetentionMode_REPORT_RETENTION_MODE_PRESERVE_TRANSCRIPTS,
		experimentv1.ReportRetentionMode_REPORT_RETENTION_MODE_METRICS_ONLY,
	} {
		if err := ValidateRecipe(&experimentv1.ExperimentRecipe{ReportRetentionMode: mode}); err != nil {
			t.Errorf("ValidateRecipe(mode=%s) error = %v", mode, err)
		}
	}
	if err := ValidateRecipe(&experimentv1.ExperimentRecipe{ReportRetentionMode: experimentv1.ReportRetentionMode(99)}); err == nil {
		t.Fatal("ValidateRecipe accepted an unknown report retention mode")
	}
}
