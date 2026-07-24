package speaker

import (
	"testing"

	sttpipeline "audio-tools/internal/stt/pipeline"
)

func TestDisabledConfigurationsDoNotConstructSpeakerStages(t *testing.T) {
	if got := NewIsolation(sttpipeline.SpeakerConfig{}, nil, nil); got != nil {
		t.Fatalf("NewIsolation() = %T; want nil for disabled config", got)
	}
	if got := NewExtraction(sttpipeline.SpeakerConfig{Enabled: true, ExtractionEnabled: true, ProfileIDs: []string{"profile-a"}}, nil); got != nil {
		t.Fatalf("NewExtraction() = %T; want nil without a speaker client", got)
	}
}
