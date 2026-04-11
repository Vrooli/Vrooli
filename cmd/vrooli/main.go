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
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/logx"
)

const (
	vrooliVersion   = "2.0.0"
	cliVersion      = "1.0.0"
	forceBashEnvVar = "VROOLI_FORCE_BASH"
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

var topLevelCommands = map[string]topLevelCommandHandler{
	"help":          runMainHelpCommand,
	"version":       runVersionCommand,
	"info":          runInfoCommand,
	"cleanup":       runTopLevelCleanupCommand,
	"orphans":       makeTopLevelAutohealHandler("orphans"),
	"locks":         makeTopLevelAutohealHandler("locks"),
	"diagnose-port": makeTopLevelAutohealHandler("diagnose-port"),
	"setup":         runTopLevelSetupCommand,
	"develop":       runTopLevelDevelopCommand,
	"build":         runTopLevelBuildCommand,
	"deploy":        runTopLevelDeployCommand,
	"backup":        runTopLevelBackupCommand,
	"restore":       runTopLevelRestoreCommand,
	"clean":         runTopLevelCleanupCommand,
	"status":        runTopLevelStatusCommand,
	"scenario":      runScenarioCommand,
	"resource":      runTopLevelResourceCommand,
	"stop":          runTopLevelStopCommand,
	"doctor":        runTopLevelDoctorCommand,
}

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

	if forceBashEnabled() {
		debugLog(logger, "Legacy Bash mode enabled", "command", parsed.command)
		return exitCode(runLegacyBash(root, args))
	}
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
				"Use --no-stale-check for local experiments, or VROOLI_FORCE_BASH=1 to bypass this path",
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

func forceBashEnabled() bool {
	return strings.TrimSpace(os.Getenv(forceBashEnvVar)) == "1"
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
	handler, ok := topLevelCommands[parsed.command]
	if !ok {
		return newUnknownCommandError(parsed.command)
	}
	return handler(root, parsed.globals, parsed.args, stdout, stderr)
}

func makeTopLevelScriptHandler(scriptPath, command string) topLevelCommandHandler {
	return func(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
		callArgs := append([]string{}, args...)
		if strings.TrimSpace(command) != "" {
			callArgs = append([]string{command}, callArgs...)
		}
		if err := runBashScript(root, globals, scriptPath, callArgs...); err != nil {
			return newErrorWithCategory(
				err,
				errorCategoryRuntime,
				"Check command arguments and script availability, or set VROOLI_FORCE_BASH=1 to reuse legacy entrypoint behavior",
				nil,
			)
		}
		return nil
	}
}

func makeTopLevelAutohealHandler(action string) topLevelCommandHandler {
	return func(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
		return runAutohealCommand(root, globals, append([]string{action}, args...)...)
	}
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

func runLegacyBash(root string, args []string) error {
	return execCommandFn(commandSpec{
		name: "bash",
		args: append([]string{filepath.Join(root, "cli", "vrooli")}, args...),
		dir:  root,
		env:  commandEnv(root, globalOptions{}),
	})
}

func runBashScript(root string, globals globalOptions, script string, args ...string) error {
	scriptArgs := append([]string{filepath.Join(root, filepath.FromSlash(script))}, args...)
	scriptArgs = append(scriptArgs, passthroughFlags(globals, args)...)
	return execCommandFn(commandSpec{
		name: "bash",
		args: scriptArgs,
		dir:  root,
		env:  commandEnv(root, globals),
	})
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
		return runAutohealCommand(root, parsed.globals, append([]string{"orphans", "kill"}, rest...)...)
	case "locks":
		return runAutohealCommand(root, parsed.globals, append([]string{"locks", "clean"}, rest...)...)
	case "help", "--help", "-h":
		showCleanupHelp(stdout)
		return nil
	default:
		_, _ = fmt.Fprintf(stderr, "Unknown cleanup target: %s\n", target)
		_, _ = fmt.Fprintln(stderr, "Run 'vrooli cleanup help' for usage")
		return exitCodeError{code: 1, message: "unknown cleanup target"}
	}
}

func runAutohealCommand(root string, globals globalOptions, args ...string) error {
	binary, err := lookPathFn("vrooli-autoheal")
	if err != nil {
		return newErrorWithCategory(
			errors.New("vrooli-autoheal not installed. Run 'vrooli setup' first"),
			errorCategoryRuntime,
			"Run 'vrooli setup' to install required lifecycle tools",
			nil,
		)
	}
	return execCommandFn(commandSpec{
		name: binary,
		args: append([]string{}, append(args, passthroughFlags(globals, args)...)...),
		dir:  root,
		env:  commandEnv(root, globals),
	})
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

func showMainHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "                          ___")
	_, _ = fmt.Fprintln(w, " _   _ _ __ ___   ___    / (_)")
	_, _ = fmt.Fprintln(w, "| | | | '__/ _ \\ / _ \\  / /| |")
	_, _ = fmt.Fprintln(w, "| |_| | | | (_) | (_) |/ / | |")
	_, _ = fmt.Fprintln(w, " \\___/|_|  \\___/ \\___//_/  |_|")
	_, _ = fmt.Fprintln(w, "                                   ")
	_, _ = fmt.Fprintln(w, "🚀 Vrooli CLI - AI Platform Management Tool")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "📋 USAGE:")
	_, _ = fmt.Fprintln(w, "    vrooli <command> [options]")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "🔄 LIFECYCLE COMMANDS:")
	_, _ = fmt.Fprintln(w, "    setup               Initialize the development environment")
	_, _ = fmt.Fprintln(w, "    develop             Start development servers")
	_, _ = fmt.Fprintln(w, "    build               Build the project")
	_, _ = fmt.Fprintln(w, "    deploy              Deploy to production")
	_, _ = fmt.Fprintln(w, "    clean               Clean build artifacts")
	_, _ = fmt.Fprintln(w, "    status              Show system health and status overview")
	_, _ = fmt.Fprintln(w, "    stop                Stop all or specific components (scenarios, resources, containers)")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "🧭 CONTEXT COMMANDS:")
	_, _ = fmt.Fprintln(w, "    info                Show consolidated project briefing")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "🎯 SCENARIO MANAGEMENT:")
	_, _ = fmt.Fprintln(w, "    # Scenarios run directly from their source location")
	_, _ = fmt.Fprintln(w, "    scenario list       List available scenarios")
	_, _ = fmt.Fprintln(w, "    scenario info <name> Show scenario metadata")
	_, _ = fmt.Fprintln(w, "    scenario status     Show scenario runtime state")
	_, _ = fmt.Fprintln(w, "    scenario run <name> Run a scenario directly")
	_, _ = fmt.Fprintln(w, "    scenario test <name> Test a scenario")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "🔧 RESOURCE MANAGEMENT:")
	_, _ = fmt.Fprintln(w, "    # Resources are external services and dependencies (databases, APIs, etc.)")
	_, _ = fmt.Fprintln(w, "    resource list       List available resources")
	_, _ = fmt.Fprintln(w, "    resource status     Show resource status")
	_, _ = fmt.Fprintln(w, "    resource install    Install a resource (initial setup)")
	_, _ = fmt.Fprintln(w, "    resource start      Start a resource")
	_, _ = fmt.Fprintln(w, "    resource start-all  Start all enabled resources")
	_, _ = fmt.Fprintln(w, "    resource stop       Stop a resource")
	_, _ = fmt.Fprintln(w, "    resource stop-all   Stop all running resources")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "⚙️ OPTIONS:")
	_, _ = fmt.Fprintln(w, "    --help, -h          Show help for a command")
	_, _ = fmt.Fprintln(w, "    --version, -v       Show version information")
	_, _ = fmt.Fprintln(w, "    --json              Forward JSON output mode to compatible commands")
	_, _ = fmt.Fprintln(w, "    --verbose           Forward verbose output mode to compatible commands")
	_, _ = fmt.Fprintln(w, "    --no-color          Disable ANSI color output")
	_, _ = fmt.Fprintln(w, "    --no-stale-check    Skip the Go source freshness check")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "📖 For more help on a specific command:")
	_, _ = fmt.Fprintln(w, "    vrooli <command> --help")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "📚 Documentation: docs/")
}

func showCleanupHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "vrooli cleanup - Clean up orphaned processes and stale locks")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  vrooli cleanup orphans    Kill orphaned Vrooli processes")
	_, _ = fmt.Fprintln(w, "  vrooli cleanup locks      Clean stale port lock files")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Options:")
	_, _ = fmt.Fprintln(w, "  --help, -h    Show this help message")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Examples:")
	_, _ = fmt.Fprintln(w, "  vrooli cleanup orphans    # Kill orphaned processes (interactive)")
	_, _ = fmt.Fprintln(w, "  vrooli cleanup locks      # Remove stale lock files")
}

func printUnknownCommand(w io.Writer, command string, suggestions []string) {
	_, _ = fmt.Fprintf(w, "Unknown command: %s\n", command)
	if len(suggestions) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Did you mean one of these?")
		for _, suggestion := range suggestions {
			_, _ = fmt.Fprintf(w, "  %s\n", suggestion)
		}
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Run 'vrooli --help' for usage information")
}

func suggestTopLevelCommands(command string) []string {
	candidates := make([]string, 0, len(topLevelCommands))
	for candidate := range topLevelCommands {
		if candidate == command {
			continue
		}
		if simpleDistance(command, candidate) <= 2 {
			candidates = append(candidates, candidate)
		}
	}
	sort.Strings(candidates)
	return candidates
}

