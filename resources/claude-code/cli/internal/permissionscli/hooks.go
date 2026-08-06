package permissionscli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"resource-claude-code/cli/internal/permissions"

	"github.com/vrooli/cli-core/cliapp"
)

// HookCommands exposes project and global hook reconciliation to the resource
// CLI. It intentionally owns only hooks identified by the caller.
func HookCommands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default("", "")
	}
	return cliapp.SubcommandGroup{
		Name:        "hooks",
		Description: "Reconcile Claude Code settings hooks without sourcing resource shell code",
		Subcommands: []cliapp.Command{
			{Name: "reconcile", Description: "Add or update an identified hook (JSON)", Run: h.ReconcileHook},
			{Name: "remove", Description: "Remove an identified hook", Run: h.RemoveHook},
		},
	}
}

func (h *Handlers) ReconcileHook(args []string) error {
	fs := h.flagSet("hooks reconcile")
	event := fs.String("event", "", "Claude hook event name")
	identifier := fs.String("id", "", "Stable hook identifier")
	hookJSON := fs.String("hook-json", "", "Hook object as JSON")
	scope := fs.String("scope", "project", "Settings scope: project, project-local, or global")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *event == "" || *identifier == "" || *hookJSON == "" {
		return errors.New("--event, --id, and --hook-json are required")
	}
	var hook map[string]any
	if err := json.Unmarshal([]byte(*hookJSON), &hook); err != nil {
		return fmt.Errorf("invalid --hook-json: %w", err)
	}
	adapter, err := h.adapterForScope(*scope)
	if err != nil {
		return err
	}
	result, err := adapter.ReconcileHook(*event, *identifier, hook)
	result.Scope = *scope
	result.Settings = adapter.SettingsPath
	if writeErr := writeHookResult(h.Stdout, result); writeErr != nil {
		return writeErr
	}
	return err
}

func (h *Handlers) RemoveHook(args []string) error {
	fs := h.flagSet("hooks remove")
	event := fs.String("event", "", "Claude hook event name")
	identifier := fs.String("id", "", "Stable hook identifier")
	scope := fs.String("scope", "project", "Settings scope: project, project-local, or global")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *event == "" || *identifier == "" {
		return errors.New("--event and --id are required")
	}
	adapter, err := h.adapterForScope(*scope)
	if err != nil {
		return err
	}
	result, err := adapter.RemoveHook(*event, *identifier)
	result.Scope = *scope
	result.Settings = adapter.SettingsPath
	if writeErr := writeHookResult(h.Stdout, result); writeErr != nil {
		return writeErr
	}
	return err
}

func writeHookResult(out interface{ Write([]byte) (int, error) }, result permissions.HookResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "%s\n", data)
	return err
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
		SettingsPath:  filepath.Join(root, ".claude", settingsName),
		HookScriptDir: filepath.Join(root, ".claude", ".vrooli-hooks"),
	}, nil
}
