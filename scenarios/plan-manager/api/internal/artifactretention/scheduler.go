package artifactretention

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Settings resolves the owner's environment overrides.
//
//	PLAN_ARTIFACTS_RETENTION_MODE      enforce (default) | report | off
//	PLAN_ARTIFACTS_RETENTION_GRACE     idle window before pruning (default 168h)
//	PLAN_ARTIFACTS_RETENTION_INTERVAL  cycle interval (default 6h)
type Settings struct {
	Mode     Mode
	Disabled bool
	Grace    time.Duration
	Interval time.Duration
}

// SettingsFromEnv parses the overrides; an invalid value is an error rather
// than a silent fallback.
func SettingsFromEnv(getenv func(string) string) (Settings, error) {
	s := Settings{Mode: ModeEnforce, Grace: DefaultGrace, Interval: 6 * time.Hour}
	switch raw := getenv("PLAN_ARTIFACTS_RETENTION_MODE"); raw {
	case "", string(ModeEnforce):
	case string(ModeReport):
		s.Mode = ModeReport
	case "off":
		s.Disabled = true
	default:
		return s, fmt.Errorf("invalid PLAN_ARTIFACTS_RETENTION_MODE %q", raw)
	}
	for _, item := range []struct {
		key string
		dst *time.Duration
	}{{"PLAN_ARTIFACTS_RETENTION_GRACE", &s.Grace}, {"PLAN_ARTIFACTS_RETENTION_INTERVAL", &s.Interval}} {
		raw := getenv(item.key)
		if raw == "" {
			continue
		}
		value, err := time.ParseDuration(raw)
		if err != nil || value <= 0 {
			return s, fmt.Errorf("invalid %s %q", item.key, raw)
		}
		*item.dst = value
	}
	return s, nil
}

// Start runs a cycle immediately and then on every interval until ctx ends.
// Each cycle replaces the receipt at receiptPath.
func Start(ctx context.Context, root, receiptPath string, settings Settings, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	cycle := func() {
		runCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
		defer cancel()
		receipt, err := Run(runCtx, Config{Root: root, Grace: settings.Grace, Mode: settings.Mode, LivePaths: ProcLivePaths})
		logger.Info("plan-artifacts isolated-tree retention cycle", "mode", receipt.Mode, "trees", len(receipt.Trees),
			"freed_bytes", receipt.FreedBytes, "reclaimable_bytes", receipt.ReclaimableBytes,
			"operator_review", len(receipt.OperatorReview), "error", err)
		if writeErr := WriteReceipt(receiptPath, receipt); writeErr != nil {
			logger.Warn("plan-artifacts retention receipt write failed", "path", receiptPath, "error", writeErr)
		}
	}
	go func() {
		cycle()
		ticker := time.NewTicker(settings.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cycle()
			}
		}
	}()
}
