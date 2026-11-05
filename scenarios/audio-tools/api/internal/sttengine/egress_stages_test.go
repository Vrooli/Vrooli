package sttengine_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/stt/egress"
	"audio-tools/internal/stt/egress/mocks"
	"audio-tools/internal/sttengine"
)

// stageNames builds a gate from the derived stages and returns the ordered
// stage names — the observable shape of the manifest-driven derivation.
func stageNames(stages []egress.Stage) []string {
	return egress.NewGate(stages...).Stages()
}

func TestEgressStages_ManifestDrivenStageSet(t *testing.T) {
	reg := sttengine.Default() // whisper-local (confidence signals) + kyutai (none)
	iso := &mocks.FakeSpeakerIsolation{}

	// Whisper: text (hallucination) + signal (confidence) + audio (speaker).
	whisper := stageNames(reg.EgressStages("whisper-local", sttengine.EgressParams{
		HallucinationFilterEnabled: true,
		IsHallucination:            func(string) bool { return false },
		SpeakerIsolation:           iso,
	}))
	require.Equal(t, []string{"hallucination", "confidence", "speaker"}, whisper)

	// Kyutai declares no confidence signals -> signal-domain stage is skipped.
	kyutai := stageNames(reg.EgressStages("kyutai", sttengine.EgressParams{
		HallucinationFilterEnabled: true,
		IsHallucination:            func(string) bool { return false },
		SpeakerIsolation:           iso,
	}))
	require.Equal(t, []string{"hallucination", "speaker"}, kyutai)

	// No isolation wired -> no audio-domain stage; hallucination filter off ->
	// no text-domain stage. Whisper keeps only the signal stage.
	none := stageNames(reg.EgressStages("whisper-local", sttengine.EgressParams{
		HallucinationFilterEnabled: false,
	}))
	require.Equal(t, []string{"confidence"}, none)
}
