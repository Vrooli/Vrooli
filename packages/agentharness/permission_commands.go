package agentharness

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

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// PermissionPolicy is the provider-neutral projection consumed by the shared
// permissions command surface. Native adapters remain responsible for mapping
// these three buckets into their JSON/TOML schemas.
type PermissionPolicy struct {
	BashDeny  []string
	BashAsk   []string
	BashAllow []string
	Hooks     bool
	// SettingsPath is metadata used only while writing the shared sidecar.
	SettingsPath string `json:"-"`
}

// PermissionAdapter is the narrow seam between the shared command surface and
// a provider-native adapter. Scope, serialization, hooks, and fingerprints
// remain owned by the native package.
type PermissionAdapter interface {
	Load() (PermissionPolicy, error)
	Save(PermissionPolicy) error
	SettingsPath() string
	Scope() string
	Fingerprint(PermissionPolicy) string
	LoadState() (*PermissionState, error)
	WriteState(PermissionPolicy, string) error
}

// PermissionAdapterFuncs makes the seam easy for resource packages to bridge
// without exporting their native Policy type into agentharness.
type PermissionAdapterFuncs struct {
	LoadFunc         func() (PermissionPolicy, error)
	SaveFunc         func(PermissionPolicy) error
	SettingsPathFunc func() string
	ScopeFunc        func() string
	FingerprintFunc  func(PermissionPolicy) string
	LoadStateFunc    func() (*PermissionState, error)
	WriteStateFunc   func(PermissionPolicy, string) error
}

func (a PermissionAdapterFuncs) Load() (PermissionPolicy, error) {
	return a.LoadFunc()
}

func (a PermissionAdapterFuncs) Save(policy PermissionPolicy) error {
	return a.SaveFunc(policy)
}

func (a PermissionAdapterFuncs) SettingsPath() string {
	return a.SettingsPathFunc()
}

func (a PermissionAdapterFuncs) Scope() string {
	if a.ScopeFunc == nil {
		return ""
	}
	return a.ScopeFunc()
}

func (a PermissionAdapterFuncs) Fingerprint(policy PermissionPolicy) string {
	return a.FingerprintFunc(policy)
}

func (a PermissionAdapterFuncs) LoadState() (*PermissionState, error) {
	return a.LoadStateFunc()
}

func (a PermissionAdapterFuncs) WriteState(policy PermissionPolicy, version string) error {
	return a.WriteStateFunc(policy, version)
}

// PermissionCommandConfig configures the shared CRUD/gate/drift/doctor
// commands while leaving provider-specific descriptions and diagnostics in
// the resource package.
type PermissionCommandConfig struct {
	AdapterFor   func(scope string) (PermissionAdapter, error)
	ScopeDefault string
	ScopeHelp    string

	Description         string
	CommandDescriptions map[string]string
	ResourceLabel       string
	ConfigHint          string
	VersionLabel        string
	CLIVersion          string
	PinnedVersion       string
	VersionCommand      []string
	VersionRunner       func(context.Context, []string) (string, error)
	DetectCaller        func() cliutil.CallerKind
	Policy              Policy
	ResetPolicy         PermissionPolicy
	ExclusivePatterns   bool
	ListSuffix          func(io.Writer, PermissionAdapter)
	ShowFields          func(map[string]any, PermissionAdapter, PermissionPolicy)
	DoctorExtra         func(io.Writer, io.Writer, PermissionAdapter) error
	Stdout              io.Writer
	Stderr              io.Writer
}

// PermissionHandlers owns the common permissions command behavior.
type PermissionHandlers struct {
	cfg PermissionCommandConfig
}

// NewPermissionHandlers constructs the shared command implementation.
func NewPermissionHandlers(cfg PermissionCommandConfig) *PermissionHandlers {
	if cfg.DetectCaller == nil {
		cfg.DetectCaller = cliutil.DetectCallerKind
	}
	if cfg.Policy.AgentAccess == "" {
		cfg.Policy = DefaultPolicy()
	}
	if cfg.VersionRunner == nil {
		cfg.VersionRunner = defaultPermissionVersionRunner
	}
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}
	if cfg.Stderr == nil {
		cfg.Stderr = os.Stderr
	}
	return &PermissionHandlers{cfg: cfg}
}

