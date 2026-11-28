package capabilities

import "audio-tools/internal/providercatalog"

// CanonicalProviderID maps the runtime identity a chain reports to the stable
// provider id used by health, lifecycle, settings, and usage.
//
// The chain result carries the engine/provider identity that actually executed
// (whisper-local, kyutai, sherpa-streaming, kokoro-local, ollama-local). Those
// values are useful evidence and the soak driver still compares them, but a
// public API surface should name the catalogue provider so a client can join a
// result to /status and /providers. Both identity vocabularies live in the
// provider catalogue: engine ids first, then runtime aliases. Unknown values
// pass through unchanged: a new runtime is visible before its catalogue row
// lands, rather than being silently relabelled.
func CanonicalProviderID(raw string) string {
	if raw == "" {
		return ""
	}
	if id, ok := providercatalog.ProviderIDForRuntimeIdentity(raw); ok {
		return id
	}
	return raw
}
