package cliapp

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// Command represents a runnable CLI command.
//
// Two handler shapes are supported:
//   - Run: the existing func([]string) error path. Used by commands that do
//     their own arg parsing (typically with stdlib flag.NewFlagSet).
//   - RunCtx + Args: the declarative path. The Args ArgSchema feeds both the
//     parser and helpgen; the resulting RunContext is passed to RunCtx.
//
// When both are set, RunCtx wins. When neither is set, the dispatcher returns
// an error. New scenarios should prefer RunCtx + Args.
type Command struct {
	Name            string
	Aliases         []string
	Description     string
	Usage           string
	HelpText        string
	LongDescription string
	NeedsAPI        bool
	Args            ArgSchema
	Run             func(args []string) error
	RunCtx          func(ctx RunContext) error
	// Architecture declares the command's renderer-separated primitive class
	// (or an explicit exception class for legitimate special cases). It is
	// optional and additive: a zero value means "unclassified/legacy" and
	// carries no behavioral effect. It is the machine-recognizable evidence
	// cli-health classifies command architecture maturity from, mirrored at
	// runtime from the manifest's architecture block (see LoadFromManifest).
	Architecture CommandArchitecture
	// primitiveEvidence is the cli-core primitive class that ACTUALLY built the
	// handler, stamped by construction by the primitive builders (see
	// PrimitiveHandler / LoadFromManifestPrimitives / WithPrimitive). An empty
	// value means no machine-verifiable evidence — a legacy or hand-rolled
	// handler. Unlike Architecture (which a manifest can declare freely),
	// primitiveEvidence is UNEXPORTED: only a cli-core builder in this package can
	// set it, so scenario code cannot forge verified maturity through a struct
	// literal or field assignment (plan decision D3). CLI Health treats a
	// declaration that matches the observed evidence as verified rather than
	// self-certified. Read it via PrimitiveEvidence(); see ClassifyPrimitiveEvidence.
	primitiveEvidence PrimitiveClass
}

// PrimitiveEvidence returns the cli-core primitive class the command's handler
// was built from (empty when the handler carries no evidence). Read-only: the
// evidence can only be stamped by a cli-core primitive builder via WithPrimitive,
// WithLegacyPrimitive, or LoadFromManifestPrimitives.
func (c Command) PrimitiveEvidence() PrimitiveClass { return c.primitiveEvidence }

// WithPrimitive wires a PrimitiveHandler onto the command: it sets RunCtx to the
// handler closure and records the primitive class as observed implementation
// evidence. Use it for non-manifest RunCtx commands so the evidence travels with
// the command rather than being restated by scenario code.
func (c Command) WithPrimitive(ph PrimitiveHandler) Command {
	c.RunCtx = ph.Run
	c.primitiveEvidence = ph.primitive
	return c
}

// CommandGroup bundles related commands for help output.
type CommandGroup struct {
	Title    string
	Commands []Command
}

// SubcommandGroup provides hierarchical command support (e.g., "pipeline run", "pipeline status").
// Each group automatically gets a "help" subcommand.
type SubcommandGroup struct {
	// Name is the group command (e.g., "pipeline", "telemetry")
	Name string
	// Description shown in main help
	Description string
	// Subcommands within this group (e.g., "run", "status")
	Subcommands []Command
	// NeedsAPI applies to all subcommands unless overridden
	NeedsAPI bool
	// DefaultSubcommand, when non-empty, names the subcommand to invoke when
	// args[0] is not a known subcommand and not a help token. Lets a group
	// accept `<group> <free-form args>` as shorthand for
	// `<group> <default> <free-form args>` — useful when one subcommand is so
	// dominant that requiring it harms ergonomics (e.g. search).
	DefaultSubcommand string
}

