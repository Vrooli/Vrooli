// Package effort keeps long Claude Code efforts anchored across context
// compaction. A compacted summary can drop a goal's objective, the operator's
// stance and its stop rules (docs/agent-system/LONG_EFFORT_DESIGN.md §1); the
// SessionStart hook with the `compact` matcher adds a reminder to re-read the
// active charter before acting. Claude Code adds a SessionStart hook's stdout
// to the model's context.
package effort

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/resources/internal/agentapp"
)

// HookID identifies the managed SessionStart entry.
const HookID = "vrooli-compact-reread"

// DiscoveryHookID identifies the task-time program discovery reminder.
const DiscoveryHookID = "vrooli-program-discovery"

// CompactContext is what the model sees right after a compaction.
const CompactContext = "Context was just compacted. If you are working under a goal, plan, charter or delegated brief, " +
	"re-read it from its source file now, before your next action: the compacted summary may have dropped its objective, " +
	"the operator's stance (quoted verbatim there) and its stop and closure rules. Then continue at the size it names; " +
	"do not narrow the work because the summary is shorter."

// DiscoveryContext prompts a fresh Claude session to find a task-specific owner skill
// and governed program before repeating a multi-step CLI workflow.
const DiscoveryContext = "For a recurring or multi-step scenario/CLI task, discover the owner skill and matching " +
	"governed program at task time: run `search-hub query \"<task>\"`, read the matching skill, and check `program-runtime " +
	"library search \"<task>\"`. Call a contract only when its declared inputs and effects cover the task; do not count a " +
	"partial match as full coverage."

// Commands is the `effort` group; `compact-context` is the hook command.
func Commands(stdout io.Writer) cliapp.SubcommandGroup {
	if stdout == nil {
		stdout = os.Stdout
	}
	return cliapp.SubcommandGroup{
		Name:        "effort",
		Description: "Keep long efforts anchored across context compaction",
		Subcommands: []cliapp.Command{{
			Name:         "compact-context",
			Description:  "Print the post-compaction re-read reminder (SessionStart hook command)",
			LatencyBound: true,
			Run: func([]string) error {
				_, err := fmt.Fprintln(stdout, CompactContext)
				return err
			},
		}, {
			Name:         "discover-programs",
			Description:  "Print the task-time program discovery reminder (SessionStart hook command)",
			LatencyBound: true,
			Run: func([]string) error {
				_, err := fmt.Fprintln(stdout, DiscoveryContext)
				return err
			},
		}},
	}
}

// Registration is the managed hook. executable is the resource CLI that
// wrote it, so the hook runs the same implementation with no PATH lookup.
func Registration(executable string) agentharness.HookRegistration {
	return agentharness.HookRegistration{
		Event:   "SessionStart",
		ID:      HookID,
		Matcher: "compact",
		Hook:    map[string]any{"type": "command", "command": shellQuote(executable) + " effort compact-context"},
	}
}

// DiscoveryRegistration adds a short task-time lookup reminder to fresh sessions.
func DiscoveryRegistration(executable string) agentharness.HookRegistration {
	return agentharness.HookRegistration{
		Event:   "SessionStart",
		ID:      DiscoveryHookID,
		Matcher: "startup",
		Hook:    map[string]any{"type": "command", "command": shellQuote(executable) + " effort discover-programs"},
	}
}

// EnsureStep reconciles both SessionStart hooks into the user's global settings file.
func EnsureStep(settingsPath func() string, executable func() string) agentapp.EnsureStep {
	return agentapp.EnsureStep{
		Name: "session-start-hooks",
		Run: func(stdout, _ io.Writer) error {
			path := settingsPath()
			if path == "" {
				return fmt.Errorf("claude settings path is unavailable")
			}
			broker := agentharness.NewHookBroker()
			target := agentharness.HookTarget{Agent: "claude-code", Path: path}
			result, err := broker.Reconcile(target, Registration(executable()))
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout, "compact re-read hook: %s (%s)\n", result.Status, path)
			discovery, err := broker.Reconcile(target, DiscoveryRegistration(executable()))
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout, "program discovery hook: %s (%s)\n", discovery.Status, path)
			return nil
		},
	}
}

// Executable resolves the running binary, following symlinks.
func Executable() string {
	executable, err := os.Executable()
	if err != nil {
		return "resource-claude-code"
	}
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		return resolved
	}
	return executable
}

func shellQuote(value string) string {
	if value != "" && strings.IndexFunc(value, func(r rune) bool {
		return !(r == '/' || r == '.' || r == '-' || r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z')
	}) < 0 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
