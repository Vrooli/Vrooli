// Package permissionscli registers the `permissions` subcommand group on
// resource-codex. Native TOML scope handling remains in the resource adapter;
// common command behavior is provided by agentharness.
package permissionscli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/vrooli/resources/codex/cli/internal/permissions"
	"github.com/vrooli/vrooli/resources/internal/agentapp"

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
	// ManagedHooksPath overrides resources/codex/managed-hooks.json (tests).
	ManagedHooksPath string
	// TrustProbe asks Codex whether it trusts the installed hooks; the default
	// asks `codex app-server`.
	TrustProbe permissions.TrustProbe
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
}

func Default(cliVersion, pinnedVersion string) *Handlers {
	return &Handlers{
		AdapterFor: permissions.DefaultAdapter, DetectCaller: cliutil.DetectCallerKind,
		Policy: agentharness.DefaultPolicy(), CLIVersion: cliVersion, PinnedVersion: pinnedVersion,
		VersionCommand: []string{"codex", "--version"},
		Stdin:          os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr,
	}
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
		ScopeHelp:    "Config scope: user (~/.codex/config.toml) or admin (system requirements.toml)",
		Description:  "View and manage Codex permission intent ($CODEX_HOME/.vrooli-permissions-intent.json) and its native projections",
		CommandDescriptions: map[string]string{
			"show":        "Print the managed intent, or the native config file with --raw",
			"deny":        "Add a bash deny pattern (mutating)",
			"allow":       "Add a bash allow pattern (mutating)",
			"ask":         "Add a bash ask pattern (mutating)",
			"reset":       "Clear Vrooli-managed Bash rules; retain native execution profile",
			"drift-check": "Compare current config fingerprint to last Vrooli write",
			"doctor":      "Check installed codex version and explain enforcement caveats",
		},
		ResourceLabel: "resource-codex", ConfigHint: "$CODEX_HOME/.vrooli-permissions-intent.json",
		VersionLabel: "resource-codex", CLIVersion: h.CLIVersion, PinnedVersion: h.PinnedVersion,
		VersionCommand: h.VersionCommand, VersionRunner: h.VersionRunner, DetectCaller: h.DetectCaller,
		Policy: h.Policy, Stdout: h.Stdout, Stderr: h.Stderr,
		ExclusivePatterns: true,
		ListSuffix: func(stdout io.Writer, shared agentharness.PermissionAdapter) {
			if a, err := h.adapter(shared.Scope()); err == nil {
				fmt.Fprintf(stdout, "intent: %s\n", a.IntentPath())
			}
		},
		ShowFields: func(fields map[string]any, shared agentharness.PermissionAdapter, _ agentharness.PermissionPolicy) {
			if a, err := h.adapter(shared.Scope()); err == nil {
				fields["intentPath"] = a.IntentPath()
			}
		},
		NativeDrift: func(shared agentharness.PermissionAdapter) error {
			a, err := h.adapter(shared.Scope())
			if err != nil {
				return err
			}
			p, err := a.Load()
			if err != nil {
				return err
			}
			if p.HookFingerprint != a.DesiredHookFingerprint(p) {
				return errors.New("managed PreToolUse hook registration differs from the recorded intent")
			}
			if p.NativeFingerprint != permissions.DesiredExecutionFingerprint(p.Execution, p.ExecutionActive) {
				return errors.New("native execution projection differs from the recorded intent")
			}
			return h.managedHookDrift(a)
		},
		DoctorExtra: func(stdout, _ io.Writer, shared agentharness.PermissionAdapter) error {
			a, err := h.adapter(shared.Scope())
			if err != nil {
				return err
			}
			p, err := a.Load()
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(stdout, "note: Codex does NOT enforce Vrooli's per-command-pattern deny/allow/ask rules natively.")
			fmt.Fprintf(stdout, "      They are recorded in %s and project to a PreToolUse hook;\n", a.IntentPath())
			_, _ = fmt.Fprintln(stdout, "      native filesystem/network/approval settings are managed by v2 execution documents.")
			if p.LegacyTable {
				fmt.Fprintf(stdout, "legacy: %s still carries the retired [vrooli] table; run `resource-codex ensure` to move it\n", a.SettingsPath)
			}
			fmt.Fprintf(stdout, "execution: configured=%t active_intent=%t effective_runtime=unverified\n", p.Execution != nil, p.ExecutionActive)
			if p.Execution != nil {
				fmt.Fprintf(stdout, "native_projection_matches=%t\n", p.NativeFingerprint == permissions.DesiredExecutionFingerprint(p.Execution, p.ExecutionActive))
			}
			fmt.Fprintf(stdout, "hook_registration_matches=%t\n", p.HookFingerprint == a.DesiredHookFingerprint(p))
			if a.HookPath != "" {
				_ = h.reconcileManagedHooks(a, false, stdout)
			}
			fmt.Fprintln(stdout, "Higher-precedence config, launch flags and managed requirements may override this file; verify a fresh desktop session separately.")
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

func (h *Handlers) flagSet(name string) (*flag.FlagSet, *string) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	scope := fs.String("scope", string(permissions.ScopeUser), "Config scope: user (~/.codex/config.toml) or admin (system requirements.toml)")
	return fs, scope
}

func (h *Handlers) gate(verb string, mutating, authorized bool) error {
	return h.shared().Gate(verb, mutating, authorized)
}

func HookCommands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default("", "")
	}
	group := agentharness.HookCommands(agentharness.HookCommandConfig{
		Agent: "codex", Description: "Reconcile Codex hooks through the shared Vrooli hook broker",
		ScopeDefault: string(permissions.ScopeUser),
		ScopeHelp:    "Hook scope: user (~/.codex/hooks.json) or admin (unsupported)",
		Target: func(scope string) (agentharness.HookTarget, error) {
			a, err := h.adapter(scope)
			if err != nil {
				return agentharness.HookTarget{}, err
			}
			return agentharness.HookTarget{Agent: "codex", Path: a.HookPath}, nil
		},
		Stdout: h.Stdout, Stderr: h.Stderr,
	})
	group.Subcommands = append(group.Subcommands, h.managedHookCommands()...)
	return group
}

