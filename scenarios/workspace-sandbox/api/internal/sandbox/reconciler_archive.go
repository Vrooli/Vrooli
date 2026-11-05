package sandbox

// reconciler_archive.go — diff-archive retention reconciler.
//
// Plugs into the same Runner the lifecycle/heal/orphan/daemon-reaper
// reconcilers use. The actual eviction logic lives on Service
// (ReconcileArchiveRetention); this file is a thin adapter that pulls
// the current retention policy from a provider and shapes the result
// into a ReconcileReport for the dispatcher.
//
// The provider indirection (RetentionPolicyProvider) is what lets
// PUT /config/retention take effect on the next tick without re-wiring
// the reconciler graph: the handler updates the store, the provider
// reads from the store on every tick. No restart, no race.

import (
	"context"
	"log"
)

// RetentionPolicyProvider returns the current retention policy. The
// reconciler calls it on every Run so updates to the policy (typically
// via PUT /config/retention) propagate to the next tick automatically.
//
// Returning the zero RetentionPolicy disables every lever, which is
// equivalent to "do nothing." That is the safe default if the provider
// hits an internal error and wants to skip eviction this pass.
type RetentionPolicyProvider func() RetentionPolicy

// ArchiveRetentionReconciler enforces RetentionPolicy on each tick.
type ArchiveRetentionReconciler struct {
	svc      *Service
	provider RetentionPolicyProvider
}

// NewArchiveRetentionReconciler constructs the reconciler. provider
// must be non-nil; pass a closure that reads from the retention store.
func NewArchiveRetentionReconciler(svc *Service, provider RetentionPolicyProvider) *ArchiveRetentionReconciler {
	if svc == nil {
		panic("sandbox.NewArchiveRetentionReconciler: service is required")
	}
	if provider == nil {
		panic("sandbox.NewArchiveRetentionReconciler: provider is required")
	}
	return &ArchiveRetentionReconciler{svc: svc, provider: provider}
}

// Name is the stable identifier for the admin endpoint and metrics.
func (r *ArchiveRetentionReconciler) Name() string { return "archive-retention" }

// Run executes one retention pass and folds the per-pass counters into
// the dispatcher's ReconcileReport shape.
func (r *ArchiveRetentionReconciler) Run(ctx context.Context) ReconcileReport {
	if r == nil || r.svc == nil || r.provider == nil {
		return ReconcileReport{}
	}
	policy := r.provider()
	if policy == (RetentionPolicy{}) {
		// All levers disabled — fast path, no DB query.
		return ReconcileReport{}
	}
	report := r.svc.ReconcileArchiveRetention(ctx, policy)

	out := ReconcileReport{
		ItemsProcessed: report.Scanned,
		ItemsFailed:    report.BlobFailures,
		Duration:       report.Duration,
		LastError:      report.LastError,
		Details: map[string]any{
			"evictedAge":        report.EvictedAge,
			"evictedSize":       report.EvictedSize,
			"evictedPerProject": report.EvictedPerProject,
			"totalEvicted":      report.TotalEvicted(),
			"blobFailures":      report.BlobFailures,
		},
	}
	if report.TotalEvicted() > 0 || report.BlobFailures > 0 {
		log.Printf(
			"archive-retention: scanned=%d evicted=age:%d/size:%d/perProject:%d blobFailures=%d duration=%s",
			report.Scanned, report.EvictedAge, report.EvictedSize, report.EvictedPerProject,
			report.BlobFailures, report.Duration,
		)
	}
	return out
}

// Compile-time guard.
var _ Reconciler = (*ArchiveRetentionReconciler)(nil)
