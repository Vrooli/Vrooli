package main

import (
	"regexp"
	"strings"

	"program-runtime/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "program-runtime"
	appVersion     = "0.1.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

type App struct {
	core *cliapp.ScenarioApp
}

func NewApp() (*App, error) {
	app := &App{}
	subcommandGroups := func(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
		groups, err := domains.SubcommandGroups(core, manifestBytes)
		if err != nil {
			// Manifest parse / binding wiring is a programmer error caught
			// at NewApp time; surface it as a panic so misconfigured builds
			// fail loudly during the first CLI invocation rather than after
			// a user actually runs a command.
			panic(err)
		}
		return groups
	}
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:               appName,
		Version:            appVersion,
		Description:        "Program Runtime CLI",
		DefaultAPIBase:     defaultAPIBase,
		ExtraAPIEnvVars:    []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint:   buildFingerprint,
		BuildTimestamp:     buildTimestamp,
		BuildSourceRoot:    buildSourceRoot,
		AllowAnonymous:     true,
		CommandGroups:      domains.CommandGroups,
		SubcommandGroups:   subcommandGroups,
		UnknownCommandHint: programRuntimeCommandHint,
	})
	if err != nil {
		return nil, err
	}
	app.core = core
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

// programRuntimeCommandHint teaches the most common wrong shape: treating a
// declared program name as a top-level CLI command instead of selecting it
// through the library group. When the invocation shows which program was
// meant, it prints the same invocation rewritten to the library form.
func programRuntimeCommandHint(args []string) string {
	const guide = "Declared programs are not top-level commands. Search with `program-runtime library search \"<intent>\"`, then run one with `program-runtime library run <scenario>.<program> --input key=value`."
	if corrected := libraryRunRewrite(args); corrected != "" {
		return guide + "\nRun this invocation as:\n  " + corrected
	}
	return guide + " Example: `program-runtime library run program-runtime.prepare-operation --input name=<scenario>.<program>`"
}

// libraryRunRewrite maps `run <program> ...`, `<scenario>.<program> ...` and
// `<program> --input ...` onto `program-runtime library run <program> ...`.
// It returns "" when the invocation does not name a program.
func libraryRunRewrite(args []string) string {
	if len(args) == 0 {
		return ""
	}
	program, rest := args[0], args[1:]
	if program == "run" || program == "exec" {
		if len(rest) == 0 || strings.HasPrefix(rest[0], "-") {
			return ""
		}
		program, rest = rest[0], rest[1:]
	}
	switch {
	case programNamePattern.MatchString(program) && strings.Contains(program, "."):
	case programNamePattern.MatchString(program) && hasInputFlag(rest):
		program = "program-runtime." + program
	default:
		return ""
	}
	parts := []string{"program-runtime", "library", "run", program}
	for _, arg := range rest {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

var programNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*(\.[a-z0-9][a-z0-9-]*)?$`)

func hasInputFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--input" || strings.HasPrefix(arg, "--input=") {
			return true
		}
	}
	return false
}

func shellQuote(arg string) string {
	if arg != "" && !strings.ContainsAny(arg, " \t\n'\"\\$`!*?;&|<>()[]{}#~") {
		return arg
	}
	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}
