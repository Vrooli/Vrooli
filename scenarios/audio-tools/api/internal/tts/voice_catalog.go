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

// Stable adapter ids. These are the same strings the provider catalog declares
// as adapter_id and the UI uses as voice-override matrix columns; the
// conformance test fails when the two drift.
const (
	AdapterLocalKokoro       = "local:kokoro"
	AdapterOpenAITTS         = "byok:openai-tts"
	AdapterElevenLabs        = "byok:elevenlabs"
	LegacyAdapterLocalKokoro = "local:kokoro-local"
)

// AdapterVoiceMap is the type each registered TTS adapter exposes for
// startup verification. Keys are canonical voice IDs; values are the
// adapter's backend voice identifier.
type AdapterVoiceMap map[string]string

// Resolve returns the override for this adapter when present, otherwise the
// canonical mapping. ok is false when neither exists, letting each adapter
// apply its own fallback (pass-through for Kokoro, a provider default for the
// hosted adapters).
func (m AdapterVoiceMap) Resolve(canonical string, overrides map[string]string, adapterID string) (string, bool) {
	if v, ok := overrides[adapterID]; ok && v != "" {
		return v, true
	}
	v, ok := m[canonical]
	return v, ok && v != ""
}

// AdapterCatalogEntry is one registered adapter with its declared mapping.
type AdapterCatalogEntry struct {
	TierProvider string // "local:kokoro", "byok:openai-tts", "byok:elevenlabs"
	Mapping      AdapterVoiceMap
}

// CanonicalAdapterMappings is the production adapter inventory checked at
// startup. Adding a TTS adapter means adding its mapping here (or wiring a
// catalog-derived entry) so VerifyCatalog cannot pass by omission.
var CanonicalAdapterMappings = []AdapterCatalogEntry{
	{TierProvider: AdapterLocalKokoro, Mapping: LocalKokoroMapping},
	{TierProvider: AdapterOpenAITTS, Mapping: OpenAIMapping},
	{TierProvider: AdapterElevenLabs, Mapping: ElevenLabsMapping},
}

// VerifyCatalog asserts every registered adapter declares a mapping for
// every canonical voice. Returns a structured error naming the first
// adapter/voice gap; bootstrap fails fast on a non-nil error rather than
// silently launching with incomplete coverage.
func VerifyCatalog(entries []AdapterCatalogEntry) error {
	for _, e := range entries {
		if e.TierProvider == "" {
			return &MissingCanonicalVoiceError{Adapter: "(unnamed)", CanonicalVoice: "(adapter has no id)"}
		}
		for _, canonical := range CanonicalVoiceIDs {
			if value, ok := e.Mapping[canonical]; !ok || value == "" {
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
// adapter.
var LocalKokoroMapping = AdapterVoiceMap{
	"voice.feminine.warm":     "af_bella",
	"voice.feminine.neutral":  "af_sarah",
	"voice.masculine.warm":    "am_adam",
	"voice.masculine.neutral": "am_michael",
	"voice.neutral.default":   "af_nicole",
}

// OpenAIMapping is the canonical->backend mapping for OpenAI TTS.
var OpenAIMapping = AdapterVoiceMap{
	"voice.feminine.warm":     "shimmer",
	"voice.feminine.neutral":  "nova",
	"voice.masculine.warm":    "onyx",
	"voice.masculine.neutral": "echo",
	"voice.neutral.default":   "alloy",
}

// ElevenLabsMapping is the canonical->backend mapping for ElevenLabs
// multilingual v2. The values are public voice ids.
var ElevenLabsMapping = AdapterVoiceMap{
	"voice.feminine.warm":     "EXAVITQu4vr4xnSDxMaL", // Bella
	"voice.feminine.neutral":  "21m00Tcm4TlvDq8ikWAM", // Rachel
	"voice.masculine.warm":    "ErXwobaYiN019PkySvjV", // Antoni
	"voice.masculine.neutral": "29vD33N1CtxCmqQRPOHJ", // Drew
	"voice.neutral.default":   "21m00Tcm4TlvDq8ikWAM", // Rachel
}
