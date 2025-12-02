package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	appName    = "test-genie"
	appVersion = "0.2.0"

	defaultAPIBase          = ""
	configDirEnvVar         = "TEST_GENIE_CONFIG_DIR"
	configDirGenericEnvVar  = "VROOLI_CLI_CONFIG_DIR"
	genericSourceRootEnvVar = "VROOLI_CLI_SOURCE_ROOT"
	legacySourceRootEnvVar  = "TEST_GENIE_CLI_SOURCE_ROOT"
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

type App struct {
	configStore *cliutil.ConfigFile
	config      cliutil.APIConfig
	apiOverride string
	client      *cliutil.HTTPClient
	api         *cliutil.APIClient
	services    *Services
	cli         *cliapp.App
}

func main() {
	app, err := NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func NewApp() (*App, error) {
	configStore, cfg, err := cliutil.LoadAPIConfig(appName, configDirEnvVar, configDirGenericEnvVar)
	if err != nil {
		return nil, err
	}
	app := &App{
		configStore: configStore,
		config:      cfg,
		apiOverride: "",
		client:      cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
	}
	app.api = cliutil.NewAPIClient(app.client, app.buildAPIBaseOptions, func() string { return app.config.Token })
	app.services = NewServices(app.api)
	app.cli = cliapp.NewApp(cliapp.AppOptions{
		Name:         appName,
		Version:      appVersion,
		Description:  "Test Genie CLI",
		Commands:     app.registerCommands(),
		APIOverride:  &app.apiOverride,
		ColorEnabled: cliapp.DefaultColorEnabled(),
		StaleChecker: cliutil.NewStaleChecker(appName, buildFingerprint, buildTimestamp, buildSourceRoot, genericSourceRootEnvVar, legacySourceRootEnvVar),
		Preflight:    app.preflightAPI,
	})
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.cli.Run(args)
}

func (a *App) saveConfig() error {
	return a.configStore.Save(a.config)
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	exec := cliapp.CommandGroup{
		Title: "Suites",
		Commands: []cliapp.Command{
			{Name: "generate", NeedsAPI: true, Description: "Queue suite generation", Run: a.cmdGenerate},
			{Name: "execute", NeedsAPI: true, Description: "Execute a suite for a scenario", Run: a.cmdExecute},
		},
	}
	local := cliapp.CommandGroup{
		Title: "Local",
		Commands: []cliapp.Command{
			{Name: "run-tests", NeedsAPI: true, Description: "Trigger scenario-local test runner", Run: a.cmdRunTests},
		},
	}
	system := cliapp.CommandGroup{
		Title: "System",
		Commands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Check Test Genie health", Run: func(args []string) error { return a.cmdStatus() }},
		},
	}
	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			{Name: "configure", NeedsAPI: false, Description: "View or update CLI settings", Run: a.cmdConfigure},
		},
	}
	return []cliapp.CommandGroup{exec, local, system, config}
}

func (a *App) buildAPIBaseOptions() cliutil.APIBaseOptions {
	return cliutil.APIBaseOptions{
		Override: a.apiOverride,
		EnvVars: []string{
			"TEST_GENIE_API_BASE",
			"TEST_GENIE_API_URL",
		},
		ConfigBase: a.config.APIBase,
		PortEnvVars: []string{
			"API_PORT",
			"TEST_GENIE_API_PORT",
		},
		PortDetector: cliutil.DetectPortFromVrooli(appName, "API_PORT"),
		DefaultBase:  defaultAPIBase,
	}
}

func (a *App) preflightAPI(cmd cliapp.Command, _ cliapp.GlobalOptions) error {
	if !cmd.NeedsAPI {
		return nil
	}
	_, err := cliutil.ValidateAPIBase(a.buildAPIBaseOptions())
	return err
}

func parseCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	var cleaned []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		cleaned = append(cleaned, part)
	}
	return cleaned
}

func parseMultiArgs(primary []string, extra []string) []string {
	result := append([]string{}, primary...)
	for _, item := range extra {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		result = append(result, item)
	}
	return result
}

func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func usageError(msg string) error {
	return errors.New(msg)
}