// AppOptions configure a CLI application with common behaviors.
type AppOptions struct {
	Name             string
	Version          string
	Description      string
	Commands         []CommandGroup
	SubcommandGroups []SubcommandGroup
	APIOverride      *string
	ColorEnabled     bool
	OnColor          func(enabled bool)
	StaleChecker     *cliutil.StaleChecker
	Preflight        func(cmd Command, global GlobalOptions) error
	// UnknownCommandHint, when set, returns extra recovery text for known bad
	// command shapes. It is advisory only; the dispatcher still returns an
	// unknown-command error and never executes a replacement command.
	UnknownCommandHint func(args []string) string
}

// GlobalOptions holds parsed global flags that all scenario CLIs share.
type GlobalOptions struct {
	APIBaseOverride string
	ColorEnabled    bool
	AutoStart       bool
	DryRun          bool
	// Instance selects which variant of the CLI's own scenario its API calls
	// target (e.g. "shadow"). Empty means "use the default routing" (ambient
	// VROOLI_SHADOW_SCENARIOS, else live). An explicit "live" forces live even
	// when the scenario is ambiently shadowed.
	Instance string
}

// DefaultColorEnabled derives the default color setting from NO_COLOR.
func DefaultColorEnabled() bool {
	return os.Getenv("NO_COLOR") == ""
}

// App coordinates command dispatch, global flags, help/version, and stale checks.
type App struct {
	opts                  AppOptions
	global                GlobalOptions
	commands              []Command
	commandLookup         map[string]Command
	subcommandGroupLookup map[string]*SubcommandGroup
	scenario              *ScenarioApp
}

// AttachScenario records the ScenarioApp so RunCtx-style handlers can call
// RunContext.Core() to reach the API client. Called by ScenarioApp.SetCommandsWithSubgroups.
func (a *App) AttachScenario(s *ScenarioApp) {
	a.scenario = s
}

// NewApp builds an App with meta commands (help/version) included automatically.
func NewApp(opts AppOptions) *App {
	app := &App{
		opts: opts,
		global: GlobalOptions{
			ColorEnabled: opts.ColorEnabled,
		},
	}
	app.buildCommands()
	app.applyColor()
	return app
}

// Run parses global flags, routes to a command, and triggers stale checks when needed.
func (a *App) Run(args []string) error {
	if len(args) == 0 {
		a.PrintHelp()
		return nil
	}

	remaining, err := ParseGlobalFlags(args, &a.global, a.opts.APIOverride)
	if err != nil {
		return err
	}
	a.applyColor()
	// Scope the --instance selection to this CLI's own scenario so its API-base
	// port detector (Case A) routes to the chosen variant, without affecting how
	// it resolves unrelated targets.
	cliutil.SetInstanceOverride(a.opts.Name, a.global.Instance)

	if len(remaining) == 0 {
		a.PrintHelp()
		return nil
	}

	// Check for subcommand group first
	if group, ok := a.subcommandGroupLookup[remaining[0]]; ok {
		return a.runSubcommand(group, remaining[1:], args)
	}

	cmd, ok := a.commandLookup[remaining[0]]
	if !ok {
		return fmt.Errorf("Unknown command: %s%s%s", remaining[0], a.suggestCommand(remaining[0]), a.unknownCommandHint(remaining))
	}
	if wantsHelp(remaining[1:]) {
		a.printCommandHelp(a.opts.Name, cmd)
		return nil
	}

	if cmd.NeedsAPI && a.opts.StaleChecker != nil {
		a.opts.StaleChecker.ReexecArgs = args
		if restarted := a.opts.StaleChecker.CheckAndMaybeRebuild(); restarted {
			return nil
		}
	}

	if a.opts.Preflight != nil {
		if err := a.opts.Preflight(cmd, a.global); err != nil {
			return err
		}
	}

	return a.dispatchCommand(a.opts.Name, cmd, remaining[1:])
}

