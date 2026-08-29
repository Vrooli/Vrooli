// Package permissionscli registers the `permissions` subcommand group on
// resource-grok. The native TOML adapter and hook projection remain local;
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

	"github.com/vrooli/vrooli/resources/grok/cli/internal/permissions"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type Handlers struct {
	AdapterFor     func(scope permissions.Scope) (*permissions.Adapter, error)
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
	return &Handlers{
		AdapterFor: permissions.DefaultAdapter, DetectCaller: cliutil.DetectCallerKind,
		Policy: agentharness.DefaultPolicy(), CLIVersion: cliVersion, PinnedVersion: pinnedVersion,
		VersionCommand: []string{"grok", "--version"}, VersionRunner: defaultVersionRunner,
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

func (h *Handlers) adapter(scopeRaw string) (*permissions.Adapter, error) {
	scope := permissions.Scope(strings.ToLower(strings.TrimSpace(scopeRaw)))
	switch scope {
	case permissions.ScopeUser, permissions.ScopeAdmin, "":
	default:
		return nil, fmt.Errorf("unknown --scope %q (want user|admin)", scopeRaw)
	}
	a, err := h.AdapterFor(scope)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, errors.New("adapter resolution returned nil")
	}
	return a, nil
}

func (h *Handlers) shared() *agentharness.PermissionHandlers {
	return agentharness.NewPermissionHandlers(agentharness.PermissionCommandConfig{
		AdapterFor: func(scope string) (agentharness.PermissionAdapter, error) {
			a, err := h.adapter(scope)
			if err != nil {
				return nil, err
			}
			return bridge(a), nil
		},
		ScopeDefault: string(permissions.ScopeUser),
		ScopeHelp:    "Config scope: user (~/.grok/config.toml) or admin (~/.grok/requirements.toml)",
		Description:  "View and manage Grok permissions (~/.grok/config.toml [permission]) and the paired PreToolUse deny hook",
		CommandDescriptions: map[string]string{
			"show":        "Print the full config file (raw or pretty)",
			"deny":        "Add a deny pattern, e.g. 'Bash(rm -rf *)' (mutating)",
			"allow":       "Add an allow pattern, e.g. 'Bash(git *)' (mutating)",
			"ask":         "Add an ask pattern (mutating)",
			"drift-check": "Compare current config fingerprint to last Vrooli write",
			"doctor":      "Check installed grok version and confirm enforcement wiring",
		},
		ResourceLabel: "resource-grok", ConfigHint: "~/.grok/config.toml [permission]",
		VersionLabel: "resource-grok", CLIVersion: h.CLIVersion, PinnedVersion: h.PinnedVersion,
		VersionCommand: h.VersionCommand, VersionRunner: h.VersionRunner, DetectCaller: h.DetectCaller,
		Policy: h.Policy, ResetPolicy: agentharness.PermissionPolicy{Hooks: true}, Stdout: h.Stdout, Stderr: h.Stderr,
		ExclusivePatterns: true,
		ListSuffix: func(stdout io.Writer, shared agentharness.PermissionAdapter) {
			a, err := h.adapter(shared.Scope())
			if err == nil {
				_, _ = fmt.Fprintf(stdout, "deny-hook: %s\n", a.HookConfigPath())
			}
		},
		ShowFields: func(fields map[string]any, shared agentharness.PermissionAdapter, _ agentharness.PermissionPolicy) {
			a, err := h.adapter(shared.Scope())
			if err == nil {
				fields["denyHook"] = a.HookConfigPath()
			}
		},
		DoctorExtra: func(stdout, _ io.Writer, shared agentharness.PermissionAdapter) error {
			a, err := h.adapter(shared.Scope())
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(stdout, "enforcement: Grok natively honours [permission] deny/ask/allow (deny > ask > allow),")
			_, _ = fmt.Fprintln(stdout, "             and a portable PreToolUse policy-runner command is configured when requested.")
			if info, statErr := os.Stat(a.HookConfigPath()); statErr == nil {
				_, _ = fmt.Fprintf(stdout, "deny-hook: installed at %s (%d bytes)\n", a.HookConfigPath(), info.Size())
			} else {
				_, _ = fmt.Fprintln(stdout, "deny-hook: not installed (no Bash deny patterns yet)")
			}
			if runner := strings.TrimSpace(os.Getenv("VROOLI_AGENT_POLICY_RUNNER")); runner != "" {
				_, _ = fmt.Fprintf(stdout, "policy runner: %s (run the installed-version canary before enforcement)\n", runner)
			} else {
				_, _ = fmt.Fprintln(stdout, "policy runner: vrooli-policy-runner (run the installed-version canary before enforcement)")
			}
			return nil
		},
	})
}

func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default("", "")
	}
	return h.shared().Commands([]cliapp.Command{
		{Name: "plan", Description: "Plan a whole declared portable permission document (JSON)", Run: h.Plan},
		{Name: "reconcile", Description: "Reconcile a whole declared portable permission document (mutating, JSON)", Run: h.Reconcile},
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

func (h *Handlers) flagSet(name string) (*flag.FlagSet, *string) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	scope := fs.String("scope", string(permissions.ScopeUser), "Config scope: user (~/.grok/config.toml) or admin (~/.grok/requirements.toml)")
	return fs, scope
}

func (h *Handlers) gate(verb string, mutating, authorized bool) error {
	return h.shared().Gate(verb, mutating, authorized)
}

func HookCommands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default("", "")
	}
	return agentharness.HookCommands(agentharness.HookCommandConfig{
		Agent: "grok", Description: "Reconcile Grok hooks through the shared Vrooli hook broker",
		ScopeDefault: string(permissions.ScopeUser),
		ScopeHelp:    "Hook scope: user (~/.grok/hooks) or admin (~/.grok/hooks)",
		Target: func(scope string) (agentharness.HookTarget, error) {
			a, err := h.adapter(scope)
			if err != nil {
				return agentharness.HookTarget{}, err
			}
			return agentharness.HookTarget{Agent: "grok", Path: a.HookConfigPath()}, nil
		},
		Stdout: h.Stdout, Stderr: h.Stderr,
	})
}

func bridge(a *permissions.Adapter) agentharness.PermissionAdapter {
	return agentharness.PermissionAdapterFuncs{
		LoadFunc: func() (agentharness.PermissionPolicy, error) {
			p, err := a.Load()
			return agentharness.PermissionPolicy{BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow, Hooks: p.Hooks, SettingsPath: a.SettingsPath}, err
		},
		SaveFunc: func(p agentharness.PermissionPolicy) error {
			return a.Save(permissions.Policy{BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow, Hooks: p.Hooks})
		},
		SettingsPathFunc: func() string { return a.SettingsPath },
		ScopeFunc:        func() string { return string(a.Scope) },
		FingerprintFunc: func(p agentharness.PermissionPolicy) string {
			return permissions.Fingerprint(permissions.Policy{BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow, Hooks: p.Hooks})
		},
		LoadStateFunc: func() (*agentharness.PermissionState, error) { return a.LoadState() },
		WriteStateFunc: func(p agentharness.PermissionPolicy, version string) error {
			return a.WriteState(permissions.Policy{BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow, Hooks: p.Hooks}, version)
		},
	}
}
