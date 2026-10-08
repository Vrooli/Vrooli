package facts

import (
	"context"
	"database/sql"
	"time"

	internalfacts "code-facts/internal/facts"
	"code-facts/internal/indexcontrol"
)

type Admission = internalfacts.WeightedAdmission

func NewAdmission() *Admission { return internalfacts.NewWeightedAdmission(16, 64, 2*time.Second) }

func Schema() string {
	return internalfacts.CacheSchema()
}

func DefaultCacheMaxBytes() int64 {
	return internalfacts.DefaultCacheMaxBytes
}

func MigrateSchema(ctx context.Context, db *sql.DB) error {
	return internalfacts.MigrateCacheSchema(ctx, db)
}

func SweepCache(ctx context.Context, db *sql.DB, maxBytes int64) (internalfacts.CacheSweepResult, error) {
	return internalfacts.NewSQLiteCacheRepository(db, maxBytes).Sweep(ctx)
}

func CacheMetrics(ctx context.Context, db *sql.DB, maxBytes int64) (map[string]any, error) {
	stats, err := internalfacts.NewSQLiteCacheRepository(db, maxBytes).Stats(ctx)
	if err != nil {
		return nil, err
	}
	utilization := float64(0)
	if stats.BudgetBytes > 0 {
		utilization = float64(stats.TotalPayloadBytes) / float64(stats.BudgetBytes)
	}
	metrics := map[string]any{
		"cache_total_rows":          stats.TotalRows,
		"cache_total_payload_bytes": stats.TotalPayloadBytes,
		"cache_budget_bytes":        stats.BudgetBytes,
		"cache_utilization":         utilization,
		"cache_last_sweep_at_unix":  stats.LastSweepAtUnix,
		"indexed_count":             stats.TotalRows,
	}
	if stats.LastSweepAtUnix > 0 {
		metrics["last_indexed_at"] = time.Unix(stats.LastSweepAtUnix, 0).UTC().Format(time.RFC3339)
	}
	return metrics, nil
}

func OperationalMetrics(ctx context.Context, db *sql.DB, maxBytes int64, admission *Admission) (map[string]any, error) {
	metrics, err := CacheMetrics(ctx, db, maxBytes)
	if err != nil {
		return nil, err
	}
	status, err := indexcontrol.NewSQLiteStatusReader(db, indexcontrol.NewSQLiteJobStore(db)).Status(ctx)
	if err != nil {
		return nil, err
	}
	metrics["index_state"] = status.State
	metrics["active_generation"] = status.ActiveGeneration
	metrics["source_files"] = status.SourceFiles
	metrics["search_documents"] = status.SearchDocuments
	metrics["semantic_cards"] = status.SemanticCards
	metrics["graph_facts"] = status.GraphFacts
	metrics["index_storage_bytes"] = status.StorageBytes
	metrics["active_index_jobs"] = len(status.ActiveJobs)
	metrics["degraded_reasons"] = status.Degraded
	metrics["indexed_count"] = status.SearchDocuments
	delete(metrics, "last_indexed_at")
	if !status.LastReconcileAt.IsZero() {
		metrics["last_indexed_at"] = status.LastReconcileAt.UTC().Format(time.RFC3339)
	}
	queue := admission.Snapshot()
	metrics["admission_capacity"] = queue.Capacity
	metrics["admission_in_use"] = queue.InUse
	metrics["admission_high_water"] = queue.HighWater
	metrics["admission_queued"] = queue.Queued
	metrics["admission_queue_high_water"] = queue.QueueHighWater
	metrics["admission_admitted_total"] = queue.Admitted
	metrics["admission_rejected_total"] = queue.Rejected
	metrics["admission_cancelled_total"] = queue.Cancelled
	metrics["admission_wait_p50_ms"] = queue.WaitP50MS
	metrics["admission_wait_p95_ms"] = queue.WaitP95MS
	metrics["admission_wait_p99_ms"] = queue.WaitP99MS
	return metrics, nil
}