// IntentMigrationStep is the `ensure` step that moves a retired `[vrooli]`
// table out of config.toml, where Codex warns about it on every launch, into
// the intent sidecar. It is silent when there is nothing to move.
func IntentMigrationStep() agentapp.EnsureStep {
	return agentapp.EnsureStep{Name: "permission-intent", Run: func(stdout, _ io.Writer) error {
		a, err := permissions.DefaultAdapter(permissions.ScopeUser)
		if err != nil {
			return err
		}
		report, err := a.MigrateLegacyTable()
		if err != nil {
			return err
		}
		if report != "" {
			fmt.Fprintf(stdout, "codex permissions: %s\n", report)
		}
		return nil
	}}
}

func bridge(a *permissions.Adapter) agentharness.PermissionAdapter {
	return agentharness.PermissionAdapterFuncs{
		LoadFunc: func() (agentharness.PermissionPolicy, error) {
			p, err := a.Load()
			return agentharness.PermissionPolicy{BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow, NativeFingerprint: p.NativeFingerprint, HookFingerprint: p.HookFingerprint, SnapshotDigest: p.SnapshotDigest, SettingsPath: a.SettingsPath}, err
		},
		SaveFunc: func(p agentharness.PermissionPolicy) error {
			return a.Save(permissions.Policy{BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow, NativeFingerprint: p.NativeFingerprint, HookFingerprint: p.HookFingerprint, SnapshotDigest: p.SnapshotDigest})
		},
		SettingsPathFunc: func() string { return a.SettingsPath },
		ScopeFunc:        func() string { return string(a.Scope) },
		FingerprintFunc: func(p agentharness.PermissionPolicy) string {
			return permissions.Fingerprint(permissions.Policy{BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow, NativeFingerprint: p.NativeFingerprint, HookFingerprint: p.HookFingerprint, SnapshotDigest: p.SnapshotDigest})
		},
		LoadStateFunc: func() (*agentharness.PermissionState, error) { return a.LoadState() },
		WriteStateFunc: func(p agentharness.PermissionPolicy, version string) error {
			return a.WriteState(permissions.Policy{BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow}, version)
		},
	}
}