func printErrorWithContext(w io.Writer, err error) {
	if err == nil {
		return
	}
	annotated, ok := err.(commandError)
	if !ok {
		_, _ = fmt.Fprintln(w, err)
		return
	}
	category := strings.TrimSpace(annotated.ErrorCategory())
	message := annotated.Error()
	if strings.HasPrefix(strings.ToLower(message), "unknown command: ") && category == errorCategoryUsage {
		command := strings.TrimSpace(strings.TrimPrefix(message, "unknown command: "))
		printUnknownCommand(w, command, annotated.ErrorSuggestions())
		return
	}
	if category != "" {
		_, _ = fmt.Fprintf(w, "%s error: %s\n", category, message)
	} else {
		_, _ = fmt.Fprintln(w, message)
	}
	if hint := strings.TrimSpace(annotated.ErrorHint()); hint != "" {
		_, _ = fmt.Fprintln(w, hint)
	}
	suggestions := annotated.ErrorSuggestions()
	if len(suggestions) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Did you mean one of these?")
	for _, suggestion := range suggestions {
		_, _ = fmt.Fprintf(w, "  %s\n", suggestion)
	}
	_, _ = fmt.Fprintln(w, "Run 'vrooli --help' for usage information")
}

type commandError interface {
	error
	ErrorCategory() string
	ErrorHint() string
	ErrorSuggestions() []string
}

type categorizedError struct {
	err         error
	category    string
	hint        string
	suggestions []string
}

func (e categorizedError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return ""
}
func (e categorizedError) ErrorCategory() string {
	return e.category
}
func (e categorizedError) ErrorHint() string {
	return e.hint
}
func (e categorizedError) ErrorSuggestions() []string {
	return append([]string(nil), e.suggestions...)
}

func (e categorizedError) ExitCode() int {
	var withCode interface{ ExitCode() int }
	if errors.As(e.err, &withCode) {
		return withCode.ExitCode()
	}
	return 1
}

func newErrorWithCategory(err error, category, hint string, suggestions []string) error {
	return categorizedError{
		err:         err,
		category:    category,
		hint:        hint,
		suggestions: append([]string(nil), suggestions...),
	}
}

func newUnknownCommandError(command string) error {
	return categorizedError{
		err:         fmt.Errorf("unknown command: %s", command),
		category:    errorCategoryUsage,
		hint:        "Run 'vrooli --help' for usage information",
		suggestions: suggestTopLevelCommands(command),
	}
}

// simpleDistance intentionally uses a cheap prefix-aware heuristic instead of a
// full edit-distance implementation. The CLI only needs rough typo recovery for
// obvious mistakes, and keeping this dependency-free keeps startup lightweight.
func simpleDistance(left, right string) int {
	maxLen := len(left)
	if len(right) > maxLen {
		maxLen = len(right)
	}
	minLen := len(left)
	if len(right) < minLen {
		minLen = len(right)
	}
	distance := maxLen - minLen
	for i := 0; i < minLen; i++ {
		if left[i] != right[i] {
			distance++
		}
	}
	return distance
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if codeErr, ok := err.(exitCodeError); ok {
		return codeErr.ExitCode()
	}
	var withExitCode interface{ ExitCode() int }
	if errors.As(err, &withExitCode) {
		return withExitCode.ExitCode()
	}
	return 1
}

func runExternalCommand(spec commandSpec) error {
	cmd := exec.Command(spec.name, spec.args...)
	cmd.Dir = spec.dir
	cmd.Env = spec.env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type exitCodeError struct {
	code    int
	message string
}

func (e exitCodeError) Error() string {
	if e.message != "" {
		return e.message
	}
	return fmt.Sprintf("exit code %d", e.code)
}

func (e exitCodeError) ExitCode() int {
	return e.code
}
