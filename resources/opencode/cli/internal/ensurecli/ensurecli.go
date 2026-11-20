package ensurecli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	opencodeversion "github.com/vrooli/vrooli/packages/opencode-version"
	"github.com/vrooli/vrooli/resources/opencode/cli/internal/configcli"
	"github.com/vrooli/vrooli/resources/opencode/cli/internal/permissions"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/vrooli/resources/internal/agentapp"
)

type Handlers struct {
	Adapter *permissions.Adapter
	Config  *configcli.Handlers
	Stdout  io.Writer
	Stderr  io.Writer
}

func Default() *Handlers {
	a, err := permissions.DefaultAdapter()
	if err != nil {
		a = &permissions.Adapter{}
	}
	return &Handlers{
		Adapter: a,
		Config:  configcli.Default(),
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}
}

// Step is OpenCode's contribution to the shared `ensure` command: the policy
// hook, the permission block and the config file.
func Step(h *Handlers) agentapp.EnsureStep {
	if h == nil {
		h = Default()
	}
	return agentapp.EnsureStep{
		Name: "hooks-and-permissions",
		Run: func(stdout, stderr io.Writer) error {
			h.Stdout, h.Stderr = stdout, stderr
			return h.Ensure(nil)
		},
	}
}

func warnIfOpenCodeV2(stderr io.Writer) {
	path, err := exec.LookPath("opencode")
	if err != nil {
		return
	}
	out, err := exec.Command(path, "--version").Output()
	if err != nil {
		return
	}
	if major, ok := opencodeversion.ParseMajor(string(out)); ok && major >= 2 {
		fmt.Fprintf(stderr, "info: OpenCode %d detected; native permissions and the V%d policy plugin are projected. Runtime hook firing remains unverified until a live canary.\n", major, major)
	}
}

func (h *Handlers) Ensure(args []string) error {
	warnIfOpenCodeV2(h.Stderr)
	if h.Config != nil {
		if err := h.Config.Ensure(nil); err != nil {
			fmt.Fprintf(h.Stderr, "warn: opencode config: %v\n", err)
		}
	}
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
	// A clean permission map can still need a plugin change after a major
	// upgrade (V1 module -> V2 module). Reconcile the projection every time.
	if err := h.Adapter.SyncPlugin(p); err != nil {
		return fmt.Errorf("sync policy plugin: %w", err)
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
