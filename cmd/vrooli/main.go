package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cliout"
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
	lookPathFn          = exec.LookPath
	execCommandFn       = runExternalCommand
	newLoggerFn         = createCommandLogger
)

var infoDefaultFiles = []string{"docs/context.md"}

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

type parsedArgs struct {
	command string
	args    []string
	globals globalOptions
}

type commandSpec struct {
	name string
	args []string
	dir  string
	env  []string
}

type infoManifest struct {
	Files []string `json:"files"`
}

type versionOutput struct {
	CLIVersion      string `json:"cli_version"`
	PlatformVersion string `json:"platform_version"`
	Root            string `json:"root"`
}

type infoFileOutput struct {
	Path     string `json:"path"`
	Contents string `json:"contents,omitempty"`
}

type infoOutput struct {
	Root  string           `json:"root"`
	Files []infoFileOutput `json:"files"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return configuredApp().Run(args, stdout, stderr)
}

func resolveRoot() (string, error) {
	return configuredApp().resolveRoot()
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
	handler, ok := topLevelCommands[parsed.command]
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

func (app *App) runTopLevelCleanupCommand(ctx *commandContext, args []string) error {
	return runCleanupCommand(ctx.Root, parsedArgs{globals: ctx.Globals, args: args}, ctx.Stdout, ctx.Stderr)
}

func (app *App) runTopLevelSetupCommand(ctx *commandContext, args []string) error {
	return app.runTopLevelSetup(ctx, args)
}

func (app *App) runTopLevelDevelopCommand(ctx *commandContext, args []string) error {
	return app.runTopLevelDevelop(ctx, args)
}

func runCleanupCommand(root string, parsed parsedArgs, stdout, stderr io.Writer) error {
	if len(parsed.args) == 0 {
		showCleanupHelp(stdout)
		return nil
	}

	target := parsed.args[0]
	rest := parsed.args[1:]
	switch target {
	case "orphans":
		return runTopLevelOrphansCommand(root, parsed.globals, append([]string{"kill"}, rest...), stdout, stderr)
	case "locks":
		return runTopLevelLocksCommand(root, parsed.globals, append([]string{"clean"}, rest...), stdout, stderr)
	case "help", "--help", "-h":
		showCleanupHelp(stdout)
		return nil
	default:
		_, _ = fmt.Fprintf(stderr, "Unknown cleanup target: %s\n", target)
		_, _ = fmt.Fprintln(stderr, "Run 'vrooli cleanup help' for usage")
		return exitCodeError{code: 1, message: "unknown cleanup target"}
	}
}

func runInfoCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	showPathsOnly := false
	for _, arg := range args {
		switch arg {
		case "--list":
			showPathsOnly = true
		case "--help", "-h":
			_, _ = fmt.Fprintln(stdout, "Usage: vrooli info [--list]")
			_, _ = fmt.Fprintln(stdout)
			_, _ = fmt.Fprintln(stdout, "Display consolidated Vrooli project context in a single stream.")
			_, _ = fmt.Fprintln(stdout)
			_, _ = fmt.Fprintln(stdout, "    --list     Print the resolved file paths without emitting file contents.")
			return nil
		default:
			return fmt.Errorf("unknown option for info: %s", arg)
		}
	}

	format, err := cliout.ParseFormat("", globals.json)
	if err != nil {
		return err
	}

	infoFiles, err := collectInfoSources(root)
	if err != nil {
		return err
	}
	if len(infoFiles) == 0 {
		return errors.New("no context sources defined for vrooli info")
	}

	if showPathsOnly {
		paths := make([]string, 0, len(infoFiles))
		for _, file := range infoFiles {
			paths = append(paths, resolveInfoPath(root, file))
		}
		if format == cliout.FormatJSON {
			return cliout.WriteJSON(stdout, map[string]any{
				"root":  root,
				"files": paths,
			})
		}
		for _, path := range paths {
			_, _ = fmt.Fprintln(stdout, path)
		}
		return nil
	}

	if format == cliout.FormatJSON {
		payload := infoOutput{
			Root:  root,
			Files: make([]infoFileOutput, 0, len(infoFiles)),
		}
		for _, source := range infoFiles {
			resolved := resolveInfoPath(root, source)
			contents, readErr := os.ReadFile(resolved)
			if readErr != nil {
				if os.IsNotExist(readErr) {
					_, _ = fmt.Fprintf(stderr, "[WARNING] Skipping missing context file: %s\n", source)
					continue
				}
				return fmt.Errorf("read info source %s: %w", resolved, readErr)
			}
			payload.Files = append(payload.Files, infoFileOutput{
				Path:     resolved,
				Contents: string(contents),
			})
		}
		return cliout.WriteJSON(stdout, payload)
	}

	_, _ = fmt.Fprintln(stdout, "[HEADER]  Vrooli Context Briefing")
	_, _ = fmt.Fprintf(stdout, "[INFO]    Project root: %s\n", root)
	for _, source := range infoFiles {
		resolved := resolveInfoPath(root, source)
		contents, readErr := os.ReadFile(resolved)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				_, _ = fmt.Fprintf(stderr, "[WARNING] Skipping missing context file: %s\n", source)
				continue
			}
			return fmt.Errorf("read info source %s: %w", resolved, readErr)
		}
		_, _ = fmt.Fprintf(stdout, "\n===== %s =====\n", resolved)
		_, _ = stdout.Write(contents)
		if len(contents) == 0 || contents[len(contents)-1] != '\n' {
			_, _ = fmt.Fprintln(stdout)
		}
	}

	return nil
}

func collectInfoSources(root string) ([]string, error) {
	if envValue := strings.TrimSpace(os.Getenv("VROOLI_INFO_FILES")); envValue != "" {
		parts := strings.Split(envValue, ":")
		files := make([]string, 0, len(parts))
		for _, entry := range parts {
			entry = strings.TrimSpace(entry)
			if entry != "" {
				files = append(files, entry)
			}
		}
		return files, nil
	}

	manifestPath := filepath.Join(root, ".vrooli", "info-manifest.json")
	if data, err := os.ReadFile(manifestPath); err == nil {
		var manifest infoManifest
		if err := json.Unmarshal(data, &manifest); err == nil && len(manifest.Files) > 0 {
			return manifest.Files, nil
		}
	}

	return append([]string(nil), infoDefaultFiles...), nil
}

func resolveInfoPath(root, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(root, filepath.FromSlash(path))
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

func commandEnv(root string, globals globalOptions) []string {
	return configuredApp().commandEnv(root, globals)
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
	format, err := cliout.ParseFormat("", globals.json)
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
