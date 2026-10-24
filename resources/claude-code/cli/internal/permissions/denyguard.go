package permissions

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/agentharness"
)

// The PreToolUse guard is Claude Code's entry into the shared agent policy
// runtime, the same one Codex, OpenCode, and Grok reach through
// vrooli-policy-runner. Filesystem deletion is decided there by resolved path;
// the remaining Bash deny patterns are matched here as a native backstop.
// Command text is data: the guard parses it and never executes it.

// Claude Code's hook contract: exit zero continues (optionally with a JSON
// decision on stdout), exit two denies with stderr as the reason.
const (
	GuardExitContinue = 0
	GuardExitDeny     = 2
)

// Operator-tunable log limits, matching the environment variable names the
// hook has always honoured.
const (
	defaultGuardLogMaxBytes    = 8 << 20
	defaultGuardLogCommandMax  = 300
	guardLogMaxBytesEnv        = "VROOLI_HOOK_LOG_MAX_BYTES"
	guardLogCommandMaxCharsEnv = "VROOLI_HOOK_LOG_CMD_MAX_CHARS"
	// guardPolicyModeEnv selects the rollout profile, as it does for
	// vrooli-policy-runner.
	guardPolicyModeEnv = "VROOLI_AGENT_POLICY_MODE"
)

// interactivePermissionModes are the Claude Code modes in which a hook's "ask"
// reaches a person. In any other mode (bypassPermissions, a headless run, an
// unknown future mode) nobody would see the prompt, so ask becomes deny.
var interactivePermissionModes = map[string]bool{"default": true, "acceptEdits": true, "plan": true}

// GuardEnv is the resolved context a single guard decision reads. It is a
// value rather than ambient process state so the decision is testable in
// process and behaves identically when invoked as a hook.
type GuardEnv struct {
	// Home resolves home references in native deny patterns.
	Home string
	// Runtime is the shared agent policy runtime that decides the event.
	Runtime agentharness.Runtime
	// LogPath is the append-only audit log for hook decisions.
	LogPath string
	// LogMaxBytes is the size at which the audit log rotates.
	LogMaxBytes int64
	// LogCommandMax caps how much command text one log line carries.
	LogCommandMax int
}

// LoadGuardEnv resolves the guard context from the process environment.
func LoadGuardEnv() GuardEnv {
	home, _ := os.UserHomeDir()
	runtime := agentharness.Runtime{Profile: agentharness.RolloutProfile(strings.ToLower(strings.TrimSpace(os.Getenv(guardPolicyModeEnv))))}
	if runtime.Profile == "" {
		runtime.Profile = agentharness.ProfileAdvisory
	}
	// Without a store the runtime reports that no provider rules apply and the
	// floor decides alone, which is stricter, never looser.
	if dir, err := agentharness.DefaultDataDir(); err == nil {
		runtime.Store = agentharness.NewBundleStore(dir)
	}
	return GuardEnv{
		Home:          home,
		Runtime:       runtime,
		LogPath:       filepath.Join(home, ".claude", HookStateDirName, "log"),
		LogMaxBytes:   positiveEnvInt(guardLogMaxBytesEnv, defaultGuardLogMaxBytes),
		LogCommandMax: int(positiveEnvInt(guardLogCommandMaxCharsEnv, defaultGuardLogCommandMax)),
	}
}

func positiveEnvInt(name string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	var value int64
	if _, err := fmt.Sscanf(raw, "%d", &value); err != nil || value <= 0 {
		return fallback
	}
	return value
}

// HookEvent is the part of a PreToolUse payload the guard reads.
type HookEvent struct {
	Command          string
	WorkingDirectory string
	PermissionMode   string
}

