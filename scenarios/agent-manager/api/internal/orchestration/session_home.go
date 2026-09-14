// This file prepares and cleans isolated runner session-home directories.
package orchestration

import (
	"fmt"
	"os"
	"path/filepath"

	"agent-manager/internal/domain"
	"agent-manager/internal/runstate"

	"github.com/google/uuid"
)

const (
	// RuntimeDirName is the single per-run folder every runner-private path
	// lives under. The run, not the runner, is the unit of sandbox isolation.
	RuntimeDirName = "runtime"
	// RuntimeRootEnvKey carries the per-run runtime folder to the sandbox
	// launcher, which mounts it writable for every runner.
	RuntimeRootEnvKey = "VROOLI_AGENT_RUNTIME_ROOT"
)

// PrepareRunnerRuntimeRoot creates the per-run runtime folder and returns the
// env entry the protected sandbox launcher mounts. Every runner's private state
// (Codex/Grok homes, skill scopes, OpenCode data) lives beneath it.
func PrepareRunnerRuntimeRoot(root string, runID uuid.UUID) (map[string]string, error) {
	runDir, err := runstate.RunDir(root, runID)
	if err != nil {
		return nil, err
	}
	runtimeDir := filepath.Join(runDir, RuntimeDirName)
	if err := os.MkdirAll(runtimeDir, 0o700); err != nil {
		return nil, fmt.Errorf("create run runtime folder: %w", err)
	}
	return map[string]string{RuntimeRootEnvKey: runtimeDir}, nil
}

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
	home := filepath.Join(runDir, RuntimeDirName, subdir)
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
	home := filepath.Join(runDir, RuntimeDirName, subdir)
	var firstErr error
	for _, name := range sessionHomeSeedFiles(runnerType) {
		if err := os.Remove(filepath.Join(home, name)); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// runnerCacheDirs are downloaded, regenerable caches inside a run's private
// homes. Every fresh CODEX_HOME re-fetches the remote plugin catalog (~20 MB)
// and the curated plugin bundles (~27 MB): 19.3 GB of 23.4 GB of run state on
// 2026-09-14. Rollouts, transcripts and state databases are never listed.
var runnerCacheDirs = []string{
	filepath.Join(RuntimeDirName, "codex", "cache"),
	filepath.Join(RuntimeDirName, "codex", "plugins", "cache"),
	// The interactive substrate's run-scoped CODEX_HOME.
	filepath.Join("codex", "cache"),
	filepath.Join("codex", "plugins", "cache"),
}

// PruneRunnerCaches removes a run's regenerable runner caches. A later turn of
// the same run downloads them again.
func PruneRunnerCaches(runDir string) error {
	var firstErr error
	for _, relative := range runnerCacheDirs {
		if err := os.RemoveAll(filepath.Join(runDir, relative)); err != nil && firstErr == nil {
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
