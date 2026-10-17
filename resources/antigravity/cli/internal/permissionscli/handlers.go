// Package permissionscli registers the `permissions` subcommand group on
// resource-antigravity. Native Antigravity storage stays in the resource
// adapter; the common CRUD and gate surface lives in agentharness.
package permissionscli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/resources/antigravity/cli/internal/permissions"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	resourceLabel = "resource-antigravity"
	configHint    = "~/.gemini/antigravity-cli/settings.json [permissions]"
)

type Handlers struct {
	AdapterFor     func(scope permissions.Scope) (*permissions.Adapter, error)
	DetectCaller   func() cliutil.CallerKind
	Policy         agentharness.Policy
	CLIVersion     string
	PinnedVersion  string
	VersionCommand []string
	VersionRunner  func(context.Context, []string) (string, error)
	Stdout         io.Writer
	Stderr         io.Writer
}

func Default(cliVersion, pinnedVersion string) *Handlers {
	return &Handlers{
		AdapterFor: permissions.DefaultAdapter, DetectCaller: cliutil.DetectCallerKind,
		Policy: agentharness.DefaultPolicy(), CLIVersion: cliVersion, PinnedVersion: pinnedVersion,
		VersionCommand: []string{"agy", "--version"}, VersionRunner: defaultVersionRunner,
		Stdout: os.Stdout, Stderr: os.Stderr,
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
	case permissions.ScopeUser, "":
	default:
		return nil, fmt.Errorf("unknown --scope %q (only 'user' is supported; project-scoped grants are a follow-up)", scopeRaw)
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
		ScopeHelp:    "Config scope: user (global ~/.gemini/antigravity-cli/settings.json)",
		Description:  "View and manage Antigravity permission grants (~/.gemini/antigravity-cli/settings.json `permissions`)",
		CommandDescriptions: map[string]string{
			"show":        "Print the full settings file (raw or pretty)",
			"deny":        "Add a deny rule, e.g. 'command(rm -rf)' (mutating)",
			"allow":       "Add an allow rule, e.g. 'command(git)' (mutating)",
			"ask":         "Add an ask rule, e.g. 'mcp(*)' (mutating)",
			"drift-check": "Compare current settings fingerprint to last Vrooli write",
			"doctor":      "Check installed agy version and confirm enforcement wiring",
		},
		ResourceLabel: resourceLabel, ConfigHint: configHint, VersionLabel: resourceLabel,
		CLIVersion: h.CLIVersion, PinnedVersion: h.PinnedVersion, VersionCommand: h.VersionCommand,
		VersionRunner: h.VersionRunner, DetectCaller: h.DetectCaller, Policy: h.Policy,
		ExclusivePatterns: true,
		Stdout:            h.Stdout, Stderr: h.Stderr,
		DoctorExtra: func(stdout, _ io.Writer, shared agentharness.PermissionAdapter) error {
			a, err := h.adapter(shared.Scope())
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(stdout, "enforcement: Antigravity reads allow/deny/ask grants from the native `permissions`")
			_, _ = fmt.Fprintln(stdout, "             object in settings.json, evaluated Deny > Ask > Allow. Rules use the")
			_, _ = fmt.Fprintln(stdout, "             action(target) vocabulary, e.g. command(rm -rf), read_file(*), mcp(*).")
			_, _ = fmt.Fprintln(stdout, "             A project-scoped PreToolUse command hook is projected when VROOLI_AGENT_HOOK_PATH is set;")
			_, _ = fmt.Fprintln(stdout, "             hook firing requires a live canary. Schema confirmed 2026-06-29; run a denied command inside an")
			_, _ = fmt.Fprintln(stdout, "             agy session to confirm end-to-end enforcement (live canary).")
			if info, statErr := os.Stat(a.SettingsPath); statErr == nil {
				_, _ = fmt.Fprintf(stdout, "settings: present at %s (%d bytes)\n", a.SettingsPath, info.Size())
			} else {
				_, _ = fmt.Fprintf(stdout, "settings: not yet present at %s (agy writes it on first run / first grant)\n", a.SettingsPath)
			}
			return nil
		},
	})
}

func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default("", "")
	}
	return h.shared().Commands(nil)
}

func HookCommands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default("", "")
	}
	return agentharness.HookCommands(agentharness.HookCommandConfig{
		Agent: "antigravity", Description: "Reconcile Antigravity hooks through the shared Vrooli hook broker",
		ScopeDefault: string(permissions.ScopeUser),
		ScopeHelp:    "Hook scope: project path from VROOLI_AGENT_HOOK_PATH",
		Target: func(scope string) (agentharness.HookTarget, error) {
			a, err := h.adapter(scope)
			if err != nil {
				return agentharness.HookTarget{}, err
			}
			path := a.HookPath
			if strings.TrimSpace(path) == "" {
				root, _ := os.Getwd()
				path = filepath.Join(root, ".agents", "hooks.json")
			}
			return agentharness.HookTarget{Agent: "antigravity", Path: path}, nil
		},
		Stdout: h.Stdout, Stderr: h.Stderr,
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

func bridge(a *permissions.Adapter) agentharness.PermissionAdapter {
	return agentharness.PermissionAdapterFuncs{
		LoadFunc: func() (agentharness.PermissionPolicy, error) {
			p, err := a.Load()
			return agentharness.PermissionPolicy{BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow, SettingsPath: a.SettingsPath}, err
		},
		SaveFunc: func(p agentharness.PermissionPolicy) error {
			return a.Save(permissions.Policy{BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow})
		},
		SettingsPathFunc: func() string { return a.SettingsPath },
		ScopeFunc:        func() string { return string(a.Scope) },
		FingerprintFunc: func(p agentharness.PermissionPolicy) string {
			return permissions.Fingerprint(permissions.Policy{BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow})
		},
		LoadStateFunc: func() (*agentharness.PermissionState, error) { return a.LoadState() },
		WriteStateFunc: func(p agentharness.PermissionPolicy, version string) error {
			return a.WriteState(permissions.Policy{BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow}, version)
		},
	}
}
