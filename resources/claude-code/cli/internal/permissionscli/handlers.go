// Package permissionscli registers the `permissions` subcommand group on
// resource-claude-code. The CRUD, gate, state, and doctor plumbing is shared
// with the other coding-agent CLIs; this package supplies the native adapter
// bridge and Claude-specific document and hook commands.
package permissionscli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/vrooli/resources/claude-code/cli/internal/permissions"

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
	VersionRunner  func(ctx context.Context, args []string) (string, error)
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
		VersionCommand: []string{"claude", "--version"},
		Stdin:          os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr,
	}
}

func (h *Handlers) shared() *agentharness.PermissionHandlers {
	return agentharness.NewPermissionHandlers(agentharness.PermissionCommandConfig{
		AdapterFor: func(string) (agentharness.PermissionAdapter, error) {
			if h.Adapter == nil {
				return nil, errors.New("Claude settings adapter is unavailable")
			}
			return bridge(h.Adapter, h.CLIVersion), nil
		},
		Description: "View and manage Claude Code permissions (~/.claude/settings.json) and the paired PreToolUse hook",
		CommandDescriptions: map[string]string{
			"deny":        "Add a Bash deny pattern (mutating)",
			"allow":       "Add a Bash allow pattern (mutating)",
			"ask":         "Add a Bash ask pattern (mutating)",
			"show":        "Print the full settings.json (raw or pretty)",
			"doctor":      "Check installed claude version against the pinned upstream version",
			"drift-check": "Compare current settings.json fingerprint to last Vrooli write",
		},
		ResourceLabel: "resource-claude-code", ConfigHint: "~/.claude/settings.json",
		VersionLabel: "resource-claude-code", CLIVersion: h.CLIVersion, PinnedVersion: h.PinnedVersion,
		VersionCommand: h.VersionCommand, VersionRunner: h.VersionRunner,
		DetectCaller: h.DetectCaller, Policy: h.Policy, ResetPolicy: agentharness.PermissionPolicy{Hooks: true},
		ListSuffix: func(stdout io.Writer, adapter agentharness.PermissionAdapter) {
			p, err := adapter.Load()
			if err == nil {
				_, _ = fmt.Fprintf(stdout, "hooks-paired: %v\n", p.Hooks)
			}
		},
		ShowFields: func(fields map[string]any, adapter agentharness.PermissionAdapter, _ agentharness.PermissionPolicy) {
			p, err := adapter.Load()
			if err == nil {
				fields["hooksPaired"] = p.Hooks
			}
		},
		ExclusivePatterns: false,
		PlanDocument:      h.planDocument,
		Stdout:            h.Stdout, Stderr: h.Stderr,
		DoctorExtra: func(stdout, _ io.Writer, _ agentharness.PermissionAdapter) error {
			_, _ = io.WriteString(stdout, "PreToolUse guard: "+permissions.GuardCommand()+" "+strings.Join(permissions.GuardSubcommand, " ")+" (shared agent policy runtime; replay-verified)\n")
			return nil
		},
	})
}

func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default("", "")
	}
	shared := h.shared()
	return shared.Commands(append([]cliapp.Command{h.guardCommand()}, shared.DocumentCommands()...))
}

func bridge(a *permissions.Adapter, version string) agentharness.PermissionAdapter {
	return agentharness.PermissionAdapterFuncs{
		LoadFunc: func() (agentharness.PermissionPolicy, error) {
			p, err := a.Load()
			return agentharness.PermissionPolicy{BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow, Hooks: p.Hooks, SettingsPath: a.SettingsPath}, err
		},
		SaveFunc: func(p agentharness.PermissionPolicy) error {
			return a.Save(permissions.Policy{BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow, Hooks: p.Hooks})
		},
		SettingsPathFunc: func() string { return a.SettingsPath },
		FingerprintFunc: func(p agentharness.PermissionPolicy) string {
			return permissions.Fingerprint(permissions.Policy{BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow, Hooks: p.Hooks})
		},
		LoadStateFunc: func() (*agentharness.PermissionState, error) { return a.LoadState() },
		WriteStateFunc: func(p agentharness.PermissionPolicy, _ string) error {
			return a.WriteState(permissions.Policy{BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow, Hooks: p.Hooks}, version)
		},
	}
}
