// Package permissionscli registers the `permissions` subcommand group on
// resource-opencode. Native JSON projection and migration stay local while
// common command behavior is shared through agentharness.
package permissionscli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/vrooli/vrooli/resources/opencode/cli/internal/permissions"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type Handlers struct {
	Adapter        *permissions.Adapter
	DetectCaller   func() cliutil.CallerKind
	Policy         agentharness.Policy
	CLIVersion     string
	PinnedVersion  string
	VersionCommand []string
	VersionRunner  func(context.Context, []string) (string, error)
	Stdin          io.Reader
	Stdout         io.Writer
	Stderr         io.Writer
}

func Default(cliVersion, pinnedVersion string) *Handlers {
	a, err := permissions.DefaultAdapter()
	if err != nil {
		a = &permissions.Adapter{}
	}
	return &Handlers{
		Adapter: a, DetectCaller: cliutil.DetectCallerKind, Policy: agentharness.DefaultPolicy(),
		CLIVersion: cliVersion, PinnedVersion: pinnedVersion,
		VersionCommand: []string{"opencode", "--version"}, VersionRunner: defaultVersionRunner,
		Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr,
	}
}

func defaultVersionRunner(ctx context.Context, args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("empty version command")
	}
	out, err := exec.CommandContext(ctx, args[0], args[1:]...).Output()
	return strings.TrimSpace(string(out)), err
}

func (h *Handlers) shared() *agentharness.PermissionHandlers {
	return agentharness.NewPermissionHandlers(agentharness.PermissionCommandConfig{
		AdapterFor: func(string) (agentharness.PermissionAdapter, error) {
			if h.Adapter == nil {
				return nil, errors.New("OpenCode settings adapter is unavailable")
			}
			return bridge(h.Adapter, h.CLIVersion), nil
		},
		Description: "View and manage OpenCode permissions (~/.config/opencode/opencode.json permission.bash)",
		CommandDescriptions: map[string]string{
			"show":        "Print the full opencode.json (raw or pretty)",
			"deny":        "Add a bash deny pattern (mutating)",
			"allow":       "Add a bash allow pattern (mutating)",
			"ask":         "Add a bash ask pattern (mutating)",
			"drift-check": "Compare current opencode.json fingerprint to last Vrooli write",
			"doctor":      "Check installed opencode version against the pinned upstream version",
		},
		ResourceLabel: "resource-opencode", ConfigHint: "~/.config/opencode/opencode.json permission.bash",
		VersionLabel: "resource-opencode", CLIVersion: h.CLIVersion, PinnedVersion: h.PinnedVersion,
		VersionCommand: h.VersionCommand, VersionRunner: h.VersionRunner, DetectCaller: h.DetectCaller,
		Policy: h.Policy, ExclusivePatterns: true, Stdout: h.Stdout, Stderr: h.Stderr,
	})
}

func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default("", "")
	}
	return h.shared().Commands([]cliapp.Command{
		{Name: "plan", Description: "Plan a whole declared portable permission document (JSON)", Run: h.Plan},
		{Name: "reconcile", Description: "Reconcile a whole declared portable permission document (mutating, JSON)", Run: h.Reconcile},
		{Name: "migrate", Description: "Heal a pre-1.0 opencode.json: move the retired inline managed-key into the sidecar and strip it (idempotent)", Run: h.Migrate},
	})
}

func (h *Handlers) List(args []string) error       { return h.shared().List(args) }
func (h *Handlers) Show(args []string) error       { return h.shared().Show(args) }
func (h *Handlers) DriftCheck(args []string) error { return h.shared().DriftCheck(args) }
func (h *Handlers) Doctor(args []string) error     { return h.shared().Doctor(args) }
func (h *Handlers) Deny(args []string) error       { return h.shared().Deny(args) }
func (h *Handlers) Allow(args []string) error      { return h.shared().Allow(args) }
func (h *Handlers) Ask(args []string) error        { return h.shared().Ask(args) }
func (h *Handlers) Remove(args []string) error     { return h.shared().Remove(args) }
func (h *Handlers) Reset(args []string) error      { return h.shared().Reset(args) }

// Migrate heals a pre-1.0 opencode.json that still carries the retired inline
// managed-pattern key. The native adapter reads that key as a one-time
// fallback, then Save strips it and records ownership in the shared sidecar.
func (h *Handlers) Migrate(args []string) error {
	if err := h.flagSet("permissions migrate").Parse(args); err != nil {
		return err
	}
	if h.Adapter == nil {
		return errors.New("OpenCode settings adapter is unavailable")
	}
	p, err := h.Adapter.Load()
	if err != nil {
		return err
	}
	if err := h.Adapter.Save(p, h.CLIVersion); err != nil {
		return err
	}
	_, err = fmt.Fprintf(h.Stdout, "migrated managed-permissions list into the sidecar; %s now carries only schema-valid keys\n", h.Adapter.SettingsPath)
	return err
}

func (h *Handlers) flagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	return fs
}

func (h *Handlers) gate(verb string, mutating, authorized bool) error {
	return h.shared().Gate(verb, mutating, authorized)
}

func HookCommands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default("", "")
	}
	return agentharness.HookCommands(agentharness.HookCommandConfig{
		Agent: "opencode", Description: "Reconcile OpenCode hooks through the shared Vrooli hook broker",
		ScopeDefault: "user",
		ScopeHelp:    "Hook scope: user (~/.config/opencode/plugins)",
		Target: func(scope string) (agentharness.HookTarget, error) {
			if strings.TrimSpace(scope) != "" && strings.TrimSpace(scope) != "user" {
				return agentharness.HookTarget{}, fmt.Errorf("unknown --scope %q (want user)", scope)
			}
			if h.Adapter == nil {
				return agentharness.HookTarget{}, errors.New("OpenCode settings adapter is unavailable")
			}
			return agentharness.HookTarget{Agent: "opencode", Path: h.Adapter.PluginPath + ".hooks.json"}, nil
		},
		Stdout: h.Stdout, Stderr: h.Stderr,
	})
}

func bridge(a *permissions.Adapter, version string) agentharness.PermissionAdapter {
	return agentharness.PermissionAdapterFuncs{
		LoadFunc: func() (agentharness.PermissionPolicy, error) {
			p, err := a.Load()
			return agentharness.PermissionPolicy{BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow, SettingsPath: a.SettingsPath}, err
		},
		SaveFunc: func(p agentharness.PermissionPolicy) error {
			return a.Save(permissions.Policy{BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow}, version)
		},
		SettingsPathFunc: func() string { return a.SettingsPath },
		FingerprintFunc: func(p agentharness.PermissionPolicy) string {
			return permissions.Fingerprint(permissions.Policy{BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow})
		},
		LoadStateFunc: func() (*agentharness.PermissionState, error) { return a.LoadState() },
		WriteStateFunc: func(p agentharness.PermissionPolicy, writtenByVersion string) error {
			return a.WriteState(permissions.Policy{BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow}, writtenByVersion)
		},
	}
}
