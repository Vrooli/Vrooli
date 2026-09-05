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
	"scenario-to-desktop-api/pipeline"
)

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
	if err != nil { return err }
	owner.Root = root
	runner := scheduler.New(interval, func(ctx context.Context) error {
		cycle, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		result, err := owner.Prune(cycle, budget)
		logger.Info("desktop staging retention cycle", "used_bytes", result.After.Bytes,
			"max_bytes", budget.MaxBytes, "freed_bytes", result.FreedBytes, "deleted", result.Deleted,
			"incomplete", result.Incomplete, "error", err)
		if receiptErr := retention.RecordEnforcementReceipt(namespace, time.Now().UTC(), err); receiptErr != nil {
			return receiptErr
		}
		return err
	})
	runner.Start(ctx)
	logger.Info("desktop staging retention configured", "root", filepath.Clean(root), "max_bytes", budget.MaxBytes, "max_age", budget.MaxAge, "interval", interval)
	return nil
}
