package routes

import (
	"tunnel-manager/internal/routes"

	routesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/routes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// domainToProto converts an internal routes.Route into the wire shape the
// routes proto declares. Lives in the handler package by intent — the
// conversion is mechanical and only used at the transport edge.
func domainToProto(r routes.Route) *routesv1.Route {
	return &routesv1.Route{
		Id:         r.ID,
		Subdomain:  r.Subdomain,
		Scenario:   r.Scenario,
		Domain:     r.Domain,
		LocalPort:  int32(r.LocalPort),
		Tier:       tierToProto(r.Tier),
		LeaseId:    r.LeaseID,
		Enabled:    r.Enabled,
		HealthPath: r.HealthPath,
		PublicUrl:  r.PublicURL(),
		CreatedAt:  timestamppb.New(r.CreatedAt.UTC()),
		UpdatedAt:  timestamppb.New(r.UpdatedAt.UTC()),
	}
}

func tierToProto(t routes.Tier) routesv1.Tier {
	switch t {
	case routes.TierCore:
		return routesv1.Tier_TIER_CORE
	case routes.TierLeased:
		return routesv1.Tier_TIER_LEASED
	default:
		return routesv1.Tier_TIER_UNSPECIFIED
	}
}

// tierFromProto maps a wire tier to the domain tier. TIER_UNSPECIFIED
// returns the empty Tier so the service applies its default.
func tierFromProto(t routesv1.Tier) routes.Tier {
	switch t {
	case routesv1.Tier_TIER_CORE:
		return routes.TierCore
	case routesv1.Tier_TIER_LEASED:
		return routes.TierLeased
	default:
		return ""
	}
}
