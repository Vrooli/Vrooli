// Package permissionscli registers the `permissions` subcommand group
// on resource-opencode. It is the only place that ties the adapter,
// the policy gate, and the agent-detection substrate together.
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
	"resource-opencode/cli/internal/permissions"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/agentpolicy"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Handlers owns the runtime dependencies. Tests inject a fake Adapter,
// override caller detection, and capture Stdout/Stderr.
type Handlers struct {
	Adapter        *permissions.Adapter
	DetectCaller   func() cliutil.CallerKind
	Policy         agentpolicy.Policy
	CLIVersion     string
	PinnedVersion  string
	VersionCommand []string
	VersionRunner  func(ctx context.Context, args []string) (string, error)
	Stdout         io.Writer
	Stderr         io.Writer
}

// Default returns Handlers wired to ~/.config/opencode/opencode.json
// and the live agent-detection substrate. pinnedVersion is the upstream
// CLI version this build was authored against; doctor warns when the
// installed version diverges.
func Default(cliVersion, pinnedVersion string) *Handlers {
	a, err := permissions.DefaultAdapter()
	if err != nil {
		// Surface the resolution failure lazily so `--help` still works.
		a = &permissions.Adapter{}
	}
	return &Handlers{
		Adapter:        a,
		DetectCaller:   cliutil.DetectCallerKind,
		Policy:         agentpolicy.DefaultPolicy(),
		CLIVersion:     cliVersion,
		PinnedVersion:  pinnedVersion,
		VersionCommand: []string{"opencode", "--version"},
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
		Description: "View and manage OpenCode permissions (~/.config/opencode/opencode.json permission.bash)",
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List managed permission patterns", Run: h.List},
			{Name: "show", Description: "Print the full opencode.json (raw or pretty)", Run: h.Show},
			{Name: "deny", Description: "Add a bash deny pattern (mutating)", Run: h.Deny},
			{Name: "allow", Description: "Add a bash allow pattern (mutating)", Run: h.Allow},
			{Name: "ask", Description: "Add a bash ask pattern (mutating)", Run: h.Ask},
			{Name: "remove", Description: "Remove a pattern from any list (mutating)", Run: h.Remove},
			{Name: "reset", Description: "Clear all Vrooli-managed permission entries (mutating)", Run: h.Reset},
			{Name: "drift-check", Description: "Compare current opencode.json fingerprint to last Vrooli write", Run: h.DriftCheck},
			{Name: "doctor", Description: "Check installed opencode version against the pinned upstream version", Run: h.Doctor},
		},
	}
}

func (h *Handlers) flagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	return fs
}

// --- read verbs -----------------------------------------------------------

func (h *Handlers) List(args []string) error {
	if err := h.flagSet("permissions list").Parse(args); err != nil {
		return err
	}
	p, err := h.Adapter.Load()
	if err != nil {
		return err
	}
	fmt.Fprintf(h.Stdout, "settings: %s\n", h.Adapter.SettingsPath)
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
	fs := h.flagSet("permissions show")
	raw := fs.Bool("raw", false, "Print the file verbatim (no canonicalisation)")
	asJSON := fs.Bool("json", false, "Print structured JSON of the managed policy only")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *raw {
		data, err := os.ReadFile(h.Adapter.SettingsPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", h.Adapter.SettingsPath, err)
		}
		_, err = h.Stdout.Write(data)
		return err
	}
	p, err := h.Adapter.Load()
	if err != nil {
		return err
	}
	if *asJSON {
		out, _ := json.MarshalIndent(map[string]any{
			"settingsPath": h.Adapter.SettingsPath,
			"deny":         p.BashDeny,
			"ask":          p.BashAsk,
			"allow":        p.BashAllow,
			"fingerprint":  permissions.Fingerprint(p),
		}, "", "  ")
		fmt.Fprintln(h.Stdout, string(out))
		return nil
	}
	return h.List(nil)
}

