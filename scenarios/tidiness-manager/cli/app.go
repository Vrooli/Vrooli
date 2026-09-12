package main

import (
	"strings"

	"tidiness-manager/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "tidiness-manager"
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
		Description:         "Tidiness Manager CLI",
		DefaultAPIBase:      defaultAPIBase,
		ExtraAPIEnvVars:     []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		ExtraAPIPortEnvVars: []string{"TIDINESS_MANAGER_API_PORT"},
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
	case "campaign":
		cleaned[0] = "campaigns"
	case "visit":
		cleaned = append([]string{"tracking", "visit"}, cleaned[1:]...)
	case "exclude":
		cleaned = append([]string{"tracking", "exclude"}, cleaned[1:]...)
	case "campaign-note":
		cleaned = append([]string{"tracking", "campaign-note"}, cleaned[1:]...)
	}

	if cleaned[0] == "campaigns" && len(cleaned) == 1 {
		cleaned = append(cleaned, "list")
	}
	if cleaned[0] == "issues" && shouldInsertIssuesList(cleaned[1:]) {
		cleaned = append([]string{"issues", "list"}, cleaned[1:]...)
	}
	if cleaned[0] == "scenarios" && len(cleaned) == 1 {
		cleaned = append(cleaned, "list")
	}
	return cleaned
}

func shouldInsertIssuesList(args []string) bool {
	if len(args) == 0 {
		return true
	}
	switch args[0] {
	case "list", "resolve", "ignore", "reopen", "help", "-h", "--help":
		return false
	default:
		return true
	}
}
