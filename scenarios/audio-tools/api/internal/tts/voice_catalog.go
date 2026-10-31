package tts

// CanonicalVoiceIDs is the exhaustive set of canonical voice IDs that every
// TTS adapter MUST declare a mapping for.
var CanonicalVoiceIDs = []string{
	"voice.feminine.warm",
	"voice.feminine.neutral",
	"voice.masculine.warm",
	"voice.masculine.neutral",
	"voice.neutral.default",
}

// AdapterVoiceMap is the type each registered TTS adapter exposes for
// startup verification. Keys are canonical voice IDs; values are the
// adapter's backend voice identifier.
type AdapterVoiceMap map[string]string

// AdapterCatalogEntry is one registered adapter with its declared mapping.
type AdapterCatalogEntry struct {
	TierProvider string // e.g., "byok:openai-tts", "local:kokoro-local"
	Mapping      AdapterVoiceMap
}

// VerifyCatalog asserts every registered adapter declares a mapping for
// every canonical voice. Returns a structured error naming the first
// adapter/voice gap; callers (main.go startup) MUST fail-fast on a
// non-nil error rather than silently launching with incomplete coverage.
func VerifyCatalog(entries []AdapterCatalogEntry) error {
	for _, e := range entries {
		for _, canonical := range CanonicalVoiceIDs {
			if _, ok := e.Mapping[canonical]; !ok {
				return &MissingCanonicalVoiceError{
					Adapter:        e.TierProvider,
					CanonicalVoice: canonical,
				}
			}
		}
	}
	return nil
}

// MissingCanonicalVoiceError is returned by VerifyCatalog when an adapter
// lacks a mapping for a canonical voice.
type MissingCanonicalVoiceError struct {
	Adapter        string
	CanonicalVoice string
}

func (e *MissingCanonicalVoiceError) Error() string {
	return "audio-tools/tts: adapter " + e.Adapter + " has no mapping for canonical voice " + e.CanonicalVoice
}

// LocalKokoroMapping is the canonical->backend mapping for the Local Kokoro
// adapter. Exposed here so main.go can register it.
var LocalKokoroMapping = AdapterVoiceMap{
	"voice.feminine.warm":     "af_bella",
	"voice.feminine.neutral":  "af_sarah",
	"voice.masculine.warm":    "am_adam",
	"voice.masculine.neutral": "am_michael",
	"voice.neutral.default":   "af_nicole",
}
