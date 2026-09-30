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
		Id:             r.ID,
		Subdomain:      r.Subdomain,
		Scenario:       r.Scenario,
		Domain:         r.Domain,
		LocalPort:      int32(r.LocalPort),
		Tier:           tierToProto(r.Tier),
		LeaseId:        r.LeaseID,
		Enabled:        r.Enabled,
		HealthPath:     r.HealthPath,
		PublicUrl:      r.PublicURL(),
		Source:         sourceToProto(r.Source),
		ServiceTarget:  r.ServiceTarget,
		PublicExposure: publicExposureToProto(r.PublicExposure),
		CreatedAt:      timestamppb.New(r.CreatedAt.UTC()),
		UpdatedAt:      timestamppb.New(r.UpdatedAt.UTC()),
	}
}

func publicExposureToProto(p routes.PublicExposure) routesv1.PublicExposure {
	switch routes.NormalizePublicExposure(p) {
	case routes.PublicExposureEnabled:
		return routesv1.PublicExposure_PUBLIC_EXPOSURE_ENABLED
	case routes.PublicExposureDisabled:
		return routesv1.PublicExposure_PUBLIC_EXPOSURE_DISABLED
	default:
		return routesv1.PublicExposure_PUBLIC_EXPOSURE_INHERIT
	}
}

// publicExposureFromProto maps a wire override to the domain value. UNSPECIFIED
// returns the empty PublicExposure so a partial update leaves it unchanged
// (Create normalizes empty to inherit).
func publicExposureFromProto(p routesv1.PublicExposure) routes.PublicExposure {
	switch p {
	case routesv1.PublicExposure_PUBLIC_EXPOSURE_INHERIT:
		return routes.PublicExposureInherit
	case routesv1.PublicExposure_PUBLIC_EXPOSURE_ENABLED:
		return routes.PublicExposureEnabled
	case routesv1.PublicExposure_PUBLIC_EXPOSURE_DISABLED:
		return routes.PublicExposureDisabled
	default:
		return ""
	}
}

func sourceToProto(s routes.RouteSource) routesv1.RouteSource {
	switch s {
	case routes.SourceScenario:
		return routesv1.RouteSource_ROUTE_SOURCE_SCENARIO
	case routes.SourceExternal:
		return routesv1.RouteSource_ROUTE_SOURCE_EXTERNAL
	default:
		return routesv1.RouteSource_ROUTE_SOURCE_UNSPECIFIED
	}
}

// sourceFromProto maps a wire source to the domain source. UNSPECIFIED returns
// the empty RouteSource so the service applies its default (scenario).
func sourceFromProto(s routesv1.RouteSource) routes.RouteSource {
	switch s {
	case routesv1.RouteSource_ROUTE_SOURCE_SCENARIO:
		return routes.SourceScenario
	case routesv1.RouteSource_ROUTE_SOURCE_EXTERNAL:
		return routes.SourceExternal
	default:
		return ""
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
