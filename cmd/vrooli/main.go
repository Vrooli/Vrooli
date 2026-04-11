package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/logx"
)

const (
	vrooliVersion = "2.0.0"
	cliVersion    = "1.0.0"
)

var (
	resolveSourceRootFn = buildinfo.ResolveSourceRoot
	isStaleFn           = buildinfo.IsStale
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

type topLevelCommandHandler func(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error

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
	parsed, err := parseArgs(args)
	if err != nil {
		printErrorWithContext(stderr, newErrorWithCategory(err, errorCategoryUsage, "Use --help for available commands", nil))
		return 1
	}
	parsed.globals, parsed.args = consumeInlineGlobalFlags(parsed.globals, parsed.args)
	logger := newLoggerFn(parsed.globals.verbose, stderr)
	slog.SetDefault(logger)
	logx.RedirectStandardLibrary(logger)
	debugLog(logger, "Parsed command", "command", parsed.command, "args", parsed.args, "json", parsed.globals.json, "verbose", parsed.globals.verbose)

	root, err := resolveRoot()
	if err != nil {
		printErrorWithContext(stderr, newErrorWithCategory(err, errorCategoryEnvironment, "Run from a Vrooli repository root or set VROOLI_SOURCE_ROOT", nil))
		return 1
	}
	debugLog(logger, "Resolved root", "path", root)
	primeRootEnv(root)

	if parsed.globals.noColor {
		_ = os.Setenv("NO_COLOR", "1")
	}
	if parsed.globals.noColor {
		debugLog(logger, "NO_COLOR requested by user flags")
	}

	if !parsed.globals.noStaleCheck && isStaleFn() {
		debugLog(logger, "Stale check triggered")
		if err := rebuildAndReexecFn(args); err != nil {
			printErrorWithContext(stderr, newErrorWithCategory(
				fmt.Errorf("stale binary check failed: %w", err),
				errorCategoryRuntime,
				"Use --no-stale-check for local experiments",
				nil,
			))
			return 1
		}
		debugLog(logger, "Rebuilt command binary and re-executed")
		return 0
	}

	if err := dispatch(root, parsed, stdout, stderr); err != nil {
		printErrorWithContext(stderr, err)
		return exitCode(err)
	}
	return 0
}

func resolveRoot() (string, error) {
	root, err := resolveSourceRootFn()
	if err != nil {
		return "", fmt.Errorf("resolve Vrooli root: %w", err)
	}
	return filepath.Clean(root), nil
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

func dispatch(root string, parsed parsedArgs, stdout, stderr io.Writer) error {
	switch parsed.command {
	case "help":
		return runMainHelpCommand(root, parsed.globals, parsed.args, stdout, stderr)
	case "version":
		return runVersionCommand(root, parsed.globals, parsed.args, stdout, stderr)
	}
	handler, ok := topLevelCommands[parsed.command]
	if !ok {
		return newUnknownCommandError(parsed.command)
	}
	return handler(root, parsed.globals, parsed.args, stdout, stderr)
}

func runMainHelpCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	showMainHelp(stdout)
	return nil
}

func runVersionCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	return showVersion(stdout, root, globals)
}

func runTopLevelCleanupCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	return runCleanupCommand(root, parsedArgs{globals: globals, args: args}, stdout, stderr)
}

func runTopLevelSetupCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	return runProjectSetupCommand(root, args, stdout, stderr)
}

func runTopLevelDevelopCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	return runProjectDevelopCommand(root, args, stdout, stderr)
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
	env := os.Environ()
	env = setEnvValue(env, "VROOLI_ROOT", root)
	if strings.TrimSpace(os.Getenv(buildinfo.SourceRootEnvVar)) == "" {
		env = setEnvValue(env, buildinfo.SourceRootEnvVar, root)
	}
	if globals.noColor {
		env = setEnvValue(env, "NO_COLOR", "1")
	}
	return env
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
