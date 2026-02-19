package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"ecosystem-manager/cli/internal/appctx"
	"ecosystem-manager/cli/logs"
	"ecosystem-manager/cli/queue"
	"ecosystem-manager/cli/steer"
	"ecosystem-manager/cli/tasks"
)

const (
	appName        = "ecosystem-manager"
	appVersion     = "1.0.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

// App wraps the cli-core ScenarioApp with ecosystem-manager specific functionality.
type App struct {
	core *cliapp.ScenarioApp
}

// Ensure App implements appctx.Context
var _ appctx.Context = (*App)(nil)

// NewApp creates a new ecosystem-manager CLI application.
func NewApp() (*App, error) {
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{
		ExtraAPIEnvVars: []string{"API_BASE_URL", "VITE_API_BASE_URL"},
	})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:              appName,
		Version:           appVersion,
		Description:       "Ecosystem Manager CLI - manage tasks, queue, and auto-steer profiles",
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
	app := &App{core: core}
	app.core.SetCommands(app.registerCommands())
	return app, nil
}

// Run executes the CLI with the given arguments.
func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

// registerCommands returns all command groups for the CLI.
func (a *App) registerCommands() []cliapp.CommandGroup {
	health := cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Check API health and connectivity", Run: a.cmdStatus},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	groups := []cliapp.CommandGroup{
		health,
	}

	groups = append(groups, tasks.Commands(a)...)
	groups = append(groups, steer.Commands(a))
	groups = append(groups, queue.Commands(a))
	groups = append(groups, logs.Commands())
	groups = append(groups, config)

	return groups
}

// apiPath ensures the path has the /api prefix.
// ecosystem-manager uses /api (not /api/v1).
func (a *App) apiPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	base := strings.TrimRight(strings.TrimSpace(a.core.HTTPClient.BaseURL()), "/")
	if strings.HasSuffix(base, "/api") {
		return path
	}
	return "/api" + path
}

// Get performs a GET request and unmarshals the response into result.
func (a *App) Get(path string, result interface{}) error {
	return a.GetWithQuery(path, nil, result)
}

// GetWithQuery performs a GET request with query parameters.
func (a *App) GetWithQuery(path string, query url.Values, result interface{}) error {
	body, err := a.core.APIClient.Request("GET", a.apiPath(path), query, nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, result)
}

// Post performs a POST request with the given payload and unmarshals the response into result.
func (a *App) Post(path string, payload interface{}, result interface{}) error {
	body, err := a.core.APIClient.Request("POST", a.apiPath(path), nil, payload)
	if err != nil {
		return err
	}
	if result != nil && len(body) > 0 {
		return json.Unmarshal(body, result)
	}
	return nil
}

// Put performs a PUT request with the given payload and unmarshals the response into result.
func (a *App) Put(path string, payload interface{}, result interface{}) error {
	body, err := a.core.APIClient.Request("PUT", a.apiPath(path), nil, payload)
	if err != nil {
		return err
	}
	if result != nil && len(body) > 0 {
		return json.Unmarshal(body, result)
	}
	return nil
}

// Delete performs a DELETE request.
func (a *App) Delete(path string) error {
	_, err := a.core.APIClient.Request("DELETE", a.apiPath(path), nil, nil)
	return err
}

// DeleteWithResult performs a DELETE request and unmarshals the response into result.
func (a *App) DeleteWithResult(path string, result interface{}) error {
	body, err := a.core.APIClient.Request("DELETE", a.apiPath(path), nil, nil)
	if err != nil {
		return err
	}
	if result != nil && len(body) > 0 {
		return json.Unmarshal(body, result)
	}
	return nil
}

// healthResponse represents the API health check response.
type healthResponse struct {
	Status     string            `json:"status"`
	Service    string            `json:"service"`
	Version    string            `json:"version"`
	Readiness  bool              `json:"readiness"`
	Timestamp  string            `json:"timestamp"`
	Deps       map[string]string `json:"dependencies"`
	Error      string            `json:"error,omitempty"`
	Message    string            `json:"message,omitempty"`
	Operations map[string]any    `json:"operations,omitempty"`
}

// cmdStatus checks the API health status.
func (a *App) cmdStatus(_ []string) error {
	body, err := a.core.APIClient.Get(a.apiPath("/health"), nil)
	if err != nil {
		return err
	}

	var parsed healthResponse
	if unmarshalErr := json.Unmarshal(body, &parsed); unmarshalErr == nil && parsed.Status != "" {
		fmt.Printf("Status: %s\n", parsed.Status)
		fmt.Printf("Ready: %v\n", parsed.Readiness)
		if parsed.Service != "" {
			fmt.Printf("Service: %s\n", parsed.Service)
		}
		if parsed.Version != "" {
			fmt.Printf("Version: %s\n", parsed.Version)
		}
		if len(parsed.Deps) > 0 {
			fmt.Println("Dependencies:")
			for key, value := range parsed.Deps {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}