// runSubcommand handles dispatch within a subcommand group.
func (a *App) runSubcommand(group *SubcommandGroup, args []string, originalArgs []string) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		a.printSubcommandHelp(group)
		return nil
	}

	// Find the subcommand
	var cmd *Command
	for i := range group.Subcommands {
		if group.Subcommands[i].Name == args[0] {
			cmd = &group.Subcommands[i]
			break
		}
		for _, alias := range group.Subcommands[i].Aliases {
			if alias == args[0] {
				cmd = &group.Subcommands[i]
				break
			}
		}
		if cmd != nil {
			break
		}
	}

	if cmd == nil && group.DefaultSubcommand != "" {
		for i := range group.Subcommands {
			if group.Subcommands[i].Name == group.DefaultSubcommand {
				cmd = &group.Subcommands[i]
				break
			}
		}
		if cmd == nil {
			return fmt.Errorf("subcommand group %q declares DefaultSubcommand %q but no such subcommand exists", group.Name, group.DefaultSubcommand)
		}
		return a.dispatchCommand(strings.TrimSpace(a.opts.Name+" "+group.Name), *cmd, args)
	}
	if cmd == nil {
		path := append([]string{group.Name}, args...)
		return fmt.Errorf("Unknown subcommand: %s %s%s%s\nRun '%s %s help' for available subcommands",
			group.Name, args[0], suggestSubcommand(group, args[0]), a.unknownCommandHint(path), a.opts.Name, group.Name)
	}
	if wantsHelp(args[1:]) {
		a.printCommandHelp(strings.TrimSpace(a.opts.Name+" "+group.Name), *cmd)
		return nil
	}

	needsAPI := cmd.NeedsAPI || group.NeedsAPI
	if needsAPI && a.opts.StaleChecker != nil {
		a.opts.StaleChecker.ReexecArgs = originalArgs
		if restarted := a.opts.StaleChecker.CheckAndMaybeRebuild(); restarted {
			return nil
		}
	}

	if a.opts.Preflight != nil {
		preflightCmd := *cmd
		preflightCmd.Name = strings.TrimSpace(group.Name + " " + cmd.Name)
		preflightCmd.NeedsAPI = needsAPI
		if err := a.opts.Preflight(preflightCmd, a.global); err != nil {
			return err
		}
	}

	return a.dispatchCommand(strings.TrimSpace(a.opts.Name+" "+group.Name), *cmd, args[1:])
}

// dispatchCommand routes a Command's execution through either the declarative
// RunCtx path (when Args/RunCtx are set) or the legacy Run path. ErrHelpRequested
// from the parser is caught here and converted to a help print with nil error.
func (a *App) dispatchCommand(prefix string, cmd Command, cmdArgs []string) error {
	if cmd.RunCtx != nil {
		ctx, err := parseArgs(cmd.Args, cmdArgs, a.scenario, os.Stdout, os.Stderr)
		if err != nil {
			if errors.Is(err, ErrHelpRequested) {
				a.printCommandHelp(prefix, cmd)
				return nil
			}
			return err
		}
		return cmd.RunCtx(ctx)
	}
	if cmd.Run != nil {
		return cmd.Run(cmdArgs)
	}
	return fmt.Errorf("command %q has no Run or RunCtx handler", cmd.Name)
}

// printSubcommandHelp prints help for a subcommand group.
func (a *App) printSubcommandHelp(group *SubcommandGroup) {
	fmt.Printf("%s %s - %s\n\n", a.opts.Name, group.Name, group.Description)
	fmt.Printf("Usage:\n  %s %s <subcommand> [options]\n\n", a.opts.Name, group.Name)
	fmt.Println("Subcommands:")
	for _, cmd := range group.Subcommands {
		fmt.Printf("  %-20s %s\n", cmd.Name, cmd.Description)
	}
	fmt.Println()
	fmt.Printf("Run '%s %s <subcommand> --help' for subcommand-specific options.\n", a.opts.Name, group.Name)
}

