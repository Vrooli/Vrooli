// Responsibility: create, link, and reap per-run coding-agent skill scopes.
package orchestration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"agent-manager/internal/domain"
	"agent-manager/internal/runstate"

	"github.com/google/uuid"
)

const skillScopeDirName = "skill-scope"

// PrepareRunnerSkillScope creates the private runtime configuration root used
// for native skill discovery. It never writes to the project directory or the
// operator's shared runtime home.
func PrepareRunnerSkillScope(root string, runID uuid.UUID, runnerType domain.RunnerType) (map[string]string, error) {
	key := ""
	switch runnerType {
	case domain.RunnerTypeClaudeCode:
		key = "CLAUDE_CONFIG_DIR"
	case domain.RunnerTypeOpenCode:
		key = "OPENCODE_CONFIG_DIR"
	case domain.RunnerTypeAntigravity:
		key = "ANTIGRAVITY_STATE_DIR"
	default:
		// Codex and Grok already receive a private CODEX_HOME/GROK_HOME from
		// PrepareCodecSessionHome; their skill directories live beneath it.
		return nil, nil
	}
	runDir, err := runstate.RunDir(root, runID)
	if err != nil {
		return nil, err
	}
	scope := filepath.Join(runDir, skillScopeDirName, string(runnerType))
	if err := os.MkdirAll(scope, 0o700); err != nil {
		return nil, fmt.Errorf("create private %s skill scope: %w", runnerType, err)
	}
	if runnerType == domain.RunnerTypeClaudeCode {
		if err := linkCredentialIfPresent(filepath.Join(".claude", ".credentials.json"), filepath.Join(scope, ".credentials.json")); err != nil {
			return nil, err
		}
		if err := writeClaudeSkillActivationHook(scope); err != nil {
			return nil, err
		}
	}
	return map[string]string{key: scope}, nil
}

// writeClaudeSkillActivationHook installs telemetry in the private run
// config. prompt-manager ignores non-skill events and its CLI path is
// explicitly time-boxed and fail-open, so the hook cannot block a turn.
func writeClaudeSkillActivationHook(scope string) error {
	settings := map[string]any{}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve Claude settings home: %w", err)
	}
	sharedSettings := filepath.Join(home, ".claude", "settings.json")
	if data, readErr := os.ReadFile(sharedSettings); readErr == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parse shared Claude settings: %w", err)
		}
	} else if !os.IsNotExist(readErr) {
		return fmt.Errorf("read shared Claude settings: %w", readErr)
	}
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		hooks = map[string]any{}
	}
	preToolUse, ok := hooks["PreToolUse"].([]any)
	if !ok {
		preToolUse = []any{}
	}
	preToolUse = append(preToolUse, map[string]any{
		"matcher": "*",
		"hooks": []any{map[string]string{
			"type": "command", "command": "prompt-manager skill activation-hook",
		}},
	})
	hooks["PreToolUse"] = preToolUse
	settings["hooks"] = hooks
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode Claude skill activation hook: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(scope, "settings.json"), data, 0o600); err != nil {
		return fmt.Errorf("write Claude skill activation hook: %w", err)
	}
	return nil
}

// CleanupRunnerSkillScope removes only the generated private configuration
// root. It is idempotent and safe to call from terminal and recovery paths.
func CleanupRunnerSkillScope(root string, runID uuid.UUID, runnerType domain.RunnerType) error {
	runDir, err := runstate.RunDir(root, runID)
	if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(runDir, skillScopeDirName, string(runnerType)))
}

// SweepOrphanedSkillScopes removes scope directories for run IDs that no
// longer have a run-state directory. The caller supplies the set of live run
// IDs from the durable repository, so a crash cannot leave native discovery
// state behind indefinitely.
func SweepOrphanedSkillScopes(root string, live map[uuid.UUID]bool) error {
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id, parseErr := uuid.Parse(entry.Name())
		if parseErr != nil || live[id] {
			continue
		}
		if err := os.RemoveAll(filepath.Join(root, entry.Name(), skillScopeDirName)); err != nil {
			return err
		}
	}
	return nil
}

func linkCredentialIfPresent(relative, destination string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve credential home: %w", err)
	}
	source := filepath.Join(home, relative)
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	if err := os.Remove(destination); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Symlink(source, destination); err != nil {
		return fmt.Errorf("link credential %s: %w", destination, err)
	}
	return nil
}
