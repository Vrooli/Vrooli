package graph

// DOC: docs/reference/storage-retention.md

import (
	"context"
	"fmt"

	"github.com/vrooli/api-core/retention"
)

// SnapshotBudgetName is the manifest key of the graph_snapshots budget, and
// therefore the name this pruner registers under. One name, declared once in
// .vrooli/service.json, used for the registry lookup, the log line, and the
// finding.
const SnapshotBudgetName = "graph_snapshots"

// SnapshotPruner adapts the keep-newest-N-per-scenario rule to the framework
// retention seam.
//
// This is the case the `custom` pruner mode exists for. The rule cannot be
// expressed as an age bound: graph_snapshots holds every distinct code state a
// scenario has ever had, and a generic "delete anything older than 30 days"
// would delete the ONLY snapshot of a stable scenario while keeping twenty of a
// noisy one. That is a correctness regression, not a tuning difference — the
// stable scenario loses its entire architecture history while the noisy one
// keeps redundant copies.
//
// The split of responsibility: the framework owns whether graph_snapshots is
// within its declared budget and reports which bound bound it; this type owns
// which rows die.
type SnapshotPruner struct {
	repo   SnapshotRetentionRepository
	policy RetentionPolicy
}

// SnapshotRetentionRepository is the narrow storage surface the pruner needs.
// Narrowing it keeps the pruner testable without a database.
type SnapshotRetentionRepository interface {
	PruneSnapshots(ctx context.Context, policy RetentionPolicy) (RetentionResult, error)
	SnapshotPayloadBytes(ctx context.Context) (int64, error)
	SnapshotCounts(ctx context.Context) (map[string]int, error)
}

// NewSnapshotPruner builds the pruner over the keep-N policy.
func NewSnapshotPruner(repo SnapshotRetentionRepository, policy RetentionPolicy) *SnapshotPruner {
	return &SnapshotPruner{repo: repo, policy: policy}
}

// Measure reports what graph_snapshots currently holds.
//
// Bytes is the live snapshot payload rather than the database file size, so the
// number the budget is judged against is the thing this pruner can actually act
// on. Items is the snapshot count across all scenarios.
func (p *SnapshotPruner) Measure(ctx context.Context) (retention.Usage, error) {
	counts, err := p.repo.SnapshotCounts(ctx)
	if err != nil {
		return retention.Usage{}, fmt.Errorf("snapshot counts: %w", err)
	}
	var items int64
	for _, count := range counts {
		items += int64(count)
	}

	// The TOTAL live payload, not the reclaimable portion. ReclaimableSnapshotBytes
	// with a zero policy would report only what lies beyond the keep-N floor,
	// because RetentionPolicy deliberately treats KeepPerScenario < 1 as the
	// default floor rather than as "keep nothing".
	bytes, err := p.repo.SnapshotPayloadBytes(ctx)
	if err != nil {
		return retention.Usage{}, fmt.Errorf("snapshot payload bytes: %w", err)
	}
	return retention.Usage{Bytes: bytes, Items: items}, nil
}

// Prune applies the keep-newest-N-per-scenario rule and reports the result in
// the framework's vocabulary.
//
// The declared budget is honoured but never overrides the selection rule: a
// scenario with exactly one snapshot keeps that snapshot no matter how far over
// the byte ceiling the table is. When the rule cannot bring the table under its
// ceiling, the framework reports BoundBytes and raises a finding, which is the
// correct outcome — it names a producer that is outgrowing its budget rather
// than silently deleting history the rule says to keep.
func (p *SnapshotPruner) Prune(ctx context.Context, b retention.Budget) (retention.Result, error) {
	before, err := p.Measure(ctx)
	if err != nil {
		return retention.Result{Budget: b.Name}, err
	}

	pruned, err := p.repo.PruneSnapshots(ctx, p.policy)
	if err != nil {
		return retention.Result{Budget: b.Name, Before: before, Incomplete: true}, fmt.Errorf("prune snapshots: %w", err)
	}

	after, err := p.Measure(ctx)
	if err != nil {
		return retention.Result{Budget: b.Name, Before: before}, err
	}

	result := retention.Result{
		Budget:     b.Name,
		Deleted:    int64(pruned.RowsRemoved),
		FreedBytes: pruned.BytesReclaimed,
		Before:     before,
		After:      after,
		BoundBy:    retention.BoundNone,
	}
	if b.HasByteBound() && after.Bytes > b.MaxBytes {
		// The keep-N rule ran to completion and the table is still over its
		// ceiling. That is a statement about the producer, not a failure here.
		result.BoundBy = retention.BoundBytes
	}
	return result, nil
}

// KeepPerScenario reports the retention floor this pruner enforces.
func (p *SnapshotPruner) KeepPerScenario() int { return p.policy.keep() }