// Commands returns the one shared permissions command surface. Resource
// packages may insert plan/reconcile/migrate commands before doctor.
func (h *PermissionHandlers) Commands(extras []cliapp.Command) cliapp.SubcommandGroup {
	d := h.cfg.CommandDescriptions
	command := func(name, fallback string) string {
		if value := d[name]; value != "" {
			return value
		}
		return fallback
	}
	commands := []cliapp.Command{
		{Name: "list", Description: command("list", "List managed permission patterns"), Run: h.List},
		{Name: "show", Description: command("show", "Print the full config file (raw or pretty)"), Run: h.Show},
		{Name: "deny", Description: command("deny", "Add a deny pattern (mutating)"), Run: h.Deny},
		{Name: "allow", Description: command("allow", "Add an allow pattern (mutating)"), Run: h.Allow},
		{Name: "ask", Description: command("ask", "Add an ask pattern (mutating)"), Run: h.Ask},
		{Name: "remove", Description: command("remove", "Remove a pattern from any list (mutating)"), Run: h.Remove},
		{Name: "reset", Description: command("reset", "Clear all Vrooli-managed permission entries (mutating)"), Run: h.Reset},
		{Name: "drift-check", Description: command("drift-check", "Compare current config fingerprint to last Vrooli write"), Run: h.DriftCheck},
	}
	commands = append(commands, extras...)
	commands = append(commands, cliapp.Command{Name: "doctor", Description: command("doctor", "Check installed CLI version against the pinned upstream version"), Run: h.Doctor})
	return cliapp.SubcommandGroup{Name: "permissions", Description: h.cfg.Description, Subcommands: commands}
}

func (h *PermissionHandlers) flagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(h.cfg.Stderr)
	if h.scopeEnabled() {
		fs.String("scope", h.cfg.ScopeDefault, h.cfg.ScopeHelp)
	}
	return fs
}

func (h *PermissionHandlers) scope(name string) (*flag.FlagSet, string) {
	fs := h.flagSet(name)
	if !h.scopeEnabled() {
		return fs, ""
	}
	return fs, h.cfg.ScopeDefault
}

func (h *PermissionHandlers) scopeEnabled() bool {
	return h.cfg.ScopeHelp != "" || h.cfg.ScopeDefault != ""
}

func (h *PermissionHandlers) adapter(scope string) (PermissionAdapter, error) {
	if h.cfg.AdapterFor == nil {
		return nil, errors.New("permissions adapter is unavailable")
	}
	a, err := h.cfg.AdapterFor(scope)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, errors.New("adapter resolution returned nil")
	}
	return a, nil
}

// List prints the managed policy without requiring authorization.
func (h *PermissionHandlers) List(args []string) error {
	fs, scope := h.scope("permissions list")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if h.scopeEnabled() {
		scope = fs.Lookup("scope").Value.String()
	}
	a, err := h.adapter(scope)
	if err != nil {
		return err
	}
	p, err := a.Load()
	if err != nil {
		return err
	}
	fmt.Fprintf(h.cfg.Stdout, "settings: %s", a.SettingsPath())
	if h.scopeEnabled() {
		fmt.Fprintf(h.cfg.Stdout, " (scope=%s)", a.Scope())
	}
	fmt.Fprintln(h.cfg.Stdout)
	printPermissionList(h.cfg.Stdout, "deny", p.BashDeny)
	printPermissionList(h.cfg.Stdout, "ask", p.BashAsk)
	printPermissionList(h.cfg.Stdout, "allow", p.BashAllow)
	if h.cfg.ListSuffix != nil {
		h.cfg.ListSuffix(h.cfg.Stdout, a)
	}
	return nil
}

func printPermissionList(w io.Writer, label string, items []string) {
	if len(items) == 0 {
		fmt.Fprintf(w, "  %s: (none)\n", label)
		return
	}
	fmt.Fprintf(w, "  %s:\n", label)
	for _, item := range items {
		fmt.Fprintf(w, "    - %s\n", item)
	}
}

// Show prints either the native file or the common managed projection.
func (h *PermissionHandlers) Show(args []string) error {
	fs, scope := h.scope("permissions show")
	raw := fs.Bool("raw", false, "Print the file verbatim (no canonicalisation)")
	asJSON := fs.Bool("json", false, "Print structured JSON of the managed policy only")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if h.scopeEnabled() {
		scope = fs.Lookup("scope").Value.String()
	}
	a, err := h.adapter(scope)
	if err != nil {
		return err
	}
	if *raw {
		data, err := os.ReadFile(a.SettingsPath())
		if err != nil {
			return fmt.Errorf("read %s: %w", a.SettingsPath(), err)
		}
		_, err = h.cfg.Stdout.Write(data)
		return err
	}
	p, err := a.Load()
	if err != nil {
		return err
	}
	if *asJSON {
		fields := map[string]any{
			"settingsPath": a.SettingsPath(),
			"deny":         p.BashDeny,
			"ask":          p.BashAsk,
			"allow":        p.BashAllow,
			"fingerprint":  a.Fingerprint(p),
		}
		if h.scopeEnabled() {
			fields["scope"] = a.Scope()
		}
		if h.cfg.ShowFields != nil {
			h.cfg.ShowFields(fields, a, p)
		}
		data, _ := json.MarshalIndent(fields, "", "  ")
		fmt.Fprintln(h.cfg.Stdout, string(data))
		return nil
	}
	listArgs := []string(nil)
	if h.scopeEnabled() {
		listArgs = []string{"-scope", scope}
	}
	return h.List(listArgs)
}

