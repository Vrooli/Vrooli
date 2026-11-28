package capabilities

import (
	"testing"

	"audio-tools/internal/providercatalog"
)

func TestCanonicalProviderID(t *testing.T) {
	cases := map[string]string{
		"whisper-local":    "whisper-stt",
		"kyutai":           "kyutai-stt",
		"sherpa-streaming": "sherpa-stt",
		"kokoro-local":     "kokoro-tts",
		"ollama-local":     "ollama",
		"deepgram":         "deepgram",
		"openai-whisper":   "openai-whisper",
		"virtual-replay":   "virtual-replay",
		"lpbs":             "lpbs",
		"some-new-runtime": "some-new-runtime",
		"":                 "",
	}
	for raw, want := range cases {
		if got := CanonicalProviderID(raw); got != want {
			t.Errorf("CanonicalProviderID(%q) = %q; want %q", raw, got, want)
		}
	}
}

// TestCanonicalProviderIDIsDerivedFromCatalog proves all three runtime identity
// forms resolve from declaration: stable IDs, engine IDs, and adapter aliases.
func TestCanonicalProviderIDIsDerivedFromCatalog(t *testing.T) {
	for _, provider := range providercatalog.All() {
		identities := append([]string{provider.ID}, provider.EngineIDs...)
		identities = append(identities, provider.RuntimeAliases...)
		for _, raw := range identities {
			if id, ok := providercatalog.ProviderIDForRuntimeIdentity(raw); !ok || id != provider.ID {
				t.Errorf("ProviderIDForRuntimeIdentity(%q) = %q, %v; want %q, true", raw, id, ok, provider.ID)
			}
			if got := CanonicalProviderID(raw); got != provider.ID {
				t.Errorf("CanonicalProviderID(%q) = %q; want catalogue owner %q", raw, got, provider.ID)
			}
		}
	}
}
