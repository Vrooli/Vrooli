package routing

import (
	"testing"
	"time"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestProjectSelectedOwnerStatusKeepsFactsSeparate(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	status := func(responseID, healthID, indexAge string, indexedAt *timestamppb.Timestamp, reachable bool) *routingv1.ProviderStatusResponse {
		return &routingv1.ProviderStatusResponse{
			ProviderId: responseID,
			ObservedAt: timestamppb.New(now),
			Health: &routingv1.ProviderHealth{
				ProviderId: healthID, Reachable: reachable, Reachability: "selected endpoint probe completed",
				IndexAge: indexAge, LastIndexedAt: indexedAt, FreshnessBudget: "1h",
			},
		}
	}

	tests := []struct {
		name      string
		response  *routingv1.ProviderStatusResponse
		identity  string
		rpc       string
		endpoint  string
		freshness string
	}{
		{
			name:     "current endpoint evidence",
			response: status("selected.synthetic", "selected.synthetic", "5m", timestamppb.New(now.Add(-5*time.Minute)), true),
			identity: selectedStatusIdentityMatched, rpc: selectedStatusRPCReachable,
			endpoint: selectedStatusEndpointReachable, freshness: selectedStatusFreshnessCurrent,
		},
		{
			name:     "stale index remains distinct from reachable endpoint",
			response: status("selected.synthetic", "selected.synthetic", "2h", timestamppb.New(now.Add(-2*time.Hour)), true),
			identity: selectedStatusIdentityMatched, rpc: selectedStatusRPCReachable,
			endpoint: selectedStatusEndpointReachable, freshness: selectedStatusFreshnessStale,
		},
		{
			name:     "unreported freshness remains unknown",
			response: status("selected.synthetic", "selected.synthetic", "unreported: timeout", nil, true),
			identity: selectedStatusIdentityMatched, rpc: selectedStatusRPCReachable,
			endpoint: selectedStatusEndpointReachable, freshness: selectedStatusUnknown,
		},
		{
			name:     "missing observation time keeps freshness unknown",
			response: &routingv1.ProviderStatusResponse{ProviderId: "selected.synthetic", Health: &routingv1.ProviderHealth{ProviderId: "selected.synthetic", Reachable: true, Reachability: "endpoint resolved", IndexAge: "5m", LastIndexedAt: timestamppb.New(now.Add(-5 * time.Minute)), FreshnessBudget: "1h"}},
			identity: selectedStatusIdentityMatched, rpc: selectedStatusRPCReachable,
			endpoint: selectedStatusEndpointReachable, freshness: selectedStatusUnknown,
		},
		{
			name:     "invalid freshness budget keeps freshness unknown",
			response: &routingv1.ProviderStatusResponse{ProviderId: "selected.synthetic", ObservedAt: timestamppb.New(now), Health: &routingv1.ProviderHealth{ProviderId: "selected.synthetic", Reachable: true, Reachability: "endpoint resolved", IndexAge: "5m", LastIndexedAt: timestamppb.New(now.Add(-5 * time.Minute)), FreshnessBudget: "unknown"}},
			identity: selectedStatusIdentityMatched, rpc: selectedStatusRPCReachable,
			endpoint: selectedStatusEndpointReachable, freshness: selectedStatusUnknown,
		},
		{
			name:     "future index timestamp keeps freshness unknown",
			response: &routingv1.ProviderStatusResponse{ProviderId: "selected.synthetic", ObservedAt: timestamppb.New(now), Health: &routingv1.ProviderHealth{ProviderId: "selected.synthetic", Reachable: true, Reachability: "endpoint resolved", IndexAge: "-5m", LastIndexedAt: timestamppb.New(now.Add(5 * time.Minute)), FreshnessBudget: "1h"}},
			identity: selectedStatusIdentityMatched, rpc: selectedStatusRPCReachable,
			endpoint: selectedStatusEndpointReachable, freshness: selectedStatusUnknown,
		},
		{
			name:     "identity mismatch does not borrow owner facts",
			response: status("other.synthetic", "selected.synthetic", "5m", timestamppb.New(now.Add(-5*time.Minute)), true),
			identity: selectedStatusIdentityMismatch, rpc: selectedStatusRPCReachable,
			endpoint: selectedStatusUnknown, freshness: selectedStatusUnknown,
		},
		{
			name:     "unreachable endpoint remains distinct from status RPC",
			response: status("selected.synthetic", "selected.synthetic", "unreported: endpoint unavailable", nil, false),
			identity: selectedStatusIdentityMatched, rpc: selectedStatusRPCReachable,
			endpoint: selectedStatusEndpointUnreachable, freshness: selectedStatusUnknown,
		},
		{
			name:     "endpoint route without reachability result remains unknown",
			response: &routingv1.ProviderStatusResponse{ProviderId: "selected.synthetic", ObservedAt: timestamppb.New(now), Health: &routingv1.ProviderHealth{ProviderId: "selected.synthetic", Reachable: true}},
			identity: selectedStatusIdentityMatched, rpc: selectedStatusRPCReachable,
			endpoint: selectedStatusUnknown, freshness: selectedStatusUnknown,
		},
		{
			name:     "nil response means status RPC delivery unknown",
			identity: selectedStatusUnknown, rpc: selectedStatusUnknown,
			endpoint: selectedStatusUnknown, freshness: selectedStatusUnknown,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := projectSelectedOwnerStatus("selected.synthetic", test.response)
			if got.identityMatch != test.identity || got.statusRPCReachability != test.rpc || got.ownerEndpointReachable != test.endpoint || got.freshness != test.freshness {
				t.Fatalf("projection = %+v; want identity=%s rpc=%s endpoint=%s freshness=%s", got, test.identity, test.rpc, test.endpoint, test.freshness)
			}
			if got.supportedFallback != selectedStatusUnknown {
				t.Fatalf("supported fallback = %q, want UNKNOWN without approved owner/task evidence", got.supportedFallback)
			}
		})
	}
}
