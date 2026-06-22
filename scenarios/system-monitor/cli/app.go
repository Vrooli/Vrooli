package main

import (
	"strings"
	"system-monitor/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "system-monitor"
	appVersion     = "1.0.0"
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
	disableStatus := false
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:                 appName,
		Version:              appVersion,
		Description:          "System Monitor CLI",
		DefaultAPIBase:       defaultAPIBase,
		ExtraAPIEnvVars:      []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint:     buildFingerprint,
		BuildTimestamp:       buildTimestamp,
		BuildSourceRoot:      buildSourceRoot,
		AllowAnonymous:       true,
		IncludeStatusCommand: &disableStatus,
		CommandGroups:        domains.CommandGroups,
		SubcommandGroups:     domains.SubcommandGroups,
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
	case "metric":
		cleaned[0] = "metrics"
	case "investigation":
		cleaned[0] = "investigations"
	case "report":
		cleaned = append([]string{"reports", "generate"}, cleaned[1:]...)
	case "investigate":
		cleaned = append([]string{"investigations", "latest"}, cleaned[1:]...)
	}

	if cleaned[0] == "metrics" && len(cleaned) == 1 {
		cleaned = append(cleaned, "current")
	}
	if cleaned[0] == "investigations" && len(cleaned) == 1 {
		cleaned = append(cleaned, "list")
	}
	if cleaned[0] == "reports" && len(cleaned) == 1 {
		cleaned = append(cleaned, "list")
	}
	if cleaned[0] == "settings" && len(cleaned) == 1 {
		cleaned = append(cleaned, "get")
	}

	return cleaned
}
