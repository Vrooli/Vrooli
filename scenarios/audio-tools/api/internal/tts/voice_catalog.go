package tts

import (
	"strings"

	"audio-tools/internal/providercatalog"
)

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
	AdapterLocalKale         = "local:kale-tts"
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

// Provider declarations own the canonical-to-runtime voice aliases. These
// exported values are compatibility views used by existing adapters/tests.
func declaredMapping(adapterID string) AdapterVoiceMap {
	for _, p := range providercatalog.All() {
		if p.AdapterID == adapterID {
			m := make(AdapterVoiceMap, len(p.VoiceAliases))
			for k, v := range p.VoiceAliases {
				m[k] = v
			}
			return m
		}
	}
	return nil
}

var (
	LocalKaleMapping   = declaredMapping(AdapterLocalKale)
	LocalKokoroMapping = declaredMapping(AdapterLocalKokoro)
	OpenAIMapping      = declaredMapping(AdapterOpenAITTS)
	ElevenLabsMapping  = declaredMapping(AdapterElevenLabs)
)

// AdapterCatalogEntry is one registered adapter with its declared mapping.
type AdapterCatalogEntry struct {
	TierProvider string // "local:kokoro", "byok:openai-tts", "byok:elevenlabs"
	Mapping      AdapterVoiceMap
}

// CanonicalAdapterMappings is the production adapter inventory checked at
// startup. Adding a TTS adapter means adding its mapping here (or wiring a
// catalog-derived entry) so VerifyCatalog cannot pass by omission.
var CanonicalAdapterMappings = func() []AdapterCatalogEntry {
	var entries []AdapterCatalogEntry
	for _, p := range providercatalog.All() {
		if p.AdapterID != "" {
			entries = append(entries, AdapterCatalogEntry{TierProvider: p.AdapterID, Mapping: declaredMapping(p.AdapterID)})
		}
	}
	return entries
}()

// MappingForAdapter returns the implementation-owned voice mapping for a
// catalog-declared adapter id. The provider catalog owns the id binding; this
// package owns the adapter-specific voice names.
func MappingForAdapter(adapterID string) (AdapterVoiceMap, bool) {
	for _, entry := range CanonicalAdapterMappings {
		if entry.TierProvider == adapterID {
			return entry.Mapping, true
		}
	}
	return nil, false
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
		known := make(map[string]bool, len(CanonicalVoiceIDs))
		for _, id := range CanonicalVoiceIDs {
			known[id] = true
		}
		for canonical, backend := range e.Mapping {
			if !known[canonical] || strings.TrimSpace(backend) == "" {
				return &InvalidVoiceMappingError{Adapter: e.TierProvider, CanonicalVoice: canonical}
			}
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

// InvalidVoiceMappingError identifies a declaration that names an unknown
// canonical voice or an empty backend voice identifier.
type InvalidVoiceMappingError struct{ Adapter, CanonicalVoice string }

func (e *InvalidVoiceMappingError) Error() string {
	return "audio-tools/tts: adapter " + e.Adapter + " has invalid mapping for canonical voice " + e.CanonicalVoice
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
