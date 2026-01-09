package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"browser-automation-studio/cli/executions"
	"browser-automation-studio/cli/internal/appctx"
	"browser-automation-studio/cli/playbooks"
	"browser-automation-studio/cli/recordings"
	"browser-automation-studio/cli/status"
	"browser-automation-studio/cli/workflows"

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

type App struct {
	core *cliapp.ScenarioApp
	ctx  *appctx.Context
}

func NewApp() (*App, error) {
	applyLegacyAPIEnv()
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{
		ExtraAPIEnvVars:     []string{"BROWSER_AUTOMATION_API_URL", "API_BASE_URL", "VITE_API_BASE_URL"},
		ExtraAPIPortEnvVars: []string{"API_PORT"},
	})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:              appName,
		Version:           appVersion,
		Description:       "Browser Automation Studio CLI - Visual browser automation with AI workflows",
		DefaultAPIBase:    defaultAPIBase,
		APIEnvVars:        env.APIEnvVars,
		APIPortEnvVars:    env.APIPortEnvVars,
		APIPortDetector:   cliutil.DetectPortFromVrooli(appName, "API_PORT"),
		ConfigDirEnvVars:  env.ConfigDirEnvVars,
		SourceRootEnvVars: env.SourceRootEnvVars,
		TokenEnvVars:      env.TokenEnvVars,
		BuildFingerprint:  buildFingerprint,
		BuildTimestamp:    buildTimestamp,
		BuildSourceRoot:   buildSourceRoot,
		AllowAnonymous:    true,
	})
	if err != nil {
		return nil, err
	}

	sourceRoot := cliutil.ResolveSourceRoot(buildSourceRoot, env.SourceRootEnvVars...)
	scenarioRoot := resolveScenarioRoot(sourceRoot)

	ctx := &appctx.Context{
		Name:         appName,
		Version:      appVersion,
		Core:         core,
		ScenarioRoot: scenarioRoot,
		TokenEnvVars: env.TokenEnvVars,
	}

	app := &App{core: core, ctx: ctx}
	app.core.SetCommands(app.registerCommands())
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	groups := []cliapp.CommandGroup{
		status.Commands(a.ctx),
		playbooks.Commands(a.ctx),
		workflows.Commands(a.ctx),
		executions.Commands(a.ctx),
		recordings.Commands(a.ctx),
		{
			Title: "Configuration",
			Commands: []cliapp.Command{
				a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
			},
		},
	}

	return groups
}

func applyLegacyAPIEnv() {
	if hasAPIBaseEnv() {
		return
	}

	host := strings.TrimSpace(os.Getenv("API_HOST"))
	port := strings.TrimSpace(os.Getenv("API_PORT"))
	if host == "" || port == "" {
		return
	}

	base := fmt.Sprintf("http://%s:%s", host, port)
	_ = os.Setenv("BROWSER_AUTOMATION_API_URL", base)
}

func hasAPIBaseEnv() bool {
	candidates := []string{
		"BROWSER_AUTOMATION_STUDIO_API_URL",
		"BROWSER_AUTOMATION_STUDIO_API_BASE",
		"BROWSER_AUTOMATION_API_URL",
		"API_BASE_URL",
		"VITE_API_BASE_URL",
	}
	for _, env := range candidates {
		if strings.TrimSpace(os.Getenv(env)) != "" {
			return true
		}
	}
	return false
}

func resolveScenarioRoot(sourceRoot string) string {
	sourceRoot = strings.TrimSpace(sourceRoot)
	if sourceRoot == "" {
		return ""
	}
	if filepath.Base(sourceRoot) == "cli" {
		return filepath.Dir(sourceRoot)
	}
	return sourceRoot
}
