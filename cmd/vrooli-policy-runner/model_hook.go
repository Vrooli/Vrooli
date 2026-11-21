package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	agentharness "github.com/vrooli/agentharness"
	// Importing cliutil registers the repository root with the policy lookup,
	// so the hook finds resources/<runner>/model-policy.json from any cwd.
	_ "github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/cli-core/usagemode"
)

// modelHookDenyExit is the exit status Codex treats as "block this tool call".
const modelHookDenyExit = 2

// maxHookInputBytes bounds what the hook reads from a tool call.
const maxHookInputBytes = 1 << 20

// runModelHook is the PreToolUse hook for spawn_agent. A sub-agent that names a
// model bypasses Agent Manager, so the resource policy's excluded_models cannot
// stop it any other way. The hook is a guard, never a dependency: it denies
// only when it read the call and the policy cleanly and the named model is
// excluded. Every other outcome (bad input, a missing or broken policy, a
// failed log write) allows the call and says so on stderr.
func runModelHook(runner string, stdin io.Reader, stderr io.Writer) error {
	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Fprintf(stderr, "vrooli-policy-runner model-hook: allowed after internal error: %v\n", recovered)
		}
	}()
	raw, err := io.ReadAll(io.LimitReader(stdin, maxHookInputBytes))
	if err != nil {
		fmt.Fprintf(stderr, "vrooli-policy-runner model-hook: allowed, input unreadable: %v\n", err)
		return nil
	}
	var call struct {
		SessionID string `json:"session_id"`
		ToolName  string `json:"tool_name"`
		ToolInput struct {
			Model string `json:"model"`
		} `json:"tool_input"`
	}
	if err := json.Unmarshal(raw, &call); err != nil {
		fmt.Fprintf(stderr, "vrooli-policy-runner model-hook: allowed, input is not JSON: %v\n", err)
		return nil
	}
	model := strings.TrimSpace(call.ToolInput.Model)
	// Codex names the tool "collaborationspawn_agent"; match on the suffix so a
	// namespace change does not silently disable the guard.
	if model == "" || !strings.HasSuffix(call.ToolName, "spawn_agent") {
		return nil
	}
	excluded, err := usagemode.ModelExcluded(runner, model)
	if err != nil {
		fmt.Fprintf(stderr, "vrooli-policy-runner model-hook: allowed, %s model policy unusable: %v\n", runner, err)
		return nil
	}
	if !excluded {
		return nil
	}
	recordModelHookEvent(stderr, map[string]string{"at": time.Now().UTC().Format(time.RFC3339), "runner": runner, "session_id": call.SessionID, "model": model, "action": "deny"})
	fmt.Fprintf(stderr, "Blocked: model %q is excluded by resources/%s/model-policy.json. Omit model and reasoning_effort so the sub-agent inherits this session's model; if the work needs another tier, tell the operator.\n", model, runner)
	return exitError{code: modelHookDenyExit}
}

// recordModelHookEvent appends one JSON line next to the policy bundle so an
// operator can see attempts. A failed write is a warning, not a failure.
func recordModelHookEvent(stderr io.Writer, event map[string]string) {
	dir, err := agentharness.DefaultDataDir()
	if err == nil {
		if err = os.MkdirAll(dir, 0o700); err == nil {
			var line []byte
			if line, err = json.Marshal(event); err == nil {
				var file *os.File
				if file, err = os.OpenFile(filepath.Join(dir, "excluded-model-spawns.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600); err == nil {
					_, err = file.Write(append(line, '\n'))
					if closeErr := file.Close(); err == nil {
						err = closeErr
					}
				}
			}
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "vrooli-policy-runner model-hook: could not record the attempt: %v\n", err)
	}
}