// DriftCheck compares the live native file with the shared sidecar.
func (h *PermissionHandlers) DriftCheck(args []string) error {
	fs, scope := h.scope("permissions drift-check")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if h.scopeEnabled() {
		scope = fs.Lookup("scope").Value.String()
	}
	a, err := h.adapter(scope)
	if err != nil {
		return err
	}
	live, err := a.Load()
	if err != nil {
		return err
	}
	liveFingerprint := a.Fingerprint(live)
	state, err := a.LoadState()
	if err != nil {
		return err
	}
	if state == nil {
		fmt.Fprintln(h.cfg.Stdout, "no prior Vrooli write recorded; nothing to compare")
		return nil
	}
	if state.SchemaVersion != PermissionStateSchemaVersion {
		return fmt.Errorf("state schema %d != current %d (state file is from a different Vrooli version)", state.SchemaVersion, PermissionStateSchemaVersion)
	}
	if state.Fingerprint == liveFingerprint {
		if h.scopeEnabled() {
			fmt.Fprintf(h.cfg.Stdout, "clean — %s matches the last Vrooli write\n", a.SettingsPath())
		} else {
			fmt.Fprintln(h.cfg.Stdout, "clean — settings.json matches the last Vrooli write")
		}
		return nil
	}
	fmt.Fprintf(h.cfg.Stderr, "drift detected\n  last write:   %s\n  current file: %s\n  last writer:  %s @ %s\n", state.Fingerprint, liveFingerprint, state.WrittenByVer, state.WrittenAt.Format("2006-01-02T15:04:05Z"))
	return errors.New("drift detected")
}

// Doctor reports the provider version and lets the native package append
// provider-specific enforcement notes.
func (h *PermissionHandlers) Doctor(args []string) error {
	fs, scope := h.scope("permissions doctor")
	pinned := fs.String("pinned-version", h.cfg.PinnedVersion, "Override the pinned upstream-CLI version for this check")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if h.scopeEnabled() {
		scope = fs.Lookup("scope").Value.String()
	}
	a, err := h.adapter(scope)
	if err != nil {
		return err
	}
	fmt.Fprintf(h.cfg.Stdout, "%s version: %s\n", h.cfg.VersionLabel, h.cfg.CLIVersion)
	fmt.Fprintf(h.cfg.Stdout, "checking upstream CLI: %v\n", h.cfg.VersionCommand)
	got, runErr := h.cfg.VersionRunner(context.Background(), h.cfg.VersionCommand)
	if runErr != nil {
		fmt.Fprintf(h.cfg.Stderr, "warn: could not run %v: %v\n", h.cfg.VersionCommand, runErr)
	} else {
		fmt.Fprintf(h.cfg.Stdout, "installed: %s\n", got)
	}
	if strings.TrimSpace(*pinned) != "" {
		if got != "" && !strings.Contains(got, *pinned) {
			fmt.Fprintf(h.cfg.Stderr, "warn: pinned %q not found in installed %q — schema may have drifted\n", *pinned, got)
		} else if got != "" || h.cfg.VersionLabel == "resource-antigravity" {
			fmt.Fprintf(h.cfg.Stdout, "pinned %q matches installed\n", *pinned)
		}
	}
	if h.cfg.DoctorExtra != nil {
		return h.cfg.DoctorExtra(h.cfg.Stdout, h.cfg.Stderr, a)
	}
	return nil
}

func (h *PermissionHandlers) Deny(args []string) error  { return h.mutate("deny", args) }
func (h *PermissionHandlers) Allow(args []string) error { return h.mutate("allow", args) }
func (h *PermissionHandlers) Ask(args []string) error   { return h.mutate("ask", args) }

