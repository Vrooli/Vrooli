package main

import (
	"time"

	"deployment-manager/cli/cmdutil"
	"deployment-manager/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "deployment-manager"
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
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:               appName,
		Version:            appVersion,
		Description:        "Deployment Manager CLI",
		DefaultAPIBase:     defaultAPIBase,
		BuildFingerprint:   buildFingerprint,
		BuildTimestamp:     buildTimestamp,
		BuildSourceRoot:    buildSourceRoot,
		DefaultHTTPTimeout: 10 * time.Minute,
		AllowAnonymous:     true,
		ConfigureTokenKeys: []string{"token", "api_token"},
		CommandGroups:      domains.CommandGroups,
	})
	if err != nil {
		return nil, err
	}
	return &App{core: core}, nil
}

func (a *App) Run(args []string) error {
	remaining := applyGlobalFormat(args)
	return a.core.CLI.Run(remaining)
}

// applyGlobalFormat consumes leading global format flags (--json, --format <fmt>)
// so they do not conflict with per-command flag parsing.
func applyGlobalFormat(args []string) []string {
	if len(args) == 0 {
		return args
	}
	remaining := []string{}
	skipNext := false
	for i, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if arg == "--json" {
			cmdutil.SetGlobalFormat("json")
			continue
		}
		if arg == "--format" {
			if i+1 < len(args) {
				cmdutil.SetGlobalFormat(args[i+1])
				skipNext = true
			}
			continue
		}
		remaining = append(remaining, arg)
		// stop scanning once we hit the command name (first non-global flag token)
		break
	}
	// append the rest (command + subargs)
	if len(remaining) > 0 {
		idx := indexOf(args, remaining[0])
		if idx >= 0 && idx+1 < len(args) {
			remaining = append(remaining, args[idx+1:]...)
		}
	}
	return remaining
}

func indexOf(slice []string, target string) int {
	for i, v := range slice {
		if v == target {
			return i
		}
	}
	return -1
}
