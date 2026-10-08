package services

import (
	"context"

	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	engine "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/hostinventory"
)

// capacityApp is the slice of the platform capacity application service this
// scenario depends on. It is an interface so tests inject a fake and the
// handler never opens the real ledger or shells out to nvidia-smi.
type capacityApp interface {
	List(ctx context.Context, req capacityapp.ListRequest) (capacityapp.ListOutput, error)
	Reconcile(ctx context.Context) (capacityapp.ReconcileOutput, error)
	PolicyGet(ctx context.Context, key string) (capacityapp.PolicyOutput, error)
	PolicySet(ctx context.Context, key, value string) (capacityapp.PolicyOutput, error)
}

// CapacityService is system-monitor's read/governance adapter over the platform
// `internal/capacity` ledger. It reads the ledger (active claims, reconciliation
// findings, policy) and the live host inventory (per-GPU contention) and mutates
// only policy. Dependency direction is strictly system-monitor -> capacity.
type CapacityService struct {
	app     capacityApp
	collect func(ctx context.Context) (hostinventory.Snapshot, error)
}

// NewCapacityService constructs the production capacity service backed by the
// real ledger (default SQLite store) and live host inventory.
func NewCapacityService() *CapacityService {
	return &CapacityService{
		app:     capacityapp.Service{},
		collect: hostinventory.Collect,
	}
}

// NewCapacityServiceWith constructs a capacity service with injected seams (tests).
func NewCapacityServiceWith(app capacityApp, collect func(ctx context.Context) (hostinventory.Snapshot, error)) *CapacityService {
	return &CapacityService{app: app, collect: collect}
}

// GpuContention is the per-GPU contention picture: live total/used capacity plus
// the sum of active claims attributed to that GPU.
type GpuContention struct {
	Index                    int
	Name                     string
	TotalBytes               int64
	UsedBytes                int64
	FreeBytes                int64
	ClaimedBytes             int64
	MemoryUtilizationPercent float64
}

// CapacityOverview is the dashboard payload: per-GPU contention, the active
// claim table, and any sensing warnings (never silently dropped).
type CapacityOverview struct {
	GPUs             []GpuContention
	Claims           []capacityapp.ClaimView
	SensingAvailable bool
	Warnings         []string
}

// Overview combines a live host-inventory snapshot with the active claim ledger
// into the per-GPU contention picture plus the claim table.
func (s *CapacityService) Overview(ctx context.Context) (CapacityOverview, error) {
	listed, err := s.app.List(ctx, capacityapp.ListRequest{ActiveOnly: true})
	if err != nil {
		return CapacityOverview{}, err
	}

	out := CapacityOverview{Claims: listed.Claims}

	snapshot, snapErr := s.collect(ctx)
	if snapErr != nil {
		// Sensing failure is not fatal — the claim table still renders. Surface
		// the failure honestly as a warning rather than dropping it.
		out.SensingAvailable = false
		out.Warnings = append(out.Warnings, "capacity sensing unavailable: "+snapErr.Error())
		return out, nil
	}

	out.SensingAvailable = snapshot.RuntimeTools["nvidia-smi"].Present
	out.Warnings = append(out.Warnings, snapshot.Warnings...)
	if snapshot.ProbeStatuses["nvidia_gpu"] == "not_present" {
		out.Warnings = append(out.Warnings, "nvidia-smi binary not found")
	}

	claimedByGPU := claimedBytesByGPU(listed.Claims)
	out.GPUs = make([]GpuContention, 0, len(snapshot.GPUs))
	for _, gpu := range snapshot.GPUs {
		total := int64(gpu.VRAMBytes)
		used := int64(gpu.VRAMUsedBytes)
		free := total - used
		if free < 0 {
			free = 0
		}
		out.GPUs = append(out.GPUs, GpuContention{
			Index:                    gpu.Index,
			Name:                     gpu.Name,
			TotalBytes:               total,
			UsedBytes:                used,
			FreeBytes:                free,
			ClaimedBytes:             claimedByGPU[gpu.Index],
			MemoryUtilizationPercent: gpu.MemoryUtilizationPercent,
		})
	}
	return out, nil
}

// claimedBytesByGPU sums active vram claim amounts per GPU index. Claims with no
// GPU index are not attributed to any single GPU and are excluded.
func claimedBytesByGPU(claims []capacityapp.ClaimView) map[int]int64 {
	totals := make(map[int]int64)
	for _, c := range claims {
		if c.ResourceKind != engine.ResourceKindVRAM || c.GPUIndex == nil {
			continue
		}
		totals[*c.GPUIndex] += c.AmountBytes
	}
	return totals
}

// ListClaims returns claims, optionally filtered to one owner / active only.
func (s *CapacityService) ListClaims(ctx context.Context, ownerID string, activeOnly bool) ([]capacityapp.ClaimView, error) {
	out, err := s.app.List(ctx, capacityapp.ListRequest{OwnerID: ownerID, ActiveOnly: activeOnly})
	if err != nil {
		return nil, err
	}
	return out.Claims, nil
}

// Reconcile classifies observed GPU consumers against the ledger.
func (s *CapacityService) Reconcile(ctx context.Context) ([]engine.Finding, error) {
	out, err := s.app.Reconcile(ctx)
	if err != nil {
		return nil, err
	}
	return out.Findings, nil
}

// Policy returns all tunable policy levers.
func (s *CapacityService) Policy(ctx context.Context) ([]capacityapp.PolicyEntry, error) {
	out, err := s.app.PolicyGet(ctx, "")
	if err != nil {
		return nil, err
	}
	return out.Entries, nil
}

// SetPolicy validates and persists one lever, returning the full policy.
func (s *CapacityService) SetPolicy(ctx context.Context, key, value string) ([]capacityapp.PolicyEntry, error) {
	out, err := s.app.PolicySet(ctx, key, value)
	if err != nil {
		return nil, err
	}
	return out.Entries, nil
}
