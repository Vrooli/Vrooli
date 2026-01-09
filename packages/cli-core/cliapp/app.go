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
type Command struct {
	Name        string
	Aliases     []string
	Description string
	NeedsAPI    bool
	Run         func(args []string) error
}

// CommandGroup bundles related commands for help output.
type CommandGroup struct {
	Title    string
	Commands []Command
}

// AppOptions configure a CLI application with common behaviors.
type AppOptions struct {
	Name         string
	Version      string
	Description  string
	Commands     []CommandGroup
	APIOverride  *string
	ColorEnabled bool
	OnColor      func(enabled bool)
	StaleChecker *cliutil.StaleChecker
	Preflight    func(cmd Command, global GlobalOptions) error
}

// GlobalOptions holds parsed global flags that all scenario CLIs share.
type GlobalOptions struct {
	APIBaseOverride string
	ColorEnabled    bool
	AutoStart       bool
}

// DefaultColorEnabled derives the default color setting from NO_COLOR.
func DefaultColorEnabled() bool {
	return os.Getenv("NO_COLOR") == ""
}

// App coordinates command dispatch, global flags, help/version, and stale checks.
type App struct {
	opts          AppOptions
	global        GlobalOptions
	commands      []Command
	commandLookup map[string]Command
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

	if len(remaining) == 0 {
		a.PrintHelp()
		return nil
	}

	cmd, ok := a.commandLookup[remaining[0]]
	if !ok {
		return fmt.Errorf("Unknown command: %s", remaining[0])
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

	return cmd.Run(remaining[1:])
}

// SetStaleChecker overrides the stale checker (useful in tests).
func (a *App) SetStaleChecker(checker *cliutil.StaleChecker) {
	a.opts.StaleChecker = checker
}

// PrintHelp renders grouped command help plus global options.
func (a *App) PrintHelp() {
	fmt.Printf("%s CLI\n\n", a.opts.Name)
	fmt.Printf("Usage:\n  %s <command> [options]\n\n", a.opts.Name)

	fmt.Print("Global Options:\n")
	fmt.Println("  --api-base <url>   Override API base URL (default: auto-detected)")
	fmt.Println("  --auto-start       Auto-start the scenario if not running")
	fmt.Println("  --no-color         Disable ANSI color output (or set NO_COLOR)")
	fmt.Println("  --color            Force-enable ANSI color output")
	fmt.Println()

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
}

func (a *App) applyColor() {
	if a.opts.OnColor != nil {
		a.opts.OnColor(a.global.ColorEnabled)
	}
}

// ParseGlobalFlags extracts shared flags from args, updating global options and an optional API override target.
func ParseGlobalFlags(args []string, global *GlobalOptions, apiOverrideTarget *string) ([]string, error) {
	if global == nil {
		return args, nil
	}

	remaining := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
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
		case "--auto-start":
			global.AutoStart = true
		case "--no-color":
			global.ColorEnabled = false
		case "--color":
			global.ColorEnabled = true
		default:
			remaining = append(remaining, args[i])
		}
	}
	return remaining, nil
}

// SortedCommands returns commands ordered by name (useful for tests).
func SortedCommands(cmds []Command) []Command {
	result := append([]Command(nil), cmds...)
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}
