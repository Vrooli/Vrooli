package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/vrooli/api-core/retention"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/packages/scheduler"
	"scenario-to-desktop-api/build"
	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/livedesktop"
	"scenario-to-desktop-api/pipeline"
)

// stagingInUse reports which live owner protects a staging entry. Every input
// must reflect work live in this process: the build stores and live desktop
// sessions are in-memory only, and the smoke-test store reconciles unfinished
// records from earlier processes at load. A persisted-only "running" record
// would otherwise protect an app's staging forever.
func stagingInUse(
	smokeTests interface{ HasActiveForScenario(string) bool },
	generationBuilds interface {
		Get(string) (*generation.BuildStatus, bool)
	},
	builds interface {
		Snapshot() map[string]*build.Status
	},
	sessions func() []*livedesktop.Session,
) func(app, path string) string {
	return func(app, path string) string {
		if smokeTests.HasActiveForScenario(app) {
			return "smoke_test"
		}
		if status, ok := generationBuilds.Get(filepath.Base(path)); ok && status != nil && status.Status == "building" {
			return "generation_build"
		}
		for _, status := range builds.Snapshot() {
			if status != nil && status.ScenarioName == app && status.Status == "building" {
				return "build"
			}
		}
		for _, session := range sessions() {
			view := session.View()
			if view.ScenarioName == app && view.State != livedesktop.StateStopped && view.State != livedesktop.StateError {
				return "live_desktop"
			}
		}
		return ""
	}
}

func startStagingRetention(ctx context.Context, manifestPath, root string, owner pipeline.StagingRetention, logger *slog.Logger) error {
	budget, err := pipeline.StagingBudget(manifestPath)
	if err != nil {
		return err
	}
	interval := 15 * time.Minute
	if raw := os.Getenv("DESKTOP_STAGING_RETENTION_INTERVAL"); raw != "" {
		interval, err = time.ParseDuration(raw)
		if err != nil || interval <= 0 {
			return fmt.Errorf("invalid staging retention interval %q", raw)
		}
	}
	namespace, err := storage.ScenarioNamespace("scenario-to-desktop")
	if err != nil {
		return err
	}
	owner.Root = root
	runner := scheduler.New(interval, func(ctx context.Context) error {
		cycle, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		result, err := owner.Prune(cycle, budget)
		logger.Info("desktop staging retention cycle", "used_bytes", result.After.Bytes,
			"max_bytes", budget.MaxBytes, "freed_bytes", result.FreedBytes, "deleted", result.Deleted,
			"incomplete", result.Incomplete, "protected", result.ProtectedTotal(),
			"protected_reasons", result.ProtectedSummary(), "error", err)
		if receiptErr := retention.RecordEnforcementReceipt(namespace, time.Now().UTC(), stagingReceiptError(result, budget, err)); receiptErr != nil {
			return receiptErr
		}
		return err
	})
	runner.Start(ctx)
	logger.Info("desktop staging retention configured", "root", filepath.Clean(root), "max_bytes", budget.MaxBytes, "max_age", budget.MaxAge, "interval", interval)
	return nil
}

// stagingReceiptError makes the durable receipt honest: a cycle that leaves
// staging over its byte budget has not enforced it, and the receipt names
// which owners protected the remaining entries. Like the storage-manager
// generic enforcer, an incomplete reconciliation is recorded as an error;
// the scheduler still only sees real cycle failures.
func stagingReceiptError(result pipeline.StagingResult, budget retention.Budget, cycleErr error) error {
	if cycleErr != nil || !result.OverBudget(budget) {
		return cycleErr
	}
	protected := "none"
	if result.ProtectedTotal() > 0 {
		protected = result.ProtectedSummary()
	}
	return fmt.Errorf("staging over budget after cycle: used %d of %d bytes, deleted %d, %d over-budget entries protected (%s)",
		result.After.Bytes, budget.MaxBytes, result.Deleted, result.ProtectedTotal(), protected)
}
