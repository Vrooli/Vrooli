package byok

import (
	"testing"

	"audio-tools/internal/providercatalog"
)

func TestRegistriesContents(t *testing.T) {
	r := NewRegistries()
	if _, ok := r.STT["openai-whisper"]; !ok {
		t.Fatal("missing openai-whisper")
	}
	if _, ok := r.STT["deepgram"]; !ok {
		t.Fatal("missing deepgram")
	}
	if _, ok := r.TTS["openai-tts"]; !ok {
		t.Fatal("missing openai-tts")
	}
	if _, ok := r.TTS["elevenlabs"]; !ok {
		t.Fatal("missing elevenlabs")
	}
	if _, ok := r.Summarize["openrouter"]; !ok {
		t.Fatal("missing openrouter")
	}
}

func TestRegistryUsesDeclaredProtocolAndPreservesProviderIdentity(t *testing.T) {
	declared, ok := providercatalog.Default().ByID("openai-whisper")
	if !ok {
		t.Fatal("openai-whisper missing from provider catalog")
	}
	declared.ID = "synthetic-whisper-provider"
	registries, err := NewRegistriesForProviders([]providercatalog.Provider{declared})
	if err != nil {
		t.Fatal(err)
	}
	adapter := registries.STT[declared.ID]
	if adapter == nil {
		t.Fatal("declaration did not register STT adapter")
	}
	if got := adapter.ID(); got != declared.ID {
		t.Fatalf("adapter identity = %q, want canonical provider ID %q", got, declared.ID)
	}
}
