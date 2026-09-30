package main

import (
	"strings"

	"scenario-stack-governor/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "scenario-stack-governor"
	appVersion     = "1.1.0"
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
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:                appName,
		Version:             appVersion,
		Description:         "Scenario Stack Governor CLI",
		DefaultAPIBase:      defaultAPIBase,
		ExtraAPIEnvVars:     []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		ExtraAPIPortEnvVars: []string{"SCENARIO_STACK_GOVERNOR_API_PORT"},
		BuildFingerprint:    buildFingerprint,
		BuildTimestamp:      buildTimestamp,
		BuildSourceRoot:     buildSourceRoot,
		AllowAnonymous:      true,
		CommandGroups:       domains.CommandGroups,
		SubcommandGroups:    domains.SubcommandGroups,
	})
	if err != nil {
		return nil, err
	}
	app.core = core
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(a.normalizeArgs(args))
}

func (a *App) normalizeArgs(args []string) []string {
	cleaned := make([]string, 0, len(args))
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg != "" {
			cleaned = append(cleaned, arg)
		}
	}
	if len(cleaned) == 0 {
		return cleaned
	}

	switch cleaned[0] {
	case "audit", "check":
		cleaned[0] = "run"
	case "apply-fixes":
		cleaned[0] = "fix"
	}

	if cleaned[0] == "rules" && len(cleaned) == 1 {
		return append(cleaned, "list")
	}
	if cleaned[0] == "scenarios" && len(cleaned) == 1 {
		return append(cleaned, "list")
	}
	return cleaned
}
