package main

import (
	"strings"

	"secrets-manager/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "secrets-manager"
	appVersion     = "1.2.0"
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
		Description:          "Secrets Manager CLI",
		DefaultAPIBase:       defaultAPIBase,
		ExtraAPIEnvVars:      []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		ExtraAPIPortEnvVars:  []string{"SECRETS_MANAGER_API_PORT"},
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
	case "vulnerabilities":
		cleaned = append([]string{"security", "vulnerabilities"}, cleaned[1:]...)
	case "scan":
		cleaned = append([]string{"security", "scan"}, cleaned[1:]...)
	case "compliance":
		cleaned = append([]string{"security", "compliance"}, cleaned[1:]...)
	case "plan":
		cleaned = append([]string{"deployment", "plan"}, cleaned[1:]...)
	case "resource":
		cleaned[0] = "resources"
	case "scenario":
		cleaned[0] = "scenarios"
	case "campaign":
		cleaned[0] = "campaigns"
	case "override":
		cleaned[0] = "overrides"
	}

	if cleaned[0] == "credentials" && shouldInsertDefault(cleaned[1:], "status", "list", "validate", "help", "-h", "--help") {
		cleaned = append([]string{"credentials", "status"}, cleaned[1:]...)
	}
	if cleaned[0] == "security" && shouldInsertDefault(cleaned[1:], "vulnerabilities", "scan", "compliance", "set-status", "fix", "help", "-h", "--help") {
		cleaned = append([]string{"security", "vulnerabilities"}, cleaned[1:]...)
	}
	if cleaned[0] == "deployment" && shouldInsertDefault(cleaned[1:], "plan", "readiness", "help", "-h", "--help") {
		cleaned = append([]string{"deployment", "plan"}, cleaned[1:]...)
	}
	if cleaned[0] == "resources" && shouldInsertDefault(cleaned[1:], "get", "update-secret", "set-strategy", "help", "-h", "--help") {
		cleaned = append([]string{"resources", "get"}, cleaned[1:]...)
	}
	if cleaned[0] == "scenarios" && len(cleaned) == 1 {
		cleaned = append(cleaned, "list")
	}
	if cleaned[0] == "campaigns" && len(cleaned) == 1 {
		cleaned = append(cleaned, "list")
	}
	if cleaned[0] == "overrides" && shouldInsertDefault(cleaned[1:], "list", "get", "set", "delete", "effective", "copy-from-tier", "copy-from-scenario", "help", "-h", "--help") {
		cleaned = append([]string{"overrides", "list"}, cleaned[1:]...)
	}
	if cleaned[0] == "admin" && shouldInsertDefault(cleaned[1:], "orphans", "cleanup-orphans", "help", "-h", "--help") {
		cleaned = append([]string{"admin", "orphans"}, cleaned[1:]...)
	}

	return cleaned
}

func shouldInsertDefault(args []string, known ...string) bool {
	if len(args) == 0 {
		return true
	}
	current := args[0]
	for _, item := range known {
		if current == item {
			return false
		}
	}
	return true
}
