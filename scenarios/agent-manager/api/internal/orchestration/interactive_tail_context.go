// Responsibility: resolve persisted per-run metadata into a scoped interactive-tail context.
package orchestration

import (
	"context"
	"fmt"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/interactive"
	"agent-manager/internal/runstate"

	"github.com/google/uuid"
)

// loadInteractiveTailContext reuses the durable run-state owner for transcript
// rediscovery. Even if the metadata file is unavailable, the canonical run
// directory remains safe to use for Codex's run-scoped rollout search.
func loadInteractiveTailContext(ctx context.Context, runID uuid.UUID, resolveRoot func(context.Context) (string, error)) (interactive.TailContext, error) {
	root, err := resolveRoot(ctx)
	if err != nil {
		return interactive.TailContext{}, err
	}
	if strings.TrimSpace(root) == "" {
		return interactive.TailContext{}, fmt.Errorf("run state root resolver returned an empty path")
	}
	runDir, err := runstate.RunDir(root, runID)
	if err != nil {
		return interactive.TailContext{}, err
	}
	tailContext := interactive.TailContext{RunDir: runDir}
	snapshot, err := runstate.Load(runID, root)
	if err != nil {
		return tailContext, fmt.Errorf("load interactive run-state metadata: %w", err)
	}
	if snapshot.Meta.RunID != runID.String() {
		return tailContext, fmt.Errorf("interactive run-state metadata belongs to %q, want %q", snapshot.Meta.RunID, runID)
	}
	tailContext.WorkingDir = snapshot.Meta.WorkingDir
	tailContext.LaunchedAt = snapshot.Meta.StartedAt
	return tailContext, nil
}

func loadInteractiveTailContextForRun(ctx context.Context, run *domain.Run, resolveRoot func(context.Context) (string, error)) (interactive.TailContext, error) {
	if run == nil {
		return interactive.TailContext{}, fmt.Errorf("interactive run is required")
	}
	tailContext, err := loadInteractiveTailContext(ctx, run.ID, resolveRoot)
	if run.ResolvedConfig != nil && run.ResolvedConfig.RunnerType == domain.RunnerTypeCodex && strings.TrimSpace(tailContext.RunDir) == "" {
		if err != nil {
			return tailContext, fmt.Errorf("Codex interactive run requires its run-scoped transcript directory: %w", err)
		}
		return tailContext, fmt.Errorf("Codex interactive run requires its run-scoped transcript directory")
	}
	return tailContext, err
}
