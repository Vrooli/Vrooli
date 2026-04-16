package main

import (
	"fmt"
	"os"

	"scenario-completeness-scoring/cli/domains"
	"scenario-completeness-scoring/cli/format"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "scenario-completeness-scoring"
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

func main() {
	app, err := NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func NewApp() (*App, error) {
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:                appName,
		Version:             appVersion,
		Description:         "scenario-completeness-scoring CLI",
		DefaultAPIBase:      defaultAPIBase,
		ExtraAPIEnvVars:     []string{"SCORING_API_BASE"},
		ExtraAPIPortEnvVars: []string{"API_PORT"},
		OnColor:             format.SetColorEnabled,
		BuildFingerprint:    buildFingerprint,
		BuildTimestamp:      buildTimestamp,
		BuildSourceRoot:     buildSourceRoot,
		AllowAnonymous:      true,
		CommandGroups:       domains.CommandGroups,
	})
	if err != nil {
		return nil, err
	}
	return &App{core: core}, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}