func (h *PermissionHandlers) mutate(verb string, args []string) error {
	fs, scope := h.scope("permissions " + verb)
	authorized := fs.Bool(OverrideFlag[2:], false, "Override the agent gate (only when a human explicitly authorized this call)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if h.scopeEnabled() {
		scope = fs.Lookup("scope").Value.String()
	}
	if fs.NArg() < 1 {
		if h.scopeEnabled() {
			return fmt.Errorf("usage: permissions %s [--scope %s] <pattern>", verb, strings.TrimPrefix(h.cfg.ScopeHelp, "Config scope: "))
		}
		return fmt.Errorf("usage: permissions %s <pattern>", verb)
	}
	if err := h.Gate("permissions "+verb, true, *authorized); err != nil {
		return err
	}
	a, err := h.adapter(scope)
	if err != nil {
		return err
	}
	p, err := a.Load()
	if err != nil {
		return err
	}
	pattern := fs.Arg(0)
	if h.cfg.ExclusivePatterns {
		p.BashDeny = removePermissionPattern(p.BashDeny, pattern)
		p.BashAsk = removePermissionPattern(p.BashAsk, pattern)
		p.BashAllow = removePermissionPattern(p.BashAllow, pattern)
	}
	switch verb {
	case "deny":
		p.BashDeny = addPermissionPattern(p.BashDeny, pattern)
	case "allow":
		p.BashAllow = addPermissionPattern(p.BashAllow, pattern)
	case "ask":
		p.BashAsk = addPermissionPattern(p.BashAsk, pattern)
	}
	if err := a.Save(p); err != nil {
		return err
	}
	if err := a.WriteState(p, h.cfg.CLIVersion); err != nil {
		return err
	}
	if h.scopeEnabled() {
		fmt.Fprintf(h.cfg.Stdout, "%s %s -> %s (scope=%s)\n", verb, pattern, a.SettingsPath(), a.Scope())
	} else {
		fmt.Fprintf(h.cfg.Stdout, "%s %s -> %s\n", verb, pattern, a.SettingsPath())
	}
	return nil
}

func (h *PermissionHandlers) Remove(args []string) error {
	fs, scope := h.scope("permissions remove")
	authorized := fs.Bool(OverrideFlag[2:], false, "Override the agent gate")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if h.scopeEnabled() {
		scope = fs.Lookup("scope").Value.String()
	}
	if fs.NArg() < 1 {
		return errors.New("usage: permissions remove <pattern>")
	}
	if err := h.Gate("permissions remove", true, *authorized); err != nil {
		return err
	}
	a, err := h.adapter(scope)
	if err != nil {
		return err
	}
	p, err := a.Load()
	if err != nil {
		return err
	}
	pattern := fs.Arg(0)
	p.BashDeny = removePermissionPattern(p.BashDeny, pattern)
	p.BashAsk = removePermissionPattern(p.BashAsk, pattern)
	p.BashAllow = removePermissionPattern(p.BashAllow, pattern)
	if err := a.Save(p); err != nil {
		return err
	}
	if err := a.WriteState(p, h.cfg.CLIVersion); err != nil {
		return err
	}
	fmt.Fprintf(h.cfg.Stdout, "removed %s from any list\n", pattern)
	return nil
}

func (h *PermissionHandlers) Reset(args []string) error {
	fs, scope := h.scope("permissions reset")
	authorized := fs.Bool(OverrideFlag[2:], false, "Override the agent gate")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if h.scopeEnabled() {
		scope = fs.Lookup("scope").Value.String()
	}
	if err := h.Gate("permissions reset", true, *authorized); err != nil {
		return err
	}
	a, err := h.adapter(scope)
	if err != nil {
		return err
	}
	if err := a.Save(h.cfg.ResetPolicy); err != nil {
		return err
	}
	if err := a.WriteState(h.cfg.ResetPolicy, h.cfg.CLIVersion); err != nil {
		return err
	}
	fmt.Fprintln(h.cfg.Stdout, "cleared all Vrooli-managed permission entries")
	return nil
}

// Gate applies the shared agent-vs-human policy to a provider command.
func (h *PermissionHandlers) Gate(verb string, mutating, authorized bool) error {
	cmd := CommandSpec{Name: verb, Mutating: mutating}
	flags := CallerOverrideFlags{AuthorizedByUser: authorized}
	switch Decide(h.cfg.DetectCaller(), cmd, flags, h.cfg.Policy) {
	case DecisionAllow:
		return nil
	case DecisionWarn:
		fmt.Fprintln(h.cfg.Stderr, RenderDenyMessage(DenyContext{ResourceLabel: h.cfg.ResourceLabel, ConfigPath: h.cfg.ConfigHint}, cmd, h.cfg.Policy))
		return nil
	case DecisionDeny:
		return errors.New(RenderDenyMessage(DenyContext{ResourceLabel: h.cfg.ResourceLabel, ConfigPath: h.cfg.ConfigHint}, cmd, h.cfg.Policy))
	default:
		return errors.New("agentharness: unknown decision")
	}
}

func addPermissionPattern(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	result := append(values, value)
	sort.Strings(result)
	return result
}

func removePermissionPattern(values []string, value string) []string {
	result := make([]string, 0, len(values))
	for _, existing := range values {
		if existing != value {
			result = append(result, existing)
		}
	}
	return result
}

func defaultPermissionVersionRunner(ctx context.Context, args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("empty version command")
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}