func (h *Handlers) DriftCheck(args []string) error {
	if err := h.flagSet("permissions drift-check").Parse(args); err != nil {
		return err
	}
	live, err := h.Adapter.Load()
	if err != nil {
		return err
	}
	liveFP := permissions.Fingerprint(live)
	st, err := h.Adapter.LoadState()
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
		fmt.Fprintln(h.Stdout, "clean — opencode.json matches the last Vrooli write")
		return nil
	}
	fmt.Fprintf(h.Stderr, "drift detected\n  last write:   %s\n  current file: %s\n  last writer:  %s @ %s\n",
		st.Fingerprint, liveFP, st.WrittenByVer, st.WrittenAt.Format("2006-01-02T15:04:05Z"))
	return errors.New("drift detected")
}

func (h *Handlers) Doctor(args []string) error {
	fs := h.flagSet("permissions doctor")
	pinned := fs.String("pinned-version", h.PinnedVersion, "Override the pinned upstream-CLI version for this check")
	if err := fs.Parse(args); err != nil {
		return err
	}
	fmt.Fprintf(h.Stdout, "resource-opencode version: %s\n", h.CLIVersion)
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
		} else {
			fmt.Fprintf(h.Stdout, "pinned %q matches installed\n", *pinned)
		}
	}
	return nil
}

// --- mutating verbs -------------------------------------------------------

func (h *Handlers) Deny(args []string) error  { return h.mutate("deny", args) }
func (h *Handlers) Allow(args []string) error { return h.mutate("allow", args) }
func (h *Handlers) Ask(args []string) error   { return h.mutate("ask", args) }

func (h *Handlers) mutate(verb string, args []string) error {
	fs := h.flagSet("permissions " + verb)
	auth := fs.Bool("i-was-explicitly-authorized", false, "Override the agent gate (only when a human explicitly authorized this call)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: permissions %s <pattern>", verb)
	}
	if err := h.gate("permissions "+verb, true, *auth); err != nil {
		return err
	}
	p, err := h.Adapter.Load()
	if err != nil {
		return err
	}
	pattern := fs.Arg(0)
	// Ensure pattern doesn't appear in multiple lists — last write wins.
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
	if err := h.Adapter.Save(p); err != nil {
		return err
	}
	if err := h.Adapter.WriteState(p, h.CLIVersion); err != nil {
		return err
	}
	fmt.Fprintf(h.Stdout, "%s %s -> %s\n", verb, pattern, h.Adapter.SettingsPath)
	return nil
}

func (h *Handlers) Remove(args []string) error {
	fs := h.flagSet("permissions remove")
	auth := fs.Bool("i-was-explicitly-authorized", false, "Override the agent gate")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return errors.New("usage: permissions remove <pattern>")
	}
	if err := h.gate("permissions remove", true, *auth); err != nil {
		return err
	}
	p, err := h.Adapter.Load()
	if err != nil {
		return err
	}
	pattern := fs.Arg(0)
	p.BashDeny = removeOne(p.BashDeny, pattern)
	p.BashAsk = removeOne(p.BashAsk, pattern)
	p.BashAllow = removeOne(p.BashAllow, pattern)
	if err := h.Adapter.Save(p); err != nil {
		return err
	}
	if err := h.Adapter.WriteState(p, h.CLIVersion); err != nil {
		return err
	}
	fmt.Fprintf(h.Stdout, "removed %s from any list\n", pattern)
	return nil
}

func (h *Handlers) Reset(args []string) error {
	fs := h.flagSet("permissions reset")
	auth := fs.Bool("i-was-explicitly-authorized", false, "Override the agent gate")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := h.gate("permissions reset", true, *auth); err != nil {
		return err
	}
	p := permissions.Policy{}
	if err := h.Adapter.Save(p); err != nil {
		return err
	}
	if err := h.Adapter.WriteState(p, h.CLIVersion); err != nil {
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
	switch agentpolicy.Decide(kind, cmd, flags, h.Policy) {
	case agentpolicy.DecisionAllow:
		return nil
	case agentpolicy.DecisionWarn:
		fmt.Fprintln(h.Stderr, agentpolicy.RenderDenyMessage(agentpolicy.DenyContext{ResourceLabel: "resource-opencode", ConfigPath: "~/.config/opencode/opencode.json permission.bash"}, cmd, h.Policy))
		return nil
	case agentpolicy.DecisionDeny:
		return errors.New(agentpolicy.RenderDenyMessage(agentpolicy.DenyContext{ResourceLabel: "resource-opencode", ConfigPath: "~/.config/opencode/opencode.json permission.bash"}, cmd, h.Policy))
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
