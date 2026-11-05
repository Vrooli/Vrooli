package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"workflow-health/internal/artifacts"
	workflowrun "workflow-health/internal/validationrun"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/retention"
	corestorage "github.com/vrooli/api-core/storage"
)

const (
	runArtifactRetentionOwner = "workflow-health"
	runArtifactReceiptRel     = "retention/run-artifacts-receipt.json"
)

// startRunArtifactRetention runs owner retention over the run-artifact store:
// shortly after start, then every interval (WORKFLOW_HEALTH_RUN_ARTIFACT_
// RETENTION_INTERVAL, default 6h). Each cycle persists a detailed receipt and
// the shared enforcement receipt storage-manager reads.
func startRunArtifactRetention(ctx context.Context, db *database.RoutedDB, repoRoot string, logger *log.Logger) error {
	interval := 6 * time.Hour
	if raw := os.Getenv("WORKFLOW_HEALTH_RUN_ARTIFACT_RETENTION_INTERVAL"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil || parsed <= 0 {
			return fmt.Errorf("invalid run artifact retention interval %q", raw)
		}
		interval = parsed
	}
	root, err := artifacts.RunArtifactsRoot()
	if err != nil {
		return err
	}
	resolver, err := corestorage.NewResolver(corestorage.ResolverConfig{AppID: "vrooli", Profile: corestorage.ProfileAuto})
	if err != nil {
		return err
	}
	receiptPath, err := resolver.Path(corestorage.Options{ScenarioID: runArtifactRetentionOwner}, corestorage.ClassState, runArtifactReceiptRel)
	if err != nil {
		return err
	}
	var citationRoots []string
	if repoRoot != "" {
		citationRoots = []string{filepath.Join(repoRoot, "docs"), filepath.Join(repoRoot, ".vrooli", "work")}
	}
	policy := artifacts.DefaultRetentionPolicy()

	cycle := func() {
		cycleCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
		defer cancel()
		result, err := runArtifactRetentionCycle(cycleCtx, workflowrun.Repository{DB: db}, root, citationRoots, policy, time.Now())
		if result.Root != "" {
			if werr := artifacts.WriteRetentionReceipt(receiptPath, result); werr != nil && err == nil {
				err = werr
			}
		}
		if rerr := retention.RecordEnforcementReceipt(runArtifactRetentionOwner, time.Now().UTC(), receiptError(result, err)); rerr != nil {
			logger.Printf("run artifact retention: record enforcement receipt: %v", rerr)
		}
		logger.Printf("run artifact retention cycle: seen=%d pruned=%d freed_bytes=%d bytes_after=%d over_budget=%t protected=%v err=%v",
			result.TimelinesSeen, len(result.Pruned), result.FreedBytes, result.BytesAfter, result.OverBudget, result.Protected, err)
	}

	go func() {
		// Let startup recovery and health settle before the first walk.
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
	logger.Printf("run artifact retention configured: root=%s interval=%s min_age=%s keep_per_workflow=%d max_bytes=%d receipt=%s",
		root, interval, policy.MinAge, policy.KeepPerWorkflow, policy.MaxBytes, receiptPath)
	return nil
}

type activeRunLister interface {
	ListActive(ctx context.Context) ([]workflowrun.ActiveRun, error)
}

// runArtifactRetentionCycle gathers live protection (active runs, cited
// evidence) and prunes. It refuses to prune when either signal is unreadable:
// an unknown active set or citation set must never widen deletion.
func runArtifactRetentionCycle(ctx context.Context, ledger activeRunLister, root string, citationRoots []string, policy artifacts.RetentionPolicy, now time.Time) (artifacts.RetentionResult, error) {
	active, err := ledger.ListActive(ctx)
	if err != nil {
		return artifacts.RetentionResult{}, fmt.Errorf("list active validation runs: %w", err)
	}
	activeIDs := make(map[string]bool, len(active))
	for _, run := range active {
		activeIDs[run.ID] = true
	}
	cited, err := artifacts.CitedRunIDs(citationRoots...)
	if err != nil {
		return artifacts.RetentionResult{}, err
	}
	return artifacts.PruneTimelines(root, policy, artifacts.RetentionInputs{Now: now, ActiveRuns: activeIDs, CitedRuns: cited})
}

// receiptError makes the enforcement receipt honest: a cycle that leaves the
// store over budget has not enforced it.
func receiptError(result artifacts.RetentionResult, cycleErr error) error {
	if cycleErr != nil || !result.OverBudget {
		return cycleErr
	}
	return fmt.Errorf("run artifacts over budget after cycle: %d of %d bytes, protected %v", result.BytesAfter, result.Policy.MaxBytes, result.Protected)
}
