// Responsibility: apply reconciliation retention and registry maintenance.
package orchestration

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"agent-manager/internal/orchestration/obs"

	"github.com/vrooli/envkit-go"
)

func (r *Reconciler) cleanupExpiredEvents(ctx context.Context) (int, error) {
	if r.eventRetention == nil || r.levers.Storage.EventRetentionDays <= 0 {
		return 0, nil
	}
	cutoff := r.now().Add(-time.Duration(r.levers.Storage.EventRetentionDays) * 24 * time.Hour)
	deleted, err := r.eventRetention.DeleteBefore(ctx, cutoff, eventRetentionBatchSize)
	if err != nil {
		return 0, err
	}
	if deleted > 0 {
		r.log().Info("event retention sweep completed", "deleted", deleted, "cutoff", cutoff.UTC().Format(time.RFC3339))
	}
	return deleted, nil
}

func (r *Reconciler) cleanupExpiredArtifacts(ctx context.Context) (int, error) {
	if r.artifactRetention == nil || r.levers.Storage.ArtifactRetentionDays <= 0 {
		return 0, nil
	}
	cutoff := r.now().Add(-time.Duration(r.levers.Storage.ArtifactRetentionDays) * 24 * time.Hour)
	deleted, err := r.artifactRetention.DeleteBefore(ctx, cutoff, eventRetentionBatchSize)
	if err != nil {
		return 0, err
	}
	if deleted > 0 {
		r.log().Info("artifact retention sweep completed", "deleted", deleted, "cutoff", cutoff.UTC().Format(time.RFC3339))
	}
	return deleted, nil
}

// evictImportedCorpus bounds the imported-transcript recall corpus to
// levers.Storage.ImportedCorpusMaxBytes. Imported events are exempt from the
// age sweep (they are the recall corpus), so without this they grow forever —
// a few multi-GB codex rollouts drove the live DB far past its budget. When the
// database's live bytes exceed the budget it evicts the oldest imported runs
// (events + derived search docs; the on-disk transcript survives and is
// re-importable) until back under budget or the per-pass batch is exhausted, so
// a large overage drains across passes rather than one giant transaction.
// Canonical (non-imported) runs are never touched.
func (r *Reconciler) evictImportedCorpus(ctx context.Context) (int, error) {
	target := r.levers.Storage.ImportedCorpusMaxBytes
	if target <= 0 || r.liveBytesFn == nil || r.events == nil || r.runs == nil {
		return 0, nil
	}
	live, err := r.liveBytesFn(ctx)
	if err != nil {
		return 0, err
	}
	if live <= target {
		return 0, nil
	}
	batch := r.levers.Storage.ImportedCorpusEvictionBatch
	if batch <= 0 {
		batch = 25
	}
	ids, err := r.runs.ListImportedRunIDsOldestFirst(ctx, batch)
	if err != nil {
		return 0, err
	}
	evicted := 0
	for _, id := range ids {
		if err := r.events.Delete(ctx, id); err != nil {
			return evicted, err
		}
		if r.searchDeleteRun != nil {
			if err := r.searchDeleteRun(ctx, id.String()); err != nil {
				// A search-doc delete failure leaves reclaimable orphan rows but
				// must not abort eviction; the next pass retries the run's docs.
				r.log().Warn("imported corpus eviction: search delete failed", "run_id", id.String(), obs.KeyError, err.Error())
			}
		}
		// The run row is INTENTIONALLY kept as a 0-event tombstone: its import
		// provenance is what tells the sweep "already adopted, do not re-import".
		// Deleting the row instead makes the sweep re-import the evicted session on
		// its next pass, which eviction then re-evicts — an import⇄evict churn
		// loop. The tombstone is tiny; keeping it is the cost of a stable LRU.
		evicted++
		if cur, measureErr := r.liveBytesFn(ctx); measureErr == nil && cur <= target {
			break
		}
	}
	if evicted > 0 {
		r.log().Info("imported corpus eviction", "evicted", evicted, "live_bytes_before", live, "target_bytes", target)
	}
	return evicted, nil
}

// reclaimSearchIndex merges the conversation-search FTS index to free the
// tombstone space deletes leave behind, but only under storage pressure: FTS5
// optimize is a full-segment merge, so it is gated on live bytes approaching the
// budget rather than run every tick. It runs BEFORE eviction so reclaimed dead
// index is counted first — otherwise eviction would shed live recall to stay
// under budget while dead index still squats on the space.
func (r *Reconciler) reclaimSearchIndex(ctx context.Context) error {
	if r.searchOptimize == nil {
		return nil
	}
	target := r.levers.Storage.ImportedCorpusMaxBytes
	if target > 0 && r.liveBytesFn != nil {
		live, err := r.liveBytesFn(ctx)
		if err == nil && live < target*7/10 {
			// Comfortably under budget; a full-index merge is not worth its cost.
			return nil
		}
	}
	if err := r.searchOptimize(ctx); err != nil {
		return err
	}
	r.log().Info("conversation-search index optimized (reclaimed FTS tombstone space)")
	return nil
}

// cleanupResourceRegistries runs cleanup on all agent resource registries.
// This removes stale entries from the file-based registries that track running agents.
func (r *Reconciler) cleanupResourceRegistries(ctx context.Context) {
	// List of resource CLI commands that maintain agent registries
	resourceCommands := []string{
		"resource-codex",
		"resource-opencode",
	}

	for _, cmd := range resourceCommands {
		// Run agents cleanup to remove stale entries
		cleanupCmd := exec.CommandContext(ctx, cmd, "agents", "cleanup")
		cleanupCmd.Env = []string(envkit.WithOverlay(envkit.Env(os.Environ()), envkit.Resource, nil))
		if err := cleanupCmd.Run(); err != nil {
			// Log but don't fail - cleanup is best-effort
			// The resource might not be installed or the command might not exist
			if !strings.Contains(err.Error(), "executable file not found") {
				r.log().Warn("resource agents cleanup failed", "command", cmd, obs.KeyError, err.Error())
			}
		}
	}
}
