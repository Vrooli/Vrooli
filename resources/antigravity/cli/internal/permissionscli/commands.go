// Package permissionscli registers the `permissions` subcommand group on
// resource-antigravity. It is the only place that ties the settings adapter, the
// policy gate, and the agent-detection substrate together.
//
// Antigravity-specific notes:
//   - The managed grants live in the native `permissions` object of
//     ~/.gemini/antigravity-cli/settings.json (the global/user scope). Antigravity
//     reads these directly — this is the native enforcement seam. A project
//     hook projection is available when VROOLI_AGENT_HOOK_PATH is configured.
//   - Rules use Antigravity's native `action(target)` vocabulary
//     (`command(rm -rf)`, `read_file(*)`, `mcp(*)`), stored as the `permissions`
//     object's `allow`/`deny`/`ask` string arrays (precedence Deny > Ask >
//     Allow). Schema confirmed 2026-06-29 via antigravity.google/docs/cli-permissions
//     and the on-disk settings agy 1.0.13 writes.
package permissionscli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"resource-antigravity/cli/internal/permissions"

	"github.com/vrooli/cli-core/agentpolicy"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	resourceLabel = "resource-antigravity"
	configHint    = "~/.gemini/antigravity-cli/settings.json [permissions]"
)

// Handlers owns the runtime dependencies. Tests inject a fake Adapter, override
// caller detection, and capture Stdout/Stderr.
type Handlers struct {
	// AdapterFor resolves the adapter for the requested scope. Tests inject a
	// stub that ignores Scope and returns a TempDir-rooted Adapter. Default
	// wires permissions.DefaultAdapter.
	AdapterFor     func(scope permissions.Scope) (*permissions.Adapter, error)
	DetectCaller   func() cliutil.CallerKind
	Policy         agentpolicy.Policy
	CLIVersion     string
	PinnedVersion  string
	VersionCommand []string
	VersionRunner  func(ctx context.Context, args []string) (string, error)
	Stdout         io.Writer
	Stderr         io.Writer
}

// Default returns Handlers wired to ~/.gemini/antigravity-cli/settings.json and
// the live agent-detection substrate.
func Default(cliVersion, pinnedVersion string) *Handlers {
	return &Handlers{
		AdapterFor:     permissions.DefaultAdapter,
		DetectCaller:   cliutil.DetectCallerKind,
		Policy:         agentpolicy.DefaultPolicy(),
		CLIVersion:     cliVersion,
		PinnedVersion:  pinnedVersion,
		VersionCommand: []string{"agy", "--version"},
		VersionRunner:  defaultVersionRunner,
		Stdout:         os.Stdout,
		Stderr:         os.Stderr,
	}
}

func defaultVersionRunner(ctx context.Context, args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("empty version command")
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// Commands returns the `permissions` subgroup for registration.
func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default("", "")
	}
	return cliapp.SubcommandGroup{
		Name:        "permissions",
		Description: "View and manage Antigravity permission grants (~/.gemini/antigravity-cli/settings.json `permissions`)",
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List managed permission patterns", Run: h.List},
			{Name: "show", Description: "Print the full settings file (raw or pretty)", Run: h.Show},
			{Name: "deny", Description: "Add a deny rule, e.g. 'command(rm -rf)' (mutating)", Run: h.Deny},
			{Name: "allow", Description: "Add an allow rule, e.g. 'command(git)' (mutating)", Run: h.Allow},
			{Name: "ask", Description: "Add an ask rule, e.g. 'mcp(*)' (mutating)", Run: h.Ask},
			{Name: "remove", Description: "Remove a pattern from any list (mutating)", Run: h.Remove},
			{Name: "reset", Description: "Clear all Vrooli-managed permission entries (mutating)", Run: h.Reset},
			{Name: "drift-check", Description: "Compare current settings fingerprint to last Vrooli write", Run: h.DriftCheck},
			{Name: "doctor", Description: "Check installed agy version and confirm enforcement wiring", Run: h.Doctor},
		},
	}
}

func (h *Handlers) flagSet(name string) (*flag.FlagSet, *string) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	scope := fs.String("scope", string(permissions.ScopeUser), "Config scope: user (global ~/.gemini/antigravity-cli/settings.json)")
	return fs, scope
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

// --- read verbs -----------------------------------------------------------

func (h *Handlers) List(args []string) error {
	fs, scope := h.flagSet("permissions list")
	if err := fs.Parse(args); err != nil {
		return err
	}
	a, err := h.adapter(*scope)
	if err != nil {
		return err
	}
	p, err := a.Load()
	if err != nil {
		return err
	}
	fmt.Fprintf(h.Stdout, "settings: %s (scope=%s)\n", a.SettingsPath, a.Scope)
	printList(h.Stdout, "deny", p.BashDeny)
	printList(h.Stdout, "ask", p.BashAsk)
	printList(h.Stdout, "allow", p.BashAllow)
	return nil
}

