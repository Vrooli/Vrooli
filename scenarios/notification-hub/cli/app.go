package main

import (
	"strings"

	"notification-hub/cli/domains"
	"notification-hub/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "notification-hub"
	appVersion     = "1.0.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

type App struct {
	core            *cliapp.ScenarioApp
	defaults        *support.DefaultsStore
	profileOverride string
}

func NewApp() (*App, error) {
	app := &App{}
	disableConfigure := false

	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:                    appName,
		Version:                 appVersion,
		Description:             "Notification Hub CLI",
		DefaultAPIBase:          defaultAPIBase,
		ExtraAPIEnvVars:         []string{"API_BASE_URL", "VITE_API_BASE_URL", "NOTIFICATION_HUB_API_URL"},
		ExtraTokenEnvVars:       []string{"NOTIFICATION_HUB_API_KEY"},
		BuildFingerprint:        buildFingerprint,
		BuildTimestamp:          buildTimestamp,
		BuildSourceRoot:         buildSourceRoot,
		AllowAnonymous:          true,
		IncludeConfigureCommand: &disableConfigure,
		CommandGroups: func(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
			return domains.CommandGroups(app.dependencies())
		},
		SubcommandGroups: func(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
			return domains.SubcommandGroups(app.dependencies())
		},
	})
	if err != nil {
		return nil, err
	}
	app.core = core

	defaults, err := support.NewDefaultsStore(core)
	if err != nil {
		return nil, err
	}
	app.defaults = defaults

	return app, nil
}

func (a *App) Run(args []string) error {
	normalized := a.normalizeArgs(args)
	remaining, profileID, apiKey, err := support.ExtractLegacyGlobals(normalized)
	if err != nil {
		return err
	}
	a.profileOverride = profileID
	if strings.TrimSpace(apiKey) != "" {
		a.core.Config.Token = strings.TrimSpace(apiKey)
	}
	return a.core.CLI.Run(remaining)
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
	case "profiles":
		if len(cleaned) == 1 {
			cleaned = append(cleaned, "list")
		}
	case "contacts":
		if len(cleaned) == 1 {
			cleaned = append(cleaned, "list")
		}
	case "templates":
		if len(cleaned) == 1 {
			cleaned = append(cleaned, "list")
		}
	case "notifications":
		if len(cleaned) == 1 {
			cleaned = append(cleaned, "list")
		}
	case "analytics":
		if len(cleaned) == 1 {
			cleaned = append(cleaned, "delivery-stats")
		}
	}

	return cleaned
}

func (a *App) dependencies() support.Dependencies {
	return support.Dependencies{
		Core: func() *cliapp.ScenarioApp {
			return a.core
		},
		Defaults: func() *support.DefaultsStore {
			return a.defaults
		},
		ProfileOverride: func() string {
			return a.profileOverride
		},
		AppName: appName,
	}
}
