// Responsibility: apply reconciliation retention and registry maintenance.
package orchestration

import (
	"context"
	"errors"
	"fmt"
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

// cleanupResourceRegistries runs cleanup on all agent resource registries.
// This removes stale entries from the file-based registries that track running agents.
func (r *Reconciler) cleanupResourceRegistries(ctx context.Context) {
	if err := r.processEffects.CleanupRunnerResources(ctx); err != nil {
		r.log().Warn("resource registry cleanup failed", obs.KeyError, err)
	}
}

func cleanupHostResourceRegistries(ctx context.Context) error {
	// List of resource CLI commands that maintain agent registries
	resourceCommands := []string{
		"resource-codex",
		"resource-opencode",
	}

	var failures []error
	for _, cmd := range resourceCommands {
		// Run agents cleanup to remove stale entries
		cleanupCmd := exec.CommandContext(ctx, cmd, "agents", "cleanup")
		cleanupCmd.Env = []string(envkit.WithOverlay(envkit.Env(os.Environ()), envkit.Resource, nil))
		if err := cleanupCmd.Run(); err != nil {
			// Log but don't fail - cleanup is best-effort
			// The resource might not be installed or the command might not exist
			if !strings.Contains(err.Error(), "executable file not found") {
				failures = append(failures, fmt.Errorf("%s agents cleanup: %w", cmd, err))
			}
		}
	}
	return errors.Join(failures...)
}