func printList(w io.Writer, label string, items []string) {
	if len(items) == 0 {
		fmt.Fprintf(w, "  %s: (none)\n", label)
		return
	}
	fmt.Fprintf(w, "  %s:\n", label)
	for _, it := range items {
		fmt.Fprintf(w, "    - %s\n", it)
	}
}

func (h *Handlers) Show(args []string) error {
	fs, scope := h.flagSet("permissions show")
	raw := fs.Bool("raw", false, "Print the file verbatim (no canonicalisation)")
	asJSON := fs.Bool("json", false, "Print structured JSON of the managed policy only")
	if err := fs.Parse(args); err != nil {
		return err
	}
	a, err := h.adapter(*scope)
	if err != nil {
		return err
	}
	if *raw {
		data, err := os.ReadFile(a.SettingsPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", a.SettingsPath, err)
		}
		_, err = h.Stdout.Write(data)
		return err
	}
	p, err := a.Load()
	if err != nil {
		return err
	}
	if *asJSON {
		out, _ := json.MarshalIndent(map[string]any{
			"settingsPath": a.SettingsPath,
			"scope":        a.Scope,
			"deny":         p.BashDeny,
			"ask":          p.BashAsk,
			"allow":        p.BashAllow,
			"fingerprint":  permissions.Fingerprint(p),
		}, "", "  ")
		fmt.Fprintln(h.Stdout, string(out))
		return nil
	}
	return h.List([]string{"-scope", *scope})
}

func (h *Handlers) DriftCheck(args []string) error {
	fs, scope := h.flagSet("permissions drift-check")
	if err := fs.Parse(args); err != nil {
		return err
	}
	a, err := h.adapter(*scope)
	if err != nil {
		return err
	}
	live, err := a.Load()
	if err != nil {
		return err
	}
	liveFP := permissions.Fingerprint(live)
	st, err := a.LoadState()
	if err != nil {
		return err
	}
	if st == nil {
		fmt.Fprintln(h.Stdout, "no prior Vrooli write recorded; nothing to compare")
		return nil
	}
	if st.SchemaVersion != permissions.StateSchemaVersion {
		return fmt.Errorf("state schema %d != current %d (state file is from a different Vrooli version)", st.SchemaVersion, permissions.StateSchemaVersion)
	}
	if st.Fingerprint == liveFP {
		fmt.Fprintf(h.Stdout, "clean — %s matches the last Vrooli write\n", a.SettingsPath)
		return nil
	}
	fmt.Fprintf(h.Stderr, "drift detected\n  last write:   %s\n  current file: %s\n  last writer:  %s @ %s\n",
		st.Fingerprint, liveFP, st.WrittenByVer, st.WrittenAt.Format("2006-01-02T15:04:05Z"))
	return errors.New("drift detected")
}

func (h *Handlers) Doctor(args []string) error {
	fs, scope := h.flagSet("permissions doctor")
	pinned := fs.String("pinned-version", h.PinnedVersion, "Override the pinned upstream-CLI version for this check")
	if err := fs.Parse(args); err != nil {
		return err
	}
	a, err := h.adapter(*scope)
	if err != nil {
		return err
	}
	fmt.Fprintf(h.Stdout, "resource-antigravity version: %s\n", h.CLIVersion)
	fmt.Fprintf(h.Stdout, "checking upstream CLI: %v\n", h.VersionCommand)
	ctx := context.Background()
	got, err := h.VersionRunner(ctx, h.VersionCommand)
	if err != nil {
		fmt.Fprintf(h.Stderr, "warn: could not run %v: %v\n", h.VersionCommand, err)
	} else {
		fmt.Fprintf(h.Stdout, "installed: %s\n", got)
	}
	if strings.TrimSpace(*pinned) != "" {
		if got != "" && !strings.Contains(got, *pinned) {
			fmt.Fprintf(h.Stderr, "warn: pinned %q not found in installed %q — schema may have drifted\n", *pinned, got)
		} else if got != "" {
			fmt.Fprintf(h.Stdout, "pinned %q matches installed\n", *pinned)
		}
	}
	// Enforcement model: Antigravity reads allow/deny/ask grants from the
	// native `permissions` object in settings.json. The exact JSON rule-encoding is confirmed against a
	// live agy grant during post-sign-in validation — report that honestly.
	fmt.Fprintln(h.Stdout, "enforcement: Antigravity reads allow/deny/ask grants from the native `permissions`")
	fmt.Fprintln(h.Stdout, "             object in settings.json, evaluated Deny > Ask > Allow. Rules use the")
	fmt.Fprintln(h.Stdout, "             action(target) vocabulary, e.g. command(rm -rf), read_file(*), mcp(*).")
	fmt.Fprintln(h.Stdout, "             A project-scoped PreToolUse command hook is projected when VROOLI_AGENT_HOOK_PATH is set;")
	fmt.Fprintln(h.Stdout, "             hook firing requires a live canary. Schema confirmed 2026-06-29; run a denied command inside an")
	fmt.Fprintln(h.Stdout, "             agy session to confirm end-to-end enforcement (live canary).")
	if info, err := os.Stat(a.SettingsPath); err == nil {
		fmt.Fprintf(h.Stdout, "settings: present at %s (%d bytes)\n", a.SettingsPath, info.Size())
	} else {
		fmt.Fprintf(h.Stdout, "settings: not yet present at %s (agy writes it on first run / first grant)\n", a.SettingsPath)
	}
	return nil
}

