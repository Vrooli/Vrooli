package main

import (
	"strings"

	"home-automation/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "home-automation"
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
		Description:         "Home Automation CLI",
		DefaultAPIBase:      defaultAPIBase,
		ExtraAPIEnvVars:     []string{"API_BASE_URL", "HOME_AUTOMATION_API_URL", "VITE_API_BASE_URL"},
		ExtraAPIPortEnvVars: []string{"HOME_AUTOMATION_API_PORT"},
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
	case "device":
		cleaned[0] = "devices"
	case "automation":
		cleaned[0] = "automations"
	case "profile":
		cleaned[0] = "profiles"
	case "context":
		cleaned[0] = "contexts"
	case "scene", "scenes":
		cleaned[0] = "contexts"
	case "ha":
		cleaned[0] = "home-assistant"
	}

	switch cleaned[0] {
	case "devices":
		if len(cleaned) == 1 || startsWithFlag(cleaned[1]) {
			cleaned = append([]string{"devices", "list"}, cleaned[1:]...)
		}
	case "automations":
		if len(cleaned) == 1 || startsWithFlag(cleaned[1]) {
			cleaned = append([]string{"automations", "list"}, cleaned[1:]...)
		}
	case "profiles":
		if len(cleaned) == 1 || startsWithFlag(cleaned[1]) {
			cleaned = append([]string{"profiles", "list"}, cleaned[1:]...)
		}
	case "contexts":
		if len(cleaned) == 1 || startsWithFlag(cleaned[1]) {
			cleaned = append([]string{"contexts", "current"}, cleaned[1:]...)
		}
	case "home-assistant":
		if len(cleaned) == 1 || startsWithFlag(cleaned[1]) {
			cleaned = append([]string{"home-assistant", "status"}, cleaned[1:]...)
		}
	}

	return cleaned
}

func startsWithFlag(arg string) bool {
	return strings.HasPrefix(strings.TrimSpace(arg), "-")
}
