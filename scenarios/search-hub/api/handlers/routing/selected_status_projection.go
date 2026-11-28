package routing

import (
	"net/http"
	"strings"
	"time"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	selectedStatusIdentityHeader      = "X-Search-Hub-Selected-Owner-Identity-Match"
	selectedStatusRPCHeader           = "X-Search-Hub-Selected-Owner-Status-RPC-Reachability"
	selectedStatusEndpointHeader      = "X-Search-Hub-Selected-Owner-Endpoint-Reachability"
	selectedStatusFreshnessHeader     = "X-Search-Hub-Selected-Owner-Freshness"
	selectedStatusFallbackHeader      = "X-Search-Hub-Selected-Owner-Supported-Fallback"
	selectedStatusUnknown             = "UNKNOWN"
	selectedStatusRPCReachable        = "REACHABLE"
	selectedStatusEndpointReachable   = "REACHABLE"
	selectedStatusEndpointUnreachable = "UNREACHABLE"
	selectedStatusIdentityMatched     = "MATCHED"
	selectedStatusIdentityMismatch    = "MISMATCH"
	selectedStatusFreshnessCurrent    = "CURRENT"
	selectedStatusFreshnessStale      = "STALE"
)

// selectedOwnerStatusProjection keeps response identity, successful status
// RPC delivery, owner endpoint reachability, freshness, and fallback evidence
// as separate facts. Fallback remains unknown unless an approved owner/task
// contract is supplied to a future projection revision.
type selectedOwnerStatusProjection struct {
	identityMatch          string
	statusRPCReachability  string
	ownerEndpointReachable string
	freshness              string
	supportedFallback      string
}

func projectSelectedOwnerStatus(providerID string, response *routingv1.ProviderStatusResponse) selectedOwnerStatusProjection {
	projection := selectedOwnerStatusProjection{
		identityMatch:          selectedStatusUnknown,
		statusRPCReachability:  selectedStatusUnknown,
		ownerEndpointReachable: selectedStatusUnknown,
		freshness:              selectedStatusUnknown,
		supportedFallback:      selectedStatusUnknown,
	}
	if response == nil {
		return projection
	}
	// This projection is attached to a successful status response. It reports
	// transport completion separately from whether the nested identities agree.
	projection.statusRPCReachability = selectedStatusRPCReachable

	expectedID := strings.TrimSpace(providerID)
	responseID := strings.TrimSpace(response.GetProviderId())
	health := response.GetHealth()
	if expectedID != "" && responseID != "" && health != nil && strings.TrimSpace(health.GetProviderId()) != "" {
		if responseID == expectedID && strings.TrimSpace(health.GetProviderId()) == expectedID {
			projection.identityMatch = selectedStatusIdentityMatched
		} else {
			projection.identityMatch = selectedStatusIdentityMismatch
		}
	}
	if health == nil || projection.identityMatch != selectedStatusIdentityMatched {
		return projection
	}
	if strings.TrimSpace(health.GetReachability()) != "" {
		if health.GetReachable() {
			projection.ownerEndpointReachable = selectedStatusEndpointReachable
		} else {
			projection.ownerEndpointReachable = selectedStatusEndpointUnreachable
		}
	}
	projection.freshness = selectedOwnerFreshness(response.GetObservedAt(), health)
	return projection
}

func selectedOwnerFreshness(observedAt *timestamppb.Timestamp, health *routingv1.ProviderHealth) string {
	if observedAt == nil || health == nil || health.GetLastIndexedAt() == nil {
		return selectedStatusUnknown
	}
	if observedAt.CheckValid() != nil || health.GetLastIndexedAt().CheckValid() != nil {
		return selectedStatusUnknown
	}
	indexAge := strings.ToLower(strings.TrimSpace(health.GetIndexAge()))
	if strings.HasPrefix(indexAge, "unreported:") || strings.HasPrefix(indexAge, "not_applicable:") {
		return selectedStatusUnknown
	}
	if indexAge == "" {
		return selectedStatusUnknown
	}
	if _, err := time.ParseDuration(indexAge); err != nil {
		return selectedStatusUnknown
	}
	budget, err := time.ParseDuration(strings.TrimSpace(health.GetFreshnessBudget()))
	if err != nil || budget <= 0 {
		return selectedStatusUnknown
	}
	age := observedAt.AsTime().Sub(health.GetLastIndexedAt().AsTime())
	if age < 0 {
		return selectedStatusUnknown
	}
	if age > budget {
		return selectedStatusFreshnessStale
	}
	return selectedStatusFreshnessCurrent
}

func (projection selectedOwnerStatusProjection) writeHeaders(header http.Header) {
	if header == nil {
		return
	}
	header.Set(selectedStatusIdentityHeader, projection.identityMatch)
	header.Set(selectedStatusRPCHeader, projection.statusRPCReachability)
	header.Set(selectedStatusEndpointHeader, projection.ownerEndpointReachable)
	header.Set(selectedStatusFreshnessHeader, projection.freshness)
	header.Set(selectedStatusFallbackHeader, projection.supportedFallback)
}
