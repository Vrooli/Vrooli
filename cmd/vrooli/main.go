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
)

var infoDefaultFiles = []string{"docs/context.md"}

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

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	root, err := resolveRoot()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "vrooli: %v\n", err)
		return 1
	}
	primeRootEnv(root)

	if forceBashEnabled() {
		return exitCode(runLegacyBash(root, args))
	}

	parsed, err := parseArgs(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "vrooli: %v\n", err)
		return 1
	}
	if parsed.globals.noColor {
		_ = os.Setenv("NO_COLOR", "1")
	}

	if !parsed.globals.noStaleCheck && isStaleFn() {
		if err := rebuildAndReexecFn(args); err != nil {
			_, _ = fmt.Fprintf(stderr, "vrooli: stale binary check failed: %v\n", err)
			return 1
		}
		return 0
	}

	if err := dispatch(root, parsed, stdout, stderr); err != nil {
		if !hasExitCode(err) {
			_, _ = fmt.Fprintf(stderr, "vrooli: %v\n", err)
		}
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

// Week 1 keeps parsing intentionally shallow: normalize a few leading flags,
// then hand the real subcommand work to the existing Bash handlers.
func dispatch(root string, parsed parsedArgs, stdout, stderr io.Writer) error {
	switch parsed.command {
	case "", "help":
		showMainHelp(stdout)
		return nil
	case "version":
		showVersion(stdout, root)
		return nil
	case "info":
		return runInfoCommand(root, parsed.args, stdout, stderr)
	case "cleanup":
		return runCleanupCommand(root, parsed, stdout, stderr)
	case "orphans":
		return runAutohealCommand(root, parsed.globals, append([]string{"orphans"}, parsed.args...)...)
	case "locks":
		return runAutohealCommand(root, parsed.globals, append([]string{"locks"}, parsed.args...)...)
	case "diagnose-port":
		return runAutohealCommand(root, parsed.globals, append([]string{"diagnose-port"}, parsed.args...)...)
	case "setup", "develop", "build", "deploy", "backup", "restore":
		return runBashScript(root, parsed.globals, "scripts/manage.sh", append([]string{parsed.command}, parsed.args...)...)
	case "clean":
		return runBashScript(root, parsed.globals, "cli/commands/clean-commands.sh", parsed.args...)
	case "status":
		return runBashScript(root, parsed.globals, "cli/commands/status-command.sh", parsed.args...)
	case "scenario":
		return runBashScript(root, parsed.globals, "cli/commands/scenario/scenario-commands.sh", parsed.args...)
	case "resource":
		return runBashScript(root, parsed.globals, "cli/commands/resource-commands.sh", parsed.args...)
	case "stop":
		return runBashScript(root, parsed.globals, "cli/commands/stop-commands.sh", parsed.args...)
	case "doctor":
		return runBashScript(root, parsed.globals, "cli/commands/doctor.sh", parsed.args...)
	default:
		printUnknownCommand(stderr, parsed.command)
		return exitCodeError{code: 1, message: "unknown command"}
	}
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
		return errors.New("vrooli-autoheal not installed. Run 'vrooli setup' first")
	}
	return execCommandFn(commandSpec{
		name: binary,
		args: append([]string{}, append(args, passthroughFlags(globals, args)...)...),
		dir:  root,
		env:  commandEnv(root, globals),
	})
}

func runInfoCommand(root string, args []string, stdout, stderr io.Writer) error {
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

	infoFiles, err := collectInfoSources(root)
	if err != nil {
		return err
	}
	if len(infoFiles) == 0 {
		return errors.New("no context sources defined for vrooli info")
	}

	if showPathsOnly {
		for _, file := range infoFiles {
			_, _ = fmt.Fprintln(stdout, resolveInfoPath(root, file))
		}
		return nil
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

func showVersion(w io.Writer, root string) {
	_, _ = fmt.Fprintf(w, "Vrooli CLI v%s\n", cliVersion)
	_, _ = fmt.Fprintf(w, "Vrooli Platform v%s\n", vrooliVersion)
	_, _ = fmt.Fprintf(w, "Root: %s\n", root)
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

func printUnknownCommand(w io.Writer, command string) {
	_, _ = fmt.Fprintf(w, "Unknown command: %s\n", command)
	suggestions := suggestCommands(command)
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

func suggestCommands(command string) []string {
	available := []string{
		"setup", "develop", "build", "deploy", "clean", "backup", "restore",
		"status", "stop", "scenario", "resource", "doctor", "info", "version", "help",
	}
	suggestions := make([]string, 0, len(available))
	for _, candidate := range available {
		if candidate == command {
			continue
		}
		if simpleDistance(command, candidate) <= 2 {
			suggestions = append(suggestions, candidate)
		}
	}
	return suggestions
}

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

func hasExitCode(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := err.(exitCodeError); ok {
		return true
	}
	var withExitCode interface{ ExitCode() int }
	return errors.As(err, &withExitCode)
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
