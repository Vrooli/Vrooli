package main

import (
	"time"

	"audio-tools/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "audio-tools"
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

func NewApp() (*App, error) {
	app := &App{}
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:             appName,
		Version:          appVersion,
		Description:      "Audio Tools CLI",
		DefaultAPIBase:   defaultAPIBase,
		ExtraAPIEnvVars:  []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint: buildFingerprint,
		BuildTimestamp:   buildTimestamp,
		BuildSourceRoot:  buildSourceRoot,
		AllowAnonymous:   true,
		CommandGroups:    domains.CommandGroups,
		SubcommandGroups: domains.SubcommandGroups,
		// Summarize via the qwen3 reasoning model routinely takes
		// 30-90s on CPU because the model emits several hundred tokens
		// of <think> reasoning before the answer (see
		// internal/summarize). 30s (cli-core default) is shorter than
		// the server-side DefaultSummarizeTimeoutSeconds=120s and was
		// the first thing to time out for real summarize calls.
		DefaultHTTPTimeout: 180 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	app.core = core
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}
