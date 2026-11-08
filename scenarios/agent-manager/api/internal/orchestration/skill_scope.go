// Responsibility: create, link, and reap per-run coding-agent skill scopes.
package orchestration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/runstate"

	"github.com/google/uuid"
)

const skillScopeDirName = "skill-scope"

// openCodeRunDataHome is shared by read-only continuation admission and runtime
// preparation so both select the same native store without creating it to probe.
func openCodeRunDataHome(runDir string) string {
	return filepath.Join(runDir, RuntimeDirName, "opencode-data")
}

// PrepareRunnerSkillScope creates the private runtime configuration root used
// for native skill discovery. The variable that relocates the skills directory
// comes from the harness's resource manifest. It never writes to the project
// directory or the operator's shared runtime home. A harness whose skills no
// variable relocates gets no scope; RunnerSkillsDir then refuses a skill pack
// for it.
func PrepareRunnerSkillScope(root string, runID uuid.UUID, runnerType domain.RunnerType) (map[string]string, error) {
	storage, declared, err := harnessSkillStorage(runnerType)
	if err != nil {
		return nil, err
	}
	// Codex and Grok already receive a private CODEX_HOME/GROK_HOME from
	// PrepareCodecSessionHome; their skill directories live beneath it.
	if !declared || !storage.Relocatable() || storage.RelocationKey == codecSessionHomeKey(runnerType) {
		return nil, nil
	}
	key := storage.RelocationKey
	runDir, err := runstate.RunDir(root, runID)
	if err != nil {
		return nil, err
	}
	scope := filepath.Join(runDir, RuntimeDirName, skillScopeDirName, string(runnerType))
	if err := os.MkdirAll(scope, 0o700); err != nil {
		return nil, fmt.Errorf("create private %s skill scope: %w", runnerType, err)
	}
	if storage.SharedBaseDirectory() {
		if err := mirrorBaseDirectory(storage, scope); err != nil {
			return nil, fmt.Errorf("prepare private %s for %s: %w", key, runnerType, err)
		}
	}
	env := map[string]string{key: scope}
	if runnerType == domain.RunnerTypeClaudeCode {
		if err := linkCredentialIfPresent(filepath.Join(".claude", ".credentials.json"), filepath.Join(scope, ".credentials.json")); err != nil {
			return nil, err
		}
		if err := writeClaudeSkillActivationHook(scope); err != nil {
			return nil, err
		}
	}
	if runnerType == domain.RunnerTypeOpenCode {
		// OpenCode keeps provider data under XDG_DATA_HOME; isolate it in the
		// same run runtime folder so a protected sandbox needs one mount.
		dataHome := openCodeRunDataHome(runDir)
		if err := os.MkdirAll(filepath.Join(dataHome, "opencode"), 0o700); err != nil {
			return nil, fmt.Errorf("create private OpenCode data home: %w", err)
		}
		// Provider credentials stay in the operator's shared store; the run
		// sees them through a link, the same way Claude's credentials reach
		// its private config. Without it every provider resolves as missing.
		sharedDataHome := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
		if sharedDataHome == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("resolve OpenCode credential home: %w", err)
			}
			sharedDataHome = filepath.Join(home, ".local", "share")
		}
		authSource := filepath.Join(sharedDataHome, "opencode", "auth.json")
		authDestination := filepath.Join(dataHome, "opencode", "auth.json")
		// Refresh only the projection on continuation, including when the
		// operator has removed the selected login. Never borrow another store.
		if authSource == authDestination {
			return nil, fmt.Errorf("shared OpenCode auth store is the run data home")
		}
		if err := os.Remove(authDestination); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		if err := linkCredentialSourceIfPresent(authSource, authDestination); err != nil {
			return nil, err
		}
		env["XDG_DATA_HOME"] = dataHome
	}
	return env, nil
}

// mirrorBaseDirectory makes scope stand in for the operator's base directory
// (for example ~/.config): every entry is linked except the path down to the
// harness's skills directory, which is private and starts empty. The harness
// and every other program in the run then read the operator's configuration,
// and only the skills directory differs. Links are refreshed on every call, so
// a continuation sees the operator's current entries.
func mirrorBaseDirectory(storage HarnessSkillStorage, scope string) error {
	source := strings.TrimSpace(os.Getenv(storage.RelocationKey))
	if source == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve user home: %w", err)
		}
		source = filepath.Clean(strings.Replace(storage.DefaultRoot, "$USER_HOME", home, 1))
	}
	if filepath.Clean(source) == filepath.Clean(scope) {
		return fmt.Errorf("shared %s is the run scope", storage.RelocationKey)
	}
	parts := strings.Split(filepath.Clean(storage.RelativePath), string(filepath.Separator))
	target := scope
	for i, part := range parts {
		if err := os.MkdirAll(target, 0o700); err != nil {
			return err
		}
		entries, err := os.ReadDir(source)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		for _, entry := range entries {
			if entry.Name() == part {
				continue
			}
			link := filepath.Join(target, entry.Name())
			if info, statErr := os.Lstat(link); statErr == nil && info.Mode()&os.ModeSymlink == 0 {
				continue // written by the run itself; keep it
			}
			if err := os.Remove(link); err != nil && !os.IsNotExist(err) {
				return err
			}
			if err := os.Symlink(filepath.Join(source, entry.Name()), link); err != nil {
				return err
			}
		}
		source, target = filepath.Join(source, part), filepath.Join(target, part)
		if i == len(parts)-1 {
			if info, statErr := os.Lstat(target); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
				if err := os.Remove(target); err != nil {
					return err
				}
			}
			return os.MkdirAll(target, 0o700)
		}
	}
	return nil
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
	return os.RemoveAll(filepath.Join(runDir, RuntimeDirName, skillScopeDirName, string(runnerType)))
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
		if err := os.RemoveAll(filepath.Join(root, entry.Name(), RuntimeDirName, skillScopeDirName)); err != nil {
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
	return linkCredentialSourceIfPresent(source, destination)
}

func linkCredentialSourceIfPresent(source, destination string) error {
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
