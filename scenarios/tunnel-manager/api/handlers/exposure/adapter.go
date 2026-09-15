package exposure

import (
	"time"

	"tunnel-manager/internal/exposure"

	exposurev1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/exposure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func leaseToProto(l exposure.Lease) *exposurev1.Lease {
	return &exposurev1.Lease{
		Id:            l.ID,
		Scenario:      l.Scenario,
		RequestedBy:   l.RequestedBy,
		CreatedAt:     optionalTS(l.CreatedAt),
		ExpiresAt:     optionalTS(l.ExpiresAt),
		ExtendedCount: int32(l.ExtendedCount),
		Status:        leaseStatusToProto(l.Status),
	}
}

func exposureToProto(e exposure.Exposure) *exposurev1.Exposure {
	out := &exposurev1.Exposure{
		Scenario:  e.Scenario,
		Subdomain: e.Subdomain,
		PublicUrl: e.PublicURL,
		LocalPort: int32(e.LocalPort),
		Tier:      e.Tier,
		Enabled:   e.Enabled,
	}
	if e.Lease != nil {
		out.Lease = leaseToProto(*e.Lease)
	}
	return out
}

func leaseStatusToProto(s exposure.LeaseStatus) exposurev1.LeaseStatus {
	switch s {
	case exposure.LeaseActive:
		return exposurev1.LeaseStatus_LEASE_STATUS_ACTIVE
	case exposure.LeaseExpired:
		return exposurev1.LeaseStatus_LEASE_STATUS_EXPIRED
	case exposure.LeaseRevoked:
		return exposurev1.LeaseStatus_LEASE_STATUS_REVOKED
	default:
		return exposurev1.LeaseStatus_LEASE_STATUS_UNSPECIFIED
	}
}

func leaseStatusFromProto(s exposurev1.LeaseStatus) exposure.LeaseStatus {
	switch s {
	case exposurev1.LeaseStatus_LEASE_STATUS_ACTIVE:
		return exposure.LeaseActive
	case exposurev1.LeaseStatus_LEASE_STATUS_EXPIRED:
		return exposure.LeaseExpired
	case exposurev1.LeaseStatus_LEASE_STATUS_REVOKED:
		return exposure.LeaseRevoked
	default:
		return ""
	}
}

func optionalTS(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t.UTC())
}
