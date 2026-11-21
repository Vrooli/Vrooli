package capabilities

import (
	"audio-tools/internal/providercatalog"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	diagv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics"
)

// CapabilityForFeature maps a Def.Features[] entry to its proto Capability. The
// feature vocabulary and its capability mapping live in providercatalog; this
// function only translates the catalogue vocabulary to the diagnostics proto.
//
// The second return is false for feature strings outside the declared
// vocabulary. The diagnostics proto has no speaker capability, so speaker
// evidence is folded into CAPABILITY_STT to preserve the existing health rollup.
func CapabilityForFeature(feature string) (diagv1.Capability, bool) {
	capability, ok := providercatalog.CapabilityForFeature(feature)
	if !ok {
		return diagv1.Capability_CAPABILITY_UNSPECIFIED, false
	}
	return protoCapability(capability), true
}

func protoCapability(capability providercatalog.Capability) diagv1.Capability {
	switch capability {
	case providercatalog.CapabilityTTS:
		return diagv1.Capability_CAPABILITY_TTS
	case providercatalog.CapabilitySummarize:
		return diagv1.Capability_CAPABILITY_SUMMARIZE
	case providercatalog.CapabilityTranscode:
		return diagv1.Capability_CAPABILITY_TRANSCODE
	case providercatalog.CapabilitySTT, providercatalog.CapabilitySpeaker:
		return diagv1.Capability_CAPABILITY_STT
	default:
		return diagv1.Capability_CAPABILITY_UNSPECIFIED
	}
}

// TierForProviderID maps a stable provider ID to its ProviderTier. Provider
// tiers are declared in providercatalog; this function translates them to the
// shared proto vocabulary.
func TierForProviderID(id string) commonv1.ProviderTier {
	return protoTier(providercatalog.TierForProviderID(id))
}

func protoTier(tier providercatalog.Tier) commonv1.ProviderTier {
	switch tier {
	case providercatalog.TierLocal:
		return commonv1.ProviderTier_PROVIDER_TIER_LOCAL
	case providercatalog.TierBYOK:
		return commonv1.ProviderTier_PROVIDER_TIER_BYOK
	case providercatalog.TierVrooli:
		return commonv1.ProviderTier_PROVIDER_TIER_VROOLI
	case providercatalog.TierBrowser:
		return commonv1.ProviderTier_PROVIDER_TIER_BROWSER
	default:
		return commonv1.ProviderTier_PROVIDER_TIER_UNSPECIFIED
	}
}
