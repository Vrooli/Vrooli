package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/shell"
)

const (
	vrooliVersion = "2.0.0"
	cliVersion    = "1.0.0"
)

var (
	resolveSourceRootFn = buildinfo.ResolveSourceRoot
	isStaleFn           = buildinfo.IsStale
	checkStalenessFn    = buildinfo.CheckStaleness
	rebuildAndReexecFn  = buildinfo.RebuildAndReexec
	lookPathFn          = shell.LookPath
	newLoggerFn         = createCommandLogger
)

const (
	errorCategoryUsage       = "Usage"
	errorCategoryEnvironment = "Environment"
	errorCategoryRuntime     = "Runtime"
)

type globalOptions struct {
	json         bool
	verbose      bool
	noColor      bool
	noStaleCheck bool
}

func (g globalOptions) logFormat() logx.Format {
	if g.json {
		return logx.FormatJSON
	}
	return ""
}

type parsedArgs struct {
	command string
	args    []string
	globals globalOptions
}

type versionOutput struct {
	CLIVersion      string `json:"cli_version"`
	PlatformVersion string `json:"platform_version"`
	Root            string `json:"root"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return configuredApp().Run(args, stdout, stderr)
}

func primeRootEnv(root string) {
	_ = os.Setenv("VROOLI_ROOT", root)
	if strings.TrimSpace(os.Getenv(buildinfo.SourceRootEnvVar)) == "" {
		_ = os.Setenv(buildinfo.SourceRootEnvVar, root)
	}
}

func parseArgs(args []string) (parsedArgs, error) {
	parsed := parsedArgs{}
	for len(args) > 0 {
		switch args[0] {
		case "--json":
			parsed.globals.json = true
			args = args[1:]
		case "--verbose":
			parsed.globals.verbose = true
			args = args[1:]
		case "--no-color":
			parsed.globals.noColor = true
			args = args[1:]
		case "--no-stale-check":
			parsed.globals.noStaleCheck = true
			args = args[1:]
		case "--help", "-h":
			return parsedArgs{command: "help", globals: parsed.globals}, nil
		case "--version", "-v":
			return parsedArgs{command: "version", globals: parsed.globals}, nil
		default:
			parsed.command = args[0]
			parsed.args = append([]string(nil), args[1:]...)
			return parsed, nil
		}
	}
	return parsedArgs{command: "help", globals: parsed.globals}, nil
}

func consumeInlineGlobalFlags(globals globalOptions, args []string) (globalOptions, []string) {
	filtered := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--":
			filtered = append(filtered, args[index:]...)
			return globals, filtered
		case "--json":
			globals.json = true
		case "--verbose":
			globals.verbose = true
		case "--no-color":
			globals.noColor = true
		case "--no-stale-check":
			globals.noStaleCheck = true
		default:
			filtered = append(filtered, args[index])
		}
	}
	return globals, filtered
}

func dispatch(app *App, ctx *commandContext, parsed parsedArgs) error {
	switch parsed.command {
	case "help":
		return app.runMainHelpCommand(ctx, parsed.args)
	case "version":
		return app.runVersionCommand(ctx, parsed.args)
	}
	handler, ok := topLevelCommands[commandtree.NormalizeName(parsed.command)]
	if !ok {
		return newUnknownCommandError(parsed.command)
	}
	return handler(app, ctx, parsed.args)
}

func (app *App) runMainHelpCommand(ctx *commandContext, args []string) error {
	showMainHelp(ctx.Stdout)
	return nil
}

func (app *App) runVersionCommand(ctx *commandContext, args []string) error {
	return showVersion(ctx.Stdout, ctx.Root, ctx.Globals)
}

func passthroughFlags(globals globalOptions, existing []string) []string {
	flags := make([]string, 0, 3)
	if globals.json && !containsArg(existing, "--json") {
		flags = append(flags, "--json")
	}
	if globals.verbose && !containsArg(existing, "--verbose") {
		flags = append(flags, "--verbose")
	}
	if globals.noColor && !containsArg(existing, "--no-color") {
		flags = append(flags, "--no-color")
	}
	return flags
}

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func setEnvValue(env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			updated := append([]string(nil), env...)
			updated[i] = prefix + value
			return updated
		}
	}
	return append(append([]string(nil), env...), prefix+value)
}

func showVersion(w io.Writer, root string, globals globalOptions) error {
	format, err := formatFromJSON(globals.json)
	if err != nil {
		return err
	}
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, versionOutput{
			CLIVersion:      cliVersion,
			PlatformVersion: vrooliVersion,
			Root:            root,
		})
	}
	_, _ = fmt.Fprintf(w, "Vrooli CLI v%s\n", cliVersion)
	_, _ = fmt.Fprintf(w, "Vrooli Platform v%s\n", vrooliVersion)
	_, _ = fmt.Fprintf(w, "Root: %s\n", root)
	return nil
}
