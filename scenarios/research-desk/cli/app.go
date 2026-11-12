package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"research-desk/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "research-desk"
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
		Version:          version(buildSourceRoot),
		Description:      "Research Desk CLI",
		DefaultAPIBase:   defaultAPIBase,
		ExtraAPIEnvVars:  []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint: buildFingerprint,
		BuildTimestamp:   buildTimestamp,
		BuildSourceRoot:  buildSourceRoot,
		AllowAnonymous:   true,
		CommandGroups: func(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
			groups, err := domains.CommandGroups(core, manifestBytes)
			if err != nil {
				panic(err) // manifest/binding drift is a build defect; fail on first run
			}
			return groups
		},
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

// version is service.version from the scenario's .vrooli/service.json, the
// one place the version is kept (the API reads the same field). cliRoot is
// the CLI source folder the installer baked in; "unknown" when it cannot be
// read.
func version(cliRoot string) string {
	data, err := os.ReadFile(filepath.Join(cliRoot, "..", ".vrooli", "service.json"))
	if err != nil {
		return "unknown"
	}
	var manifest struct {
		Service struct {
			Version string `json:"version"`
		} `json:"service"`
	}
	if json.Unmarshal(data, &manifest) != nil || manifest.Service.Version == "" {
		return "unknown"
	}
	return manifest.Service.Version
}
