package convert

import (
	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	engine "github.com/vrooli/vrooli/internal/capacity"
	capacitypb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/capacity"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services"
)

// optionalInt32 maps an optional Go int pointer to a proto optional int32.
func optionalInt32(v *int) *int32 {
	if v == nil {
		return nil
	}
	n := int32(*v)
	return &n
}

// CapacityClaimToProto converts an app claim view to the API message.
func CapacityClaimToProto(c capacityapp.ClaimView) *capacitypb.CapacityClaim {
	out := &capacitypb.CapacityClaim{
		ClaimId:        c.ClaimID,
		OwnerKind:      c.OwnerKind,
		OwnerId:        c.OwnerID,
		InstanceId:     c.InstanceID,
		ResourceKind:   c.ResourceKind,
		GpuIndex:       optionalInt32(c.GPUIndex),
		AmountBytes:    c.AmountBytes,
		PreferredBytes: c.PreferredBytes,
		FloorBytes:     c.FloorBytes,
		Priority:       int32(c.Priority),
		PriorityTier:   c.PriorityTier,
		Protected:      c.Protected,
		Status:         c.Status,
		ActivityState:  c.ActivityState,
		Generation:     c.Generation,
	}
	if c.LastActiveAt != nil {
		out.LastActiveAt = *c.LastActiveAt
	}
	return out
}

// CapacityClaimsToProto converts a claim slice to API messages.
func CapacityClaimsToProto(claims []capacityapp.ClaimView) []*capacitypb.CapacityClaim {
	out := make([]*capacitypb.CapacityClaim, 0, len(claims))
	for _, c := range claims {
		out = append(out, CapacityClaimToProto(c))
	}
	return out
}

// CapacityFindingToProto converts an engine reconciliation finding to the API message.
func CapacityFindingToProto(f engine.Finding) *capacitypb.CapacityFinding {
	return &capacitypb.CapacityFinding{
		Class:         f.Class,
		OwnerId:       f.OwnerID,
		OwnerKind:     f.OwnerKind,
		ResourceKind:  f.ResourceKind,
		GpuIndex:      optionalInt32(f.GPUIndex),
		Pid:           int32(f.PID),
		ProcessName:   f.ProcessName,
		ObservedBytes: f.ObservedBytes,
		ClaimedBytes:  f.ClaimedBytes,
		ClaimId:       f.ClaimID,
		Severity:      f.Severity,
		Message:       f.Message,
	}
}

// CapacityFindingsToProto converts a finding slice to API messages.
func CapacityFindingsToProto(findings []engine.Finding) []*capacitypb.CapacityFinding {
	out := make([]*capacitypb.CapacityFinding, 0, len(findings))
	for _, f := range findings {
		out = append(out, CapacityFindingToProto(f))
	}
	return out
}

// GpuContentionToProto converts a per-GPU contention row to the API message.
func GpuContentionToProto(g services.GpuContention) *capacitypb.GpuCapacity {
	return &capacitypb.GpuCapacity{
		Index:                    int32(g.Index),
		Name:                     g.Name,
		TotalBytes:               g.TotalBytes,
		UsedBytes:                g.UsedBytes,
		FreeBytes:                g.FreeBytes,
		ClaimedBytes:             g.ClaimedBytes,
		MemoryUtilizationPercent: g.MemoryUtilizationPercent,
	}
}

// GpuContentionsToProto converts a contention slice to API messages.
func GpuContentionsToProto(gpus []services.GpuContention) []*capacitypb.GpuCapacity {
	out := make([]*capacitypb.GpuCapacity, 0, len(gpus))
	for _, g := range gpus {
		out = append(out, GpuContentionToProto(g))
	}
	return out
}

// PolicyLeversToProto converts policy entries to API messages.
func PolicyLeversToProto(entries []capacityapp.PolicyEntry) []*capacitypb.PolicyLever {
	out := make([]*capacitypb.PolicyLever, 0, len(entries))
	for _, e := range entries {
		out = append(out, &capacitypb.PolicyLever{Key: e.Key, Value: e.Value})
	}
	return out
}
