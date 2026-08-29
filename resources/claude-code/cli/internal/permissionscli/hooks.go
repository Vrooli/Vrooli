package permissionscli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/resources/claude-code/cli/internal/permissions"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/cliapp"
)

// HookCommands exposes the shared broker contract while retaining Claude's
// project, project-local, and global settings resolution.
func HookCommands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default("", "")
	}
	return agentharness.HookCommands(agentharness.HookCommandConfig{
		Agent:        "claude-code",
		Description:  "Reconcile Claude Code settings hooks through the shared Vrooli hook broker",
		ScopeDefault: "project",
		ScopeHelp:    "Settings scope: project, project-local, or global",
		Target: func(scope string) (agentharness.HookTarget, error) {
			adapter, err := h.adapterForScope(scope)
			if err != nil {
				return agentharness.HookTarget{}, err
			}
			return agentharness.HookTarget{Agent: "claude-code", Path: adapter.SettingsPath}, nil
		},
		Stdout: h.Stdout,
		Stderr: h.Stderr,
	})
}

func (h *Handlers) adapterForScope(scope string) (*permissions.Adapter, error) {
	if scope == "global" {
		if h.Adapter == nil {
			return nil, errors.New("Claude settings adapter is unavailable")
		}
		return h.Adapter, nil
	}
	root := strings.TrimSpace(os.Getenv("CLAUDE_PROJECT_ROOT"))
	if root == "" {
		cmd := exec.Command("git", "rev-parse", "--show-toplevel")
		if output, err := cmd.Output(); err == nil {
			root = strings.TrimSpace(string(output))
		}
	}
	if root == "" {
		root, _ = os.Getwd()
	}
	settingsName := "settings.json"
	if scope == "project-local" {
		settingsName = "settings.local.json"
	} else if scope != "project" {
		return nil, fmt.Errorf("invalid scope %q (use project, project-local, or global)", scope)
	}
	return &permissions.Adapter{
		SettingsPath: filepath.Join(root, ".claude", settingsName),
		HookStateDir: filepath.Join(root, ".claude", permissions.HookStateDirName),
	}, nil
}
