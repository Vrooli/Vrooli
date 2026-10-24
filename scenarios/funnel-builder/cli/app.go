package main

import (
	"strings"

	"funnel-builder/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "funnel-builder"
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
		Description:         "Funnel Builder CLI",
		DefaultAPIBase:      defaultAPIBase,
		ExtraAPIEnvVars:     []string{"API_BASE_URL", "VITE_API_BASE_URL", "FUNNEL_API_URL"},
		ExtraAPIPortEnvVars: []string{"FUNNEL_BUILDER_API_PORT"},
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
	case "config":
		cleaned[0] = "configure"
	case "list", "create", "get", "delete", "analytics", "export-leads":
		cleaned = append([]string{"funnels", cleaned[0]}, cleaned[1:]...)
	case "templates":
		cleaned = append([]string{"templates", "list"}, cleaned[1:]...)
	}

	if cleaned[0] == "funnels" && len(cleaned) == 1 {
		cleaned = append(cleaned, "list")
	}
	if cleaned[0] == "projects" && len(cleaned) == 1 {
		cleaned = append(cleaned, "list")
	}
	if cleaned[0] == "templates" && len(cleaned) == 1 {
		cleaned = append(cleaned, "list")
	}

	return cleaned
}
