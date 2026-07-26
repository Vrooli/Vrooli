// This file prepares and cleans isolated runner session-home directories.
package orchestration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"agent-manager/internal/codexgoals"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/runstate"

	"github.com/google/uuid"
)

// PrepareCodecSessionHome creates the durable, run-scoped session home used by
// codec-pipe Codex and Grok runs. The run directory lives under the runtime
// home, which the protected sandbox mounts at the same path, so the CLI can
// write rollouts without relying on its ephemeral sandbox HOME overlay.
//
// The returned environment intentionally overrides any inherited home setting.
// Claude keeps its authenticated shared home and OpenCode has no file-backed
// continuation home.
func PrepareCodecSessionHome(root string, runID uuid.UUID, runnerType domain.RunnerType) (map[string]string, error) {
	var envKey, subdir, sharedHome string
	switch runnerType {
	case domain.RunnerTypeCodex:
		envKey, subdir, sharedHome = "CODEX_HOME", "codex", ".codex"
	case domain.RunnerTypeGrok:
		envKey, subdir, sharedHome = "GROK_HOME", "grok", ".grok"
	default:
		return nil, nil
	}

	runDir, err := runstate.RunDir(root, runID)
	if err != nil {
		return nil, err
	}
	home := filepath.Join(runDir, subdir)
	if err := os.MkdirAll(home, 0o700); err != nil {
		return nil, fmt.Errorf("create run-scoped %s: %w", envKey, err)
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve shared %s home: %w", envKey, err)
	}
	for _, name := range sessionHomeSeedFiles(runnerType) {
		if err := copySessionHomeSeed(filepath.Join(userHome, sharedHome, name), filepath.Join(home, name)); err != nil {
			return nil, fmt.Errorf("seed %s for run-scoped %s: %w", name, envKey, err)
		}
	}
	return map[string]string{envKey: home}, nil
}

// EmitCodexGoalUsage records Codex's independently-maintained goal budget
// after a terminal run. Missing stores/rows are intentionally silent: goals
// are authored only inside Codex and may not exist for a thread.
func EmitCodexGoalUsage(ctx context.Context, root string, deps phases.Deps, run *domain.Run) {
	if run == nil || run.ResolvedConfig == nil || run.ResolvedConfig.RunnerType != domain.RunnerTypeCodex || run.SessionID == "" {
		return
	}
	runDir, err := runstate.RunDir(root, run.ID)
	if err != nil {
		return
	}
	home := filepath.Join(runDir, "codex")
	goal, err := codexgoals.Read(ctx, home, run.SessionID)
	if err != nil {
		phases.EmitSystemEvent(ctx, deps, run.ID, "warn", "failed to read Codex goal accounting: "+err.Error())
		return
	}
	if goal == nil {
		return
	}
	budget := "unset"
	if goal.TokenBudget != nil {
		budget = fmt.Sprintf("%d", *goal.TokenBudget)
	}
	phases.EmitSystemEvent(ctx, deps, run.ID, "info", fmt.Sprintf(
		"codex goal accounting: goal_id=%s status=%s token_budget=%s tokens_used=%d time_used_seconds=%d",
		goal.GoalID, goal.Status, budget, goal.TokensUsed, goal.TimeUsedSeconds,
	))
}

// CleanupCodecSessionHomeCredentials removes only copied credential/config
// files after a terminal run. Rollouts remain in the run directory for replay
// and diagnosis; a later continuation can safely reseed credentials.
func CleanupCodecSessionHomeCredentials(root string, runID uuid.UUID, runnerType domain.RunnerType) error {
	var subdir string
	switch runnerType {
	case domain.RunnerTypeCodex:
		subdir = "codex"
	case domain.RunnerTypeGrok:
		subdir = "grok"
	default:
		return nil
	}
	runDir, err := runstate.RunDir(root, runID)
	if err != nil {
		return err
	}
	home := filepath.Join(runDir, subdir)
	var firstErr error
	for _, name := range sessionHomeSeedFiles(runnerType) {
		if err := os.Remove(filepath.Join(home, name)); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func sessionHomeSeedFiles(runnerType domain.RunnerType) []string {
	if runnerType == domain.RunnerTypeCodex {
		return []string{"auth.json", "config.toml"}
	}
	return nil
}

func copySessionHomeSeed(src, dst string) error {
	data, err := os.ReadFile(src)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o600)
}
