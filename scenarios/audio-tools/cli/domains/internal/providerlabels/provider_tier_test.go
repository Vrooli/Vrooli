package providerlabels

import (
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
)

func TestTierLabel(t *testing.T) {
	cases := []struct {
		name string
		tier commonv1.ProviderTier
		want string
	}{
		{name: "local", tier: commonv1.ProviderTier_PROVIDER_TIER_LOCAL, want: "local"},
		{name: "byok", tier: commonv1.ProviderTier_PROVIDER_TIER_BYOK, want: "byok"},
		{name: "vrooli", tier: commonv1.ProviderTier_PROVIDER_TIER_VROOLI, want: "vrooli"},
		{name: "unspecified", tier: commonv1.ProviderTier_PROVIDER_TIER_UNSPECIFIED, want: "unknown"},
		{name: "unknown", tier: commonv1.ProviderTier(999), want: "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := TierLabel(tc.tier); got != tc.want {
				t.Fatalf("TierLabel(%s) = %q, want %q", tc.tier, got, tc.want)
			}
		})
	}
}
