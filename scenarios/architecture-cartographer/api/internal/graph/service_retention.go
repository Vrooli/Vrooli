package graph

// DOC: docs/reference/retention.md

import (
	"context"
	"sort"
)

// defaultServiceRetentionKeep resolves the retention floor for a request.
// Zero means "use the configured default" rather than "keep nothing".
func (s *service) retentionPolicy(keepPerScenario int) RetentionPolicy {
	return RetentionPolicy{KeepPerScenario: keepPerScenario}
}

// PreviewSnapshotRetention reports what a prune would remove, and removes
// nothing.
//
// This is the estimate half of the storage-manager owner-provider contract.
// storage-manager documents that it never duplicates owner-private deletion
// logic, so it asks this scenario what is safe to drop instead of crawling the
// database itself.
func (s *service) PreviewSnapshotRetention(ctx context.Context, keepPerScenario int) (SnapshotRetentionPreview, error) {
	policy := s.retentionPolicy(keepPerScenario)

	bytes, rows, err := s.repo.ReclaimableSnapshotBytes(ctx, policy)
	if err != nil {
		return SnapshotRetentionPreview{}, err
	}
	counts, err := s.repo.SnapshotCounts(ctx)
	if err != nil {
		return SnapshotRetentionPreview{}, err
	}

	keep := policy.keep()
	preview := SnapshotRetentionPreview{
		ReclaimableBytes: bytes,
		ReclaimableRows:  rows,
		KeepPerScenario:  keep,
	}
	for scenario, count := range counts {
		preview.TotalSnapshots += count
		reclaimable := count - keep
		if reclaimable < 0 {
			reclaimable = 0
		}
		preview.Scenarios = append(preview.Scenarios, ScenarioSnapshotCount{
			Scenario:         scenario,
			SnapshotCount:    count,
			ReclaimableCount: reclaimable,
		})
	}
	// Sort by reclaimable descending so the worst offender leads the report;
	// during an incident that is the only line an operator needs.
	sort.Slice(preview.Scenarios, func(i, j int) bool {
		if preview.Scenarios[i].ReclaimableCount != preview.Scenarios[j].ReclaimableCount {
			return preview.Scenarios[i].ReclaimableCount > preview.Scenarios[j].ReclaimableCount
		}
		return preview.Scenarios[i].Scenario < preview.Scenarios[j].Scenario
	})
	return preview, nil
}

// ApplySnapshotRetention prunes snapshots beyond the retention floor.
func (s *service) ApplySnapshotRetention(ctx context.Context, keepPerScenario int) (RetentionResult, error) {
	return s.repo.PruneSnapshots(ctx, s.retentionPolicy(keepPerScenario))
}
