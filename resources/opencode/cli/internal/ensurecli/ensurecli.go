package ensurecli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/vrooli/vrooli/resources/opencode/cli/internal/permissions"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/cliapp"
)

type Handlers struct {
	Adapter       *permissions.Adapter
	Stdout        io.Writer
	Stderr        io.Writer
}

func Default() *Handlers {
	a, err := permissions.DefaultAdapter()
	if err != nil {
		a = &permissions.Adapter{}
	}
	return &Handlers{
		Adapter: a,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}
}

func Commands(h *Handlers) cliapp.Command {
	if h == nil {
		h = Default()
	}
	return cliapp.Command{
		Name:        "ensure",
		Description: "Self-heal OpenCode hooks and permissions — installs the Vrooli policy hook and reconciles the permission block",
		Run:         h.Ensure,
	}
}

func (h *Handlers) Ensure(args []string) error {
	if h.Adapter == nil {
		return errors.New("OpenCode settings adapter is unavailable")
	}

	broker := agentharness.NewHookBroker()
	target := agentharness.HookTarget{
		Agent: "opencode",
		Path:  h.Adapter.PluginPath + ".hooks.json",
	}

	if _, err := broker.Migrate(target); err != nil {
		fmt.Fprintf(h.Stderr, "warn: hook migration: %v\n", err)
	}

	hook := map[string]any{
		"command": "vrooli-policy-runner",
		"args":    []string{"hook", "--runner", "opencode"},
	}
	result, err := broker.Reconcile(target, agentharness.HookRegistration{
		Event: "tool.execute.before",
		ID:    "vrooli-policy-runner",
		Hook:  hook,
	})
	if err != nil {
		return fmt.Errorf("reconcile hooks: %w", err)
	}
	fmt.Fprintf(h.Stdout, "hooks reconcile: %s (%s)\n", result.Status, result.Reason)

	p, err := h.Adapter.Load()
	if err != nil {
		return fmt.Errorf("load permissions: %w", err)
	}

	fp := permissions.Fingerprint(p)
	state, err := h.Adapter.LoadState()
	if err != nil {
		return fmt.Errorf("load permission state: %w", err)
	}

	if state != nil && state.Fingerprint == fp {
		fmt.Fprintln(h.Stdout, "permissions: clean — no drift detected")
	} else {
		if err := h.Adapter.Save(p, "ensure"); err != nil {
			return fmt.Errorf("save permissions: %w", err)
		}
		fmt.Fprintln(h.Stdout, "permissions: reconciled")
	}

	data := struct {
		Status string `json:"status"`
		Hooks  string `json:"hooks"`
		Perms  string `json:"permissions"`
	}{
		Status: "ok",
		Hooks:  result.Status,
		Perms:  "current",
	}
	if state == nil || state.Fingerprint != fp {
		data.Perms = "reconciled"
	}
	out, _ := json.Marshal(data)
	fmt.Fprintf(h.Stdout, "%s\n", out)
	return nil
}