package main

import (
	"context"
	"log/slog"
	"os"

	"plan-manager/internal/artifactretention"

	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
)

// planArtifactsReceipt is the state-class path of the isolated-tree retention
// receipt; .vrooli/service.json declares it as storage entry
// plan_artifacts_retention_receipt.
const planArtifactsReceipt = "retention/isolated-tree-receipt.json"

// storage-manager:allow-no-writer plan_artifacts — the runtime-home
// plan_artifacts root is pinned by .vrooli/repo-contract.json and written by
// planning and effort agents that Plan Manager routes there through
// internal/artifacts.Instruction; this file owns its retention.

// startPlanArtifactsRetention bounds regenerable build outputs inside idle
// isolated source-tree copies under the plan-artifacts runtime-home root (see
// internal/artifactretention). The root is shared by every instance, so only
// the live instance enforces; a variant would otherwise prune trees it does
// not own. Failure to configure is logged, never fatal.
func startPlanArtifactsRetention(ctx context.Context, logger *slog.Logger) {
	ns, err := storage.ResolveNamespace(storage.NamespaceConfig{FallbackScenario: "plan-manager"})
	if err != nil {
		logger.Warn("plan-artifacts retention disabled: namespace unresolved", "error", err)
		return
	}
	if !ns.IsLive() {
		logger.Info("plan-artifacts retention skipped: non-live variant", "variant", ns.Variant())
		return
	}
	settings, err := artifactretention.SettingsFromEnv(os.Getenv)
	if err != nil {
		logger.Warn("plan-artifacts retention disabled: invalid settings", "error", err)
		return
	}
	if settings.Disabled {
		logger.Info("plan-artifacts retention disabled by PLAN_ARTIFACTS_RETENTION_MODE=off")
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		logger.Warn("plan-artifacts retention disabled: home unresolved", "error", err)
		return
	}
	root, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyPlanArtifacts)
	if err != nil {
		logger.Warn("plan-artifacts retention disabled: runtime-home contract unresolved", "error", err)
		return
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{})
	if err != nil {
		logger.Warn("plan-artifacts retention disabled: storage resolver", "error", err)
		return
	}
	receiptPath, err := resolver.Path(storage.Options{ScenarioID: ns.Root()}, storage.ClassState, planArtifactsReceipt)
	if err != nil {
		logger.Warn("plan-artifacts retention disabled: receipt path", "error", err)
		return
	}
	artifactretention.Start(ctx, root, receiptPath, settings, logger)
	logger.Info("plan-artifacts isolated-tree retention configured", "root", root, "mode", settings.Mode,
		"grace", settings.Grace, "interval", settings.Interval, "receipt", receiptPath)
}
