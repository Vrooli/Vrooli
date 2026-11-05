package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"experience-manager/internal/reconcile"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/retention"
	"github.com/vrooli/api-core/storage"
)

const (
	evidenceRetentionOwner = "experience-manager"
	evidenceReceiptRel     = "retention/evidence-retention-receipt.json"
)

// startEvidenceRetention prunes superseded reconcile evidence shortly after
// start and then every interval (EXPERIENCE_MANAGER_EVIDENCE_RETENTION_INTERVAL,
// default 6h). Candidate scans run on a separate read-only connection; deletes
// go through the API's routed primary in bounded batches. An hourly check runs
// the owner's one-time conversion to incremental auto-vacuum once the backlog
// has drained; the returned compactor lets health stay live during it.
func startEvidenceRetention(ctx context.Context, db *database.RoutedDB, logger *log.Logger) (*evidenceCompactor, error) {
	interval := 6 * time.Hour
	if raw := os.Getenv("EXPERIENCE_MANAGER_EVIDENCE_RETENTION_INTERVAL"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil || parsed <= 0 {
			return nil, fmt.Errorf("invalid evidence retention interval %q", raw)
		}
		interval = parsed
	}
	dbPath, err := storage.SQLitePath(storage.SQLiteConfig{Scenario: evidenceRetentionOwner})
	if err != nil {
		return nil, fmt.Errorf("resolve evidence database path: %w", err)
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return nil, err
	}
	receiptPath, err := resolver.Path(storage.Options{ScenarioID: evidenceRetentionOwner}, storage.ClassState, evidenceReceiptRel)
	if err != nil {
		return nil, err
	}
	compactionReceipt, err := resolver.Path(storage.Options{ScenarioID: evidenceRetentionOwner}, storage.ClassState, "retention/compaction-receipt.json")
	if err != nil {
		return nil, err
	}
	policy := reconcile.DefaultEvidenceRetentionPolicy()
	run := &sync.Mutex{}
	compactor := &evidenceCompactor{
		db: db.Primary, path: dbPath, now: time.Now, freeSpace: diskFreeBytes, receiptPath: compactionReceipt, run: run,
		policy: parseCompactionWindow(os.Getenv("EXPERIENCE_MANAGER_COMPACTION_WINDOW"), defaultEvidenceCompactionPolicy()),
	}

	cycle := func() {
		run.Lock()
		defer run.Unlock()
		cycleCtx, cancel := context.WithTimeout(ctx, 2*time.Hour)
		defer cancel()
		result, err := runEvidenceRetentionCycle(cycleCtx, dbPath, db.Primary(), policy, time.Now())
		compactor.recordRetention(result.Candidates, err)
		if data, merr := json.MarshalIndent(result, "", "  "); merr == nil {
			if werr := storage.WriteFileAtomic(receiptPath, append(data, '\n'), 0o644); werr != nil && err == nil {
				err = werr
			}
		}
		if rerr := retention.RecordEnforcementReceipt(evidenceRetentionOwner, time.Now().UTC(), err); rerr != nil {
			logger.Printf("evidence retention: record enforcement receipt: %v", rerr)
		}
		logger.Printf("evidence retention cycle: candidates=%d deleted=%d kept=%d released_bytes=%d cutoff=%s err=%v",
			result.Candidates, result.DeletedRows, result.Kept, result.ReleasedBytes, result.Cutoff.Format(time.RFC3339), err)
	}

	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				receipt, err := compactor.MaybeCompact(ctx)
				switch {
				case err != nil:
					logger.Printf("evidence compaction failed: %v", err)
				case receipt.Decision == compactionPerformed:
					logger.Printf("evidence compaction converted database to incremental auto-vacuum: reclaimed_bytes=%d duration_ms=%d", receipt.ReclaimedBytes, receipt.DurationMillis)
				case receipt.Decision == compactionDeferred:
					logger.Printf("evidence compaction deferred: %s", receipt.Reason)
				}
			}
		}
	}()

	go func() {
		timer := time.NewTimer(2 * time.Minute)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				cycle()
				timer.Reset(interval)
			}
		}
	}()
	logger.Printf("evidence retention configured: db=%s interval=%s keep_per_claim=%d min_age=%s receipt=%s",
		dbPath, interval, policy.KeepPerClaim, policy.MinAge, receiptPath)
	return compactor, nil
}

func runEvidenceRetentionCycle(ctx context.Context, dbPath string, writer reconcile.SQLExecutor, policy reconcile.EvidenceRetentionPolicy, now time.Time) (reconcile.EvidenceRetentionResult, error) {
	// A second connection (WAL readers never block the writer) pinned to
	// query_only, so the long candidate scan never occupies the API's single
	// pooled connection and cannot write.
	dsn, err := storage.SQLiteDSNAt(dbPath, storage.SQLiteTuning{})
	if err != nil {
		return reconcile.EvidenceRetentionResult{}, err
	}
	pool, err := sql.Open("sqlite", dsn)
	if err != nil {
		return reconcile.EvidenceRetentionResult{}, fmt.Errorf("open evidence reader: %w", err)
	}
	defer pool.Close()
	reader, err := pool.Conn(ctx)
	if err != nil {
		return reconcile.EvidenceRetentionResult{}, fmt.Errorf("connect evidence reader: %w", err)
	}
	defer reader.Close()
	if _, err := reader.ExecContext(ctx, "PRAGMA query_only=1"); err != nil {
		return reconcile.EvidenceRetentionResult{}, fmt.Errorf("pin evidence reader read-only: %w", err)
	}

	result, err := reconcile.PruneSupersededEvidence(ctx, reader, writer, policy, now)
	if err != nil {
		return result, err
	}
	return result, nil
}
