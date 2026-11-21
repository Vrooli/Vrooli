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

// TestCanonicalProviderIDIsDerivedFromCatalog refuses to let a hand table
// return: every engine ID and runtime alias the catalogue declares must
// canonicalize to its owner without a code edit here.
func TestCanonicalProviderIDIsDerivedFromCatalog(t *testing.T) {
	for _, provider := range providercatalog.All() {
		for _, raw := range append(append([]string(nil), provider.EngineIDs...), provider.RuntimeAliases...) {
			if got := CanonicalProviderID(raw); got != provider.ID {
				t.Errorf("CanonicalProviderID(%q) = %q; want catalogue owner %q", raw, got, provider.ID)
			}
		}
	}
}
