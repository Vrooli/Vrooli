package whisperinfo_test

import (
	"testing"

	"audio-tools/internal/stt/whisperinfo"
	"audio-tools/internal/testutil/mocks"
)

func TestEnvClient_Unset(t *testing.T) {
	c := whisperinfo.NewWith(mocks.NewFakeEnv(nil))
	got := c.CurrentModel()
	if got.ModelID != whisperinfo.ModelUnknown {
		t.Errorf("ModelID=%q want %q", got.ModelID, whisperinfo.ModelUnknown)
	}
}

func TestEnvClient_SetFromEnv(t *testing.T) {
	c := whisperinfo.NewWith(mocks.NewFakeEnv(map[string]string{
		"AUDIO_WHISPER_MODEL":  "medium",
		"AUDIO_WHISPER_ENGINE": "faster_whisper",
	}))
	got := c.CurrentModel()
	if got.ModelID != "whisper-medium" {
		t.Errorf("ModelID=%q", got.ModelID)
	}
	if got.Engine != "faster_whisper" {
		t.Errorf("Engine=%q", got.Engine)
	}
}

func TestEnvClient_NilEnvDefault(t *testing.T) {
	c := whisperinfo.NewWith(nil)
	// Should not panic; OS env likely unset in test environment.
	_ = c.CurrentModel()
}
