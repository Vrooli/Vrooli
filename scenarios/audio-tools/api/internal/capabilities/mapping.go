package capabilities

import (
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	diagv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics"
)

// CapabilityForFeature maps a Def.Features[] entry (the strings declared
// on entries in the Known catalog) to its proto Capability. The second
// return is false for feature strings that are not capability-keyed
// (e.g. "tts-cache", "audio-provider-routing") — callers should skip
// them rather than emit a CAPABILITY_UNSPECIFIED row.
//
// Mapping rules (Phase 1 of the health-visibility plan):
//
//	voice-input / voice-streaming  → STT
//	voice-output                   → TTS
//	ai-command-generation          → SUMMARIZE
//	(transcode is in-process, not a capability provider)
func CapabilityForFeature(feature string) (diagv1.Capability, bool) {
	switch feature {
	case "voice-input", "voice-streaming":
		return diagv1.Capability_CAPABILITY_STT, true
	case "voice-output":
		return diagv1.Capability_CAPABILITY_TTS, true
	case "ai-command-generation":
		return diagv1.Capability_CAPABILITY_SUMMARIZE, true
	default:
		return diagv1.Capability_CAPABILITY_UNSPECIFIED, false
	}
}

// TierForProviderID maps a Known[].ID to its ProviderTier.
//
//	openrouter  → BYOK (cloud LLM)
//	whisper-stt / kokoro-tts / ollama / speaker-verification → LOCAL
//	audio-tools (scenario rollup) → UNSPECIFIED
//
// Anything else falls through to UNSPECIFIED.
func TierForProviderID(id string) commonv1.ProviderTier {
	switch id {
	case "openrouter":
		return commonv1.ProviderTier_PROVIDER_TIER_BYOK
	case "whisper-stt", "kokoro-tts", "ollama", "speaker-verification":
		return commonv1.ProviderTier_PROVIDER_TIER_LOCAL
	default:
		return commonv1.ProviderTier_PROVIDER_TIER_UNSPECIFIED
	}
}
