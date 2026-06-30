package bootstrap

import (
	"testing"

	inteval "audio-tools/internal/eval"

	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

func TestBuildSpeakerConditions_AblationGrid(t *testing.T) {
	got := buildSpeakerConditions(&experimentv1.SpeakerExperimentRecipe{
		TargetProfileId:     "target-voice",
		ExtractionEnabled:   true,
		VerificationEnabled: true,
		VerificationMode:    sttv1.SpeakerMode_SPEAKER_MODE_ADVISORY,
		Threshold:           0.72,
		AblationEnabled:     true,
	})

	wantIDs := []string{
		"extract_off_verify_off",
		"extract_on_verify_off",
		"extract_off_verify_on",
		"extract_on_verify_on",
	}
	if len(got) != len(wantIDs) {
		t.Fatalf("condition count = %d, want %d", len(got), len(wantIDs))
	}
	for i, want := range wantIDs {
		if got[i].ID != want {
			t.Fatalf("condition[%d].ID = %q, want %q", i, got[i].ID, want)
		}
		if got[i].Config.ProfileIDs[0] != "target-voice" {
			t.Fatalf("condition[%d] profile = %v", i, got[i].Config.ProfileIDs)
		}
		if got[i].Config.Threshold != 0.72 {
			t.Fatalf("condition[%d] threshold = %.2f", i, got[i].Config.Threshold)
		}
	}
	if !got[1].ExtractionEnabled || got[1].VerificationEnabled {
		t.Fatalf("extract_on_verify_off toggles = extraction %t verification %t", got[1].ExtractionEnabled, got[1].VerificationEnabled)
	}
	if got[2].ExtractionEnabled || !got[2].VerificationEnabled {
		t.Fatalf("extract_off_verify_on toggles = extraction %t verification %t", got[2].ExtractionEnabled, got[2].VerificationEnabled)
	}
}

func TestBuildSpeakerConditions_SkipsMissingTargetProfile(t *testing.T) {
	got := buildSpeakerConditions(&experimentv1.SpeakerExperimentRecipe{
		VerificationEnabled: true,
	})
	if len(got) != 1 {
		t.Fatalf("condition count = %d, want 1", len(got))
	}
	if !got[0].Skipped {
		t.Fatalf("condition should be skipped without a target profile")
	}
	if got[0].Note == "" {
		t.Fatalf("skipped condition should explain why")
	}
}

func TestAppendUniqueReportWarnings_DeduplicatesRepeatedConditionWarnings(t *testing.T) {
	warning := inteval.ReportWarning{Severity: "warning", Code: "tiny_corpus", Message: "Only 1 clips were evaluated"}
	got := appendUniqueReportWarnings([]inteval.ReportWarning{warning}, warning, inteval.ReportWarning{
		Severity: "warning",
		Code:     "short_audio",
		Message:  "The evaluated audio totals 2.7 seconds",
	})

	if len(got) != 2 {
		t.Fatalf("warning count = %d, want 2: %#v", len(got), got)
	}
	if got[0].Code != "tiny_corpus" || got[1].Code != "short_audio" {
		t.Fatalf("warnings = %#v", got)
	}
}
