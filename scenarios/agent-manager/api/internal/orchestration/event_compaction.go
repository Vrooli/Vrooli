package orchestration

import (
	"context"
	"time"

	"agent-manager/internal/adapters/event"
)

// compactImportedEventPayloads bounds imported transcripts' bulky tool
// payloads. Imported events are the conversation-recall corpus and are never
// age-deleted, so compaction, not retention, is what keeps them bounded.
func (r *Reconciler) compactImportedEventPayloads(ctx context.Context) (int, error) {
	days := r.levers.Storage.ImportedToolCompactionDays
	compactor, ok := r.eventRetention.(event.PayloadCompactor)
	if days <= 0 || !ok {
		return 0, nil
	}
	cutoff := r.now().Add(-time.Duration(days) * 24 * time.Hour)
	return compactor.CompactImportedToolPayloads(ctx, cutoff, r.levers.Storage.ImportedToolCompactionMinBytes, eventRetentionBatchSize)
}

func (r *Reconciler) recordImportedEventCompaction(ctx context.Context, stats *ReconcileStats) {
	compacted, err := r.compactImportedEventPayloads(ctx)
	if err != nil {
		stats.Errors = append(stats.Errors, "imported event compaction: "+err.Error())
		return
	}
	if compacted > 0 {
		r.log().Info("imported tool payload compaction completed", "compacted", compacted)
	}
}