// ExtractHookEvent reads a PreToolUse Bash event. The payload is treated
// strictly as data.
func ExtractHookEvent(data []byte) (HookEvent, error) {
	malformed := errors.New("malformed hook input or missing tool_input.command")
	var payload struct {
		Cwd            string `json:"cwd"`
		PermissionMode string `json:"permission_mode"`
		ToolInput      struct {
			Command *string `json:"command"`
		} `json:"tool_input"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return HookEvent{}, malformed
	}
	if payload.ToolInput.Command == nil || *payload.ToolInput.Command == "" {
		return HookEvent{}, malformed
	}
	return HookEvent{Command: *payload.ToolInput.Command, WorkingDirectory: payload.Cwd, PermissionMode: payload.PermissionMode}, nil
}

// GuardDecision is the outcome of one PreToolUse evaluation. Ask continues
// the call through Claude's own confirmation prompt.
type GuardDecision struct {
	Exit   int
	Ask    bool
	Reason string
}

// Denied reports whether the decision stops the tool call.
func (d GuardDecision) Denied() bool { return d.Exit != GuardExitContinue }

func denyGuard(reason string) GuardDecision {
	return GuardDecision{Exit: GuardExitDeny, Reason: reason}
}

// EvaluateGuard decides one event: native deny patterns first, then the
// shared policy runtime.
func EvaluateGuard(event HookEvent, patterns []string, env GuardEnv) GuardDecision {
	if decision := evaluateDenyPatterns(event.Command, patterns, env.Home); decision.Denied() {
		return decision
	}
	toolEvent := agentharness.ToolEvent{Runner: "claude-code", Tool: "Bash", Shell: event.Command, WorkingDirectory: event.WorkingDirectory}
	if event.PermissionMode != "" {
		toolEvent.Context = map[string]string{"permission_mode": event.PermissionMode}
	}
	decision, err := env.Runtime.Evaluate(toolEvent)
	if err != nil {
		return denyGuard("the agent policy runtime could not evaluate the command: " + err.Error())
	}
	switch decision.Action {
	case agentharness.ActionDeny:
		return denyGuard(decision.Reason)
	case agentharness.ActionAsk:
		if interactivePermissionModes[event.PermissionMode] {
			return GuardDecision{Exit: GuardExitContinue, Ask: true, Reason: decision.Reason}
		}
		return denyGuard(decision.Reason + " (this session cannot show a confirmation prompt, so the operator must run it)")
	}
	return GuardDecision{Exit: GuardExitContinue}
}

func evaluateDenyPatterns(command string, patterns []string, home string) GuardDecision {
	for _, raw := range patterns {
		pattern := normalizeDenyPattern(raw, home)
		if pattern == "" || isRemovalDenyPattern(pattern) {
			continue
		}
		if matchShellGlob(pattern, command) {
			return denyGuard("native deny pattern=" + raw)
		}
	}
	return GuardDecision{Exit: GuardExitContinue}
}

// normalizeDenyPattern unwraps the Claude `Bash(...)` rule vocabulary and
// resolves home references so patterns compare against raw command text.
func normalizeDenyPattern(raw, home string) string {
	pattern := raw
	if strings.HasPrefix(pattern, "Bash(") && strings.HasSuffix(pattern, ")") {
		pattern = strings.TrimSuffix(strings.TrimPrefix(pattern, "Bash("), ")")
	}
	if home != "" {
		pattern = strings.ReplaceAll(pattern, "$HOME", home)
		pattern = strings.ReplaceAll(pattern, "~", home)
	}
	return pattern
}

// isRemovalDenyPattern reports whether a pattern targets a command family the
// removal engine decides by resolved path. Matching those textually would deny
// the declared-safe locations the engine exists to allow.
func isRemovalDenyPattern(pattern string) bool {
	for _, family := range []string{"rm ", "rmdir ", "unlink ", "shred ", "find ", "truncate "} {
		if strings.HasPrefix(pattern, family) || strings.HasPrefix(pattern, "sudo "+family) {
			return true
		}
	}
	return false
}

// matchShellGlob applies shell wildcard semantics, in which `*` spans any
// characters including separators. Path-oriented matching would under-match
// command text.
func matchShellGlob(pattern, value string) bool {
	patternRunes, valueRunes := []rune(pattern), []rune(value)
	patternIndex, valueIndex := 0, 0
	starPattern, starValue := -1, 0
	for valueIndex < len(valueRunes) {
		if patternIndex < len(patternRunes) {
			switch patternRunes[patternIndex] {
			case '*':
				starPattern, starValue = patternIndex, valueIndex
				patternIndex++
				continue
			case '?':
				patternIndex++
				valueIndex++
				continue
			case '[':
				if next, ok := matchGlobClass(patternRunes, patternIndex, valueRunes[valueIndex]); ok {
					patternIndex = next
					valueIndex++
					continue
				}
			default:
				if patternRunes[patternIndex] == valueRunes[valueIndex] {
					patternIndex++
					valueIndex++
					continue
				}
			}
		}
		if starPattern < 0 {
			return false
		}
		starValue++
		valueIndex = starValue
		patternIndex = starPattern + 1
	}
	for patternIndex < len(patternRunes) && patternRunes[patternIndex] == '*' {
		patternIndex++
	}
	return patternIndex == len(patternRunes)
}

// matchGlobClass evaluates the bracket expression opening at open against one
// character, returning the index just past the closing bracket.
func matchGlobClass(pattern []rune, open int, value rune) (int, bool) {
	index := open + 1
	negated := false
	if index < len(pattern) && (pattern[index] == '!' || pattern[index] == '^') {
		negated = true
		index++
	}
	matched, first := false, true
	for index < len(pattern) && (pattern[index] != ']' || first) {
		first = false
		if index+2 < len(pattern) && pattern[index+1] == '-' && pattern[index+2] != ']' {
			if value >= pattern[index] && value <= pattern[index+2] {
				matched = true
			}
			index += 3
			continue
		}
		if pattern[index] == value {
			matched = true
		}
		index++
	}
	if index >= len(pattern) {
		return 0, false
	}
	return index + 1, matched != negated
}

// RunHookGuard performs one PreToolUse decision end to end and returns the
// process exit code Claude Code reads. An ask is written to stdout as Claude's
// structured permission decision.
func RunHookGuard(input io.Reader, stdout, stderr io.Writer, patterns []string, env GuardEnv) int {
	data, err := io.ReadAll(input)
	if err != nil {
		return refuse(stderr, env, "", "unreadable hook input")
	}
	event, err := ExtractHookEvent(data)
	if err != nil {
		return refuse(stderr, env, "", err.Error())
	}
	env.appendLog(fmt.Sprintf("%s tool=Bash cwd=%q mode=%q cmd=%q patterns=%d", guardTimestamp(), event.WorkingDirectory, event.PermissionMode, env.excerpt(event.Command), len(patterns)))
	decision := EvaluateGuard(event, patterns, env)
	if decision.Denied() {
		return refuse(stderr, env, event.Command, decision.Reason)
	}
	if decision.Ask {
		env.appendLog(fmt.Sprintf("%s ASK cmd=%q reason=%q", guardTimestamp(), env.excerpt(event.Command), decision.Reason))
		if stdout != nil {
			_ = json.NewEncoder(stdout).Encode(map[string]any{"hookSpecificOutput": map[string]string{
				"hookEventName":            "PreToolUse",
				"permissionDecision":       "ask",
				"permissionDecisionReason": "vrooli agent policy: " + decision.Reason,
			}})
		}
	}
	return GuardExitContinue
}

func refuse(stderr io.Writer, env GuardEnv, command, reason string) int {
	env.appendLog(fmt.Sprintf("%s BLOCKED cmd=%q reason=%q", guardTimestamp(), env.excerpt(command), reason))
	if stderr != nil {
		fmt.Fprintf(stderr, "vrooli agent policy blocked this command: %s\n", reason)
	}
	return GuardExitDeny
}

func guardTimestamp() string { return time.Now().UTC().Format("2006-01-02T15:04:05Z") }

func (e GuardEnv) excerpt(command string) string {
	limit := e.LogCommandMax
	if limit <= 0 {
		limit = defaultGuardLogCommandMax
	}
	flattened := strings.NewReplacer("\n", " ", "\r", " ").Replace(command)
	runes := []rune(flattened)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return flattened
}

// appendLog records one decision. Audit logging is best effort: a hook that
// cannot write its log must still return a decision.
func (e GuardEnv) appendLog(line string) {
	if strings.TrimSpace(e.LogPath) == "" {
		return
	}
	// Owner-only: this directory holds an audit record of refused commands.
	if os.MkdirAll(filepath.Dir(e.LogPath), 0o700) != nil {
		return
	}
	limit := e.LogMaxBytes
	if limit <= 0 {
		limit = defaultGuardLogMaxBytes
	}
	if info, err := os.Stat(e.LogPath); err == nil && info.Size() > limit {
		_ = os.Rename(e.LogPath, e.LogPath+".1")
	}
	file, err := os.OpenFile(e.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = io.WriteString(file, line+"\n")
}
