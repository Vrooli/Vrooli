package main

import (
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
// through the library group.
func programRuntimeCommandHint(_ []string) string {
	return "Declared programs are not top-level commands. Search with `program-runtime library search \"<intent>\"`, then run one with `program-runtime library run <scenario>.<program> --input key=value`. Example: `program-runtime library run program-runtime.prepare-operation --input name=<scenario>.<program>`"
}
