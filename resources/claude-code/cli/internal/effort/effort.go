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

// CompactContext is what the model sees right after a compaction.
const CompactContext = "Context was just compacted. If you are working under a goal, plan, charter or delegated brief, " +
	"re-read it from its source file now, before your next action: the compacted summary may have dropped its objective, " +
	"the operator's stance (quoted verbatim there) and its stop and closure rules. Then continue at the size it names; " +
	"do not narrow the work because the summary is shorter."

// Commands is the `effort` group; `compact-context` is the hook command.
func Commands(stdout io.Writer) cliapp.SubcommandGroup {
	if stdout == nil {
		stdout = os.Stdout
	}
	return cliapp.SubcommandGroup{
		Name:        "effort",
		Description: "Keep long efforts anchored across context compaction",
		Subcommands: []cliapp.Command{{
			Name:        "compact-context",
			Description: "Print the post-compaction re-read reminder (SessionStart hook command)",
			Run: func([]string) error {
				_, err := fmt.Fprintln(stdout, CompactContext)
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

// EnsureStep reconciles the hook into the user's global settings file.
func EnsureStep(settingsPath func() string, executable func() string) agentapp.EnsureStep {
	return agentapp.EnsureStep{
		Name: "compact-reread-hook",
		Run: func(stdout, _ io.Writer) error {
			path := settingsPath()
			if path == "" {
				return fmt.Errorf("claude settings path is unavailable")
			}
			result, err := agentharness.NewHookBroker().Reconcile(agentharness.HookTarget{Agent: "claude-code", Path: path}, Registration(executable()))
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout, "compact re-read hook: %s (%s)\n", result.Status, path)
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
