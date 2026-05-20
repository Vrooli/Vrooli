package main

import (
	"browser-automation-studio/cli/domains"
	"browser-automation-studio/cli/internal/appctx"
	"path/filepath"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	appName        = "browser-automation-studio"
	appVersion     = "1.0.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

func boolPtr(v bool) *bool { return &v }

type App struct {
	core *cliapp.ScenarioApp
}

func NewApp() (*App, error) {
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{
		ExtraAPIEnvVars: []string{"BROWSER_AUTOMATION_API_URL", "API_BASE_URL", "VITE_API_BASE_URL"},
	})
	sourceRoot := cliutil.ResolveSourceRoot(buildSourceRoot, env.SourceRootEnvVars...)
	scenarioRoot := resolveScenarioRoot(sourceRoot)
	ctx := &appctx.Context{
		Name:         appName,
		Core:         nil,
		ScenarioRoot: scenarioRoot,
		TokenEnvVars: env.TokenEnvVars,
	}

	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:                 appName,
		Version:              appVersion,
		Description:          "Browser Automation Studio CLI - Visual browser automation with AI workflows",
		DefaultAPIBase:       defaultAPIBase,
		ExtraAPIEnvVars:      []string{"BROWSER_AUTOMATION_API_URL", "API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint:     buildFingerprint,
		BuildTimestamp:       buildTimestamp,
		BuildSourceRoot:      buildSourceRoot,
		AllowAnonymous:       true,
		IncludeStatusCommand: boolPtr(false),
		CommandGroups: func(app *cliapp.ScenarioApp) []cliapp.CommandGroup {
			ctx.Core = app
			return domains.CommandGroups(ctx)
		},
		SubcommandGroups: func(app *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
			ctx.Core = app
			return domains.SubcommandGroups(ctx, manifestBytes)
		},
	})
	if err != nil {
		return nil, err
	}

	return &App{core: core}, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func resolveScenarioRoot(sourceRoot string) string {
	if sourceRoot == "" {
		return ""
	}
	sourceRoot = filepath.Clean(sourceRoot)
	if filepath.Base(sourceRoot) == "cli" {
		return filepath.Dir(sourceRoot)
	}
	return sourceRoot
}
