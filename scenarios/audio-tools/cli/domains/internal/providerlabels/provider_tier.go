package providerlabels

import commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"

// TierLabel renders the provider routing tier for CLI output.
func TierLabel(t commonv1.ProviderTier) string {
	switch t {
	case commonv1.ProviderTier_PROVIDER_TIER_LOCAL:
		return "local"
	case commonv1.ProviderTier_PROVIDER_TIER_BYOK:
		return "byok"
	case commonv1.ProviderTier_PROVIDER_TIER_VROOLI:
		return "vrooli"
	default:
		return "unknown"
	}
}