func (a *App) printCommandHelp(prefix string, cmd Command) {
	if cmd.RunCtx != nil {
		_ = renderHelp(prefix, cmd, os.Stdout)
		return
	}

	fullName := strings.TrimSpace(prefix + " " + cmd.Name)
	title := strings.TrimSpace(fullName)
	if cmd.Description != "" {
		fmt.Printf("%s - %s\n\n", title, cmd.Description)
	} else {
		fmt.Printf("%s\n\n", title)
	}

	usage := strings.TrimSpace(cmd.Usage)
	if usage == "" {
		usage = fullName
	}
	fmt.Printf("Usage:\n  %s\n", usage)

	if len(cmd.Aliases) > 0 {
		fmt.Printf("\nAliases:\n  %s\n", strings.Join(cmd.Aliases, ", "))
	}

	if helpText := strings.TrimSpace(cmd.HelpText); helpText != "" {
		fmt.Printf("\n%s\n", helpText)
	}
}

// SetStaleChecker overrides the stale checker (useful in tests).
func (a *App) SetStaleChecker(checker *cliutil.StaleChecker) {
	a.opts.StaleChecker = checker
}

// PrintHelp renders grouped command help plus global options.
func (a *App) PrintHelp() {
	fmt.Printf("%s CLI\n\n", a.opts.Name)
	fmt.Printf("Usage:\n  %s [global options] <command> [options]\n\n", a.opts.Name)

	fmt.Print("Global Options (must be placed BEFORE the command):\n")
	fmt.Println("  --api-base <url>   Override API base URL (default: auto-detected)")
	fmt.Println("  --instance <name>  Target a scenario variant (e.g. shadow); default: live")
	fmt.Println("  --auto-start       Auto-start the scenario if not running")
	fmt.Println("  --dry-run          Validate without executing mutations")
	fmt.Println("  --no-color         Disable ANSI color output (or set NO_COLOR)")
	fmt.Println("  --color            Force-enable ANSI color output")
	fmt.Println()

	// Print subcommand groups first (these are the main features)
	if len(a.opts.SubcommandGroups) > 0 {
		fmt.Println("Command Groups (run '<group> help' for details):")
		for _, group := range a.opts.SubcommandGroups {
			fmt.Printf("  %-20s %s\n", group.Name, group.Description)
		}
		fmt.Println()
	}

	fmt.Println("Commands:")
	for _, group := range a.commandGroups() {
		fmt.Printf("  %s\n", group.Title)
		for _, cmd := range group.Commands {
			fmt.Printf("    %-28s %s\n", cmd.Name, cmd.Description)
		}
		fmt.Println()
	}
}

func (a *App) commandGroups() []CommandGroup {
	return a.opts.Commands
}

func (a *App) buildCommands() {
	var groups []CommandGroup

	// Meta commands first so help/version are always present.
	meta := CommandGroup{
		Title: "Meta",
		Commands: []Command{
			{
				Name:        "help",
				Aliases:     []string{"--help", "-h"},
				Description: "Show this help message",
				Run: func(args []string) error {
					a.PrintHelp()
					return nil
				},
			},
		},
	}
	if strings.TrimSpace(a.opts.Version) != "" {
		meta.Commands = append(meta.Commands, Command{
			Name:        "version",
			Aliases:     []string{"--version", "-v"},
			Description: "Show CLI version",
			Run: func(args []string) error {
				fmt.Printf("%s CLI version %s\n", a.opts.Name, a.opts.Version)
				return nil
			},
		})
	}
	groups = append(groups, meta)
	groups = append(groups, a.opts.Commands...)
	a.opts.Commands = groups

	a.commandLookup = make(map[string]Command)
	for _, group := range groups {
		for _, cmd := range group.Commands {
			a.commands = append(a.commands, cmd)
			a.commandLookup[cmd.Name] = cmd
			for _, alias := range cmd.Aliases {
				a.commandLookup[alias] = cmd
			}
		}
	}

	// Build subcommand group lookup
	a.subcommandGroupLookup = make(map[string]*SubcommandGroup)
	for i := range a.opts.SubcommandGroups {
		group := &a.opts.SubcommandGroups[i]
		a.subcommandGroupLookup[group.Name] = group
	}
}