// --- mutating verbs -------------------------------------------------------

func (h *Handlers) Deny(args []string) error  { return h.mutate("deny", args) }
func (h *Handlers) Allow(args []string) error { return h.mutate("allow", args) }
func (h *Handlers) Ask(args []string) error   { return h.mutate("ask", args) }

func (h *Handlers) mutate(verb string, args []string) error {
	fs, scope := h.flagSet("permissions " + verb)
	auth := fs.Bool("i-was-explicitly-authorized", false, "Override the agent gate (only when a human explicitly authorized this call)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: permissions %s [--scope user] <pattern>", verb)
	}
	if err := h.gate("permissions "+verb, true, *auth); err != nil {
		return err
	}
	a, err := h.adapter(*scope)
	if err != nil {
		return err
	}
	p, err := a.Load()
	if err != nil {
		return err
	}
	pattern := fs.Arg(0)
	// Ensure a pattern lives in exactly one bucket.
	p.BashDeny = removeOne(p.BashDeny, pattern)
	p.BashAsk = removeOne(p.BashAsk, pattern)
	p.BashAllow = removeOne(p.BashAllow, pattern)
	switch verb {
	case "deny":
		p.BashDeny = addUnique(p.BashDeny, pattern)
	case "allow":
		p.BashAllow = addUnique(p.BashAllow, pattern)
	case "ask":
		p.BashAsk = addUnique(p.BashAsk, pattern)
	}
	if err := a.Save(p); err != nil {
		return err
	}
	if err := a.WriteState(p, h.CLIVersion); err != nil {
		return err
	}
	fmt.Fprintf(h.Stdout, "%s %s -> %s (scope=%s)\n", verb, pattern, a.SettingsPath, a.Scope)
	return nil
}

func (h *Handlers) Remove(args []string) error {
	fs, scope := h.flagSet("permissions remove")
	auth := fs.Bool("i-was-explicitly-authorized", false, "Override the agent gate")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return errors.New("usage: permissions remove [--scope user] <pattern>")
	}
	if err := h.gate("permissions remove", true, *auth); err != nil {
		return err
	}
	a, err := h.adapter(*scope)
	if err != nil {
		return err
	}
	p, err := a.Load()
	if err != nil {
		return err
	}
	pattern := fs.Arg(0)
	p.BashDeny = removeOne(p.BashDeny, pattern)
	p.BashAsk = removeOne(p.BashAsk, pattern)
	p.BashAllow = removeOne(p.BashAllow, pattern)
	if err := a.Save(p); err != nil {
		return err
	}
	if err := a.WriteState(p, h.CLIVersion); err != nil {
		return err
	}
	fmt.Fprintf(h.Stdout, "removed %s from any list\n", pattern)
	return nil
}

func (h *Handlers) Reset(args []string) error {
	fs, scope := h.flagSet("permissions reset")
	auth := fs.Bool("i-was-explicitly-authorized", false, "Override the agent gate")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := h.gate("permissions reset", true, *auth); err != nil {
		return err
	}
	a, err := h.adapter(*scope)
	if err != nil {
		return err
	}
	p := permissions.Policy{}
	if err := a.Save(p); err != nil {
		return err
	}
	if err := a.WriteState(p, h.CLIVersion); err != nil {
		return err
	}
	fmt.Fprintln(h.Stdout, "cleared all Vrooli-managed permission entries")
	return nil
}

// --- helpers --------------------------------------------------------------

func (h *Handlers) gate(verb string, mutating bool, authorized bool) error {
	kind := h.DetectCaller()
	cmd := agentpolicy.CommandSpec{Name: verb, Mutating: mutating}
	flags := agentpolicy.CallerOverrideFlags{AuthorizedByUser: authorized}
	dctx := agentpolicy.DenyContext{ResourceLabel: resourceLabel, ConfigPath: configHint}
	switch agentpolicy.Decide(kind, cmd, flags, h.Policy) {
	case agentpolicy.DecisionAllow:
		return nil
	case agentpolicy.DecisionWarn:
		fmt.Fprintln(h.Stderr, agentpolicy.RenderDenyMessage(dctx, cmd, h.Policy))
		return nil
	case agentpolicy.DecisionDeny:
		return errors.New(agentpolicy.RenderDenyMessage(dctx, cmd, h.Policy))
	default:
		return errors.New("agentpolicy: unknown decision")
	}
}

func addUnique(s []string, v string) []string {
	for _, e := range s {
		if e == v {
			return s
		}
	}
	out := append(s, v)
	sort.Strings(out)
	return out
}

func removeOne(s []string, v string) []string {
	out := make([]string, 0, len(s))
	for _, e := range s {
		if e != v {
			out = append(out, e)
		}
	}
	return out
}
