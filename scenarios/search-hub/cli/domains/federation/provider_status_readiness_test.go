package federation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

func TestSelectedProviderReadinessHintsKeepSignalsAndFallbackDistinct(t *testing.T) {
	tests := []struct {
		name   string
		health *routingv1.ProviderHealth
		want   string
		forbid string
	}{
		{
			name:   "route down",
			health: &routingv1.ProviderHealth{ProviderId: "owner.alpha", Reachable: false, IndexAge: "unreported: provider unreachable"},
			want:   "route is unreachable",
			forbid: "route resolved",
		},
		{
			name:   "route up status source unavailable",
			health: &routingv1.ProviderHealth{ProviderId: "owner.alpha", Reachable: true, Degraded: true, IndexAge: "unreported: status probe failed"},
			want:   "freshness source is unreported",
			forbid: "route is unreachable",
		},
		{
			name:   "no freshness source",
			health: &routingv1.ProviderHealth{ProviderId: "owner.alpha", Reachable: true, IndexAge: "not_applicable: provider has no status_endpoint"},
			want:   "no freshness source is declared",
			forbid: "freshness source is unreported",
		},
		{
			name:   "reported source freshness",
			health: &routingv1.ProviderHealth{ProviderId: "owner.alpha", Reachable: true, IndexAge: "48h", FreshnessBudget: "24h", AutomaticEligible: false, AutomaticExclusionReason: "stale index"},
			want:   "health signals only",
			forbid: "route is unreachable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hints := strings.Join(selectedProviderReadinessHints(&routingv1.ProviderStatusResponse{
				ProviderId: "owner.alpha",
				Health:     tt.health,
			}), "\n")
			require.Contains(t, hints, tt.want)
			require.NotContains(t, hints, tt.forbid)
			require.Contains(t, hints, "Supported fallback: not established")
			require.NotContains(t, hints, "--type")
			require.NotContains(t, hints, "another owner succeeded")
		})
	}
}