func (a *App) applyColor() {
	if a.opts.OnColor != nil {
		a.opts.OnColor(a.global.ColorEnabled)
	}
}

// isGlobalFlagName reports whether name (without leading dashes) is a shared
// global flag. Global flags are parsed only before the subcommand; when one is
// misplaced after it, the subcommand parser uses this to emit a placement hint
// instead of a bare "unknown option".
func isGlobalFlagName(name string) bool {
	switch name {
	case "api-base", "instance", "auto-start", "dry-run", "no-color", "color":
		return true
	default:
		return false
	}
}

// ParseGlobalFlags extracts shared flags from args, updating global options and an optional API override target.
func ParseGlobalFlags(args []string, global *GlobalOptions, apiOverrideTarget *string) ([]string, error) {
	if global == nil {
		return args, nil
	}

	remaining := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			remaining = append(remaining, args[i:]...)
			break
		}
		switch args[i] {
		case "--api-base":
			if i+1 >= len(args) {
				return nil, errors.New("missing value for --api-base")
			}
			global.APIBaseOverride = args[i+1]
			if apiOverrideTarget != nil {
				*apiOverrideTarget = args[i+1]
			}
			i++
		case "--instance":
			if i+1 >= len(args) {
				return nil, errors.New("missing value for --instance")
			}
			global.Instance = args[i+1]
			i++
		case "--auto-start":
			global.AutoStart = true
		case "--dry-run":
			global.DryRun = true
		case "--no-color":
			global.ColorEnabled = false
		case "--color":
			global.ColorEnabled = true
		default:
			remaining = append(remaining, args[i:]...)
			return remaining, nil
		}
	}
	return remaining, nil
}

// suggestCommand returns a ` (did you mean "x"?)` fragment naming the nearest
// command or subcommand group, or "" when nothing is close. Covers the common
// singular/plural miss (e.g. `record create` → `records create`) so agents get
// a one-retry correction instead of a dead end.
func (a *App) suggestCommand(candidate string) string {
	options := make([]string, 0, len(a.commandLookup)+len(a.subcommandGroupLookup))
	for name := range a.commandLookup {
		if !strings.HasPrefix(name, "-") { // skip --help/-v style aliases
			options = append(options, name)
		}
	}
	for name := range a.subcommandGroupLookup {
		options = append(options, name)
	}
	if nearest := cliutil.NearestString(candidate, options, 2); nearest != "" {
		return fmt.Sprintf(" (did you mean %q?)", nearest)
	}
	return ""
}

func (a *App) unknownCommandHint(args []string) string {
	if a.opts.UnknownCommandHint == nil {
		return ""
	}
	hint := strings.TrimSpace(a.opts.UnknownCommandHint(append([]string(nil), args...)))
	if hint == "" {
		return ""
	}
	return "\n\n" + hint
}

// suggestSubcommand is suggestCommand scoped to one group's subcommands.
func suggestSubcommand(group *SubcommandGroup, candidate string) string {
	options := make([]string, 0, len(group.Subcommands))
	for _, sub := range group.Subcommands {
		options = append(options, sub.Name)
		options = append(options, sub.Aliases...)
	}
	if nearest := cliutil.NearestString(candidate, options, 2); nearest != "" {
		return fmt.Sprintf(" (did you mean %q?)", nearest)
	}
	return ""
}

func isHelpToken(arg string) bool {
	switch strings.TrimSpace(arg) {
	case "help", "-h", "--help":
		return true
	default:
		return false
	}
}

func wantsHelp(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if isHelpToken(arg) {
			return true
		}
	}
	return false
}

// SortedCommands returns commands ordered by name (useful for tests).
func SortedCommands(cmds []Command) []Command {
	result := append([]Command(nil), cmds...)
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}
