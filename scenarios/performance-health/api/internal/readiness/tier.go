package readiness

import readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/readiness"

// TierToProto maps the internal tier to the wire enum.
func TierToProto(t Tier) readinessv1.CaptureTier {
	switch t {
	case TierNone:
		return readinessv1.CaptureTier_CAPTURE_TIER_NONE
	case Tier0:
		return readinessv1.CaptureTier_CAPTURE_TIER_0
	case Tier1:
		return readinessv1.CaptureTier_CAPTURE_TIER_1
	default:
		return readinessv1.CaptureTier_CAPTURE_TIER_UNSPECIFIED
	}
}

// String renders the tier as the canonical short label used in fleet rollups.
func (t Tier) String() string {
	switch t {
	case TierNone:
		return "none"
	case Tier0:
		return "0"
	case Tier1:
		return "1"
	default:
		return "unspecified"
	}
}
