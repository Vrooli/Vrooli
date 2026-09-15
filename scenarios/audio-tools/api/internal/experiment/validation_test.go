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
