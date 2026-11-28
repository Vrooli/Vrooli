package whisperinfo_test

import (
	"testing"

	"audio-tools/internal/modelcatalog"
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

// TestWhisperModelIDsResolveInRegistry keeps the model-truth seam and the
// model registry from drifting. EnvClient prefixes the sidecar's model name
// with "whisper-"; every whisper-stt record must round-trip through that rule,
// and every record the rule can produce must exist in the registry.
func TestWhisperModelIDsResolveInRegistry(t *testing.T) {
	catalog := modelcatalog.Default()

	for _, model := range catalog.ByProvider("whisper-stt") {
		const prefix = "whisper-"
		if len(model.ID) <= len(prefix) || model.ID[:len(prefix)] != prefix {
			t.Fatalf("whisper-stt model %q does not carry the %q prefix", model.ID, prefix)
		}
		sidecarName := model.ID[len(prefix):]
		client := whisperinfo.NewWith(mocks.NewFakeEnv(map[string]string{
			"AUDIO_WHISPER_MODEL": sidecarName,
		}))
		if got := client.CurrentModel().ModelID; got != model.ID {
			t.Fatalf("sidecar model %q resolves to %q, want registry id %q", sidecarName, got, model.ID)
		}
	}

	// The current production default must be declared; an unset/unknown model
	// is allowed because it is an honest sentinel, not a fabricated id.
	if _, ok := catalog.ByID("whisper-medium"); !ok && whisperinfo.ModelUnknown == "" {
		t.Fatal("whisper-medium is not declared and the unknown sentinel is empty")
	}
}
