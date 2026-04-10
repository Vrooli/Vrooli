package main

import (
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	appName        = "stream-of-consciousness-analyzer"
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
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{
		ExtraAPIEnvVars: []string{"API_BASE_URL", "VITE_API_BASE_URL"},
	})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:              appName,
		Version:           appVersion,
		Description:       "Stream of Consciousness Analyzer CLI",
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

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	health := cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{Name: "status", Aliases: []string{"health"}, NeedsAPI: true, Description: "Check API health and readiness", Run: a.cmdStatus},
		},
	}

	schemes := cliapp.CommandGroup{
		Title: "Schemes",
		Commands: []cliapp.Command{
			{Name: "scheme list", NeedsAPI: true, Description: "List all schemes", Run: a.cmdSchemeList},
			{Name: "scheme get", NeedsAPI: true, Description: "Get a scheme by ID", Run: a.cmdSchemeGet},
			{Name: "scheme create", NeedsAPI: true, Description: "Create a new scheme", Run: a.cmdSchemeCreate},
			{Name: "scheme update", NeedsAPI: true, Description: "Update a scheme", Run: a.cmdSchemeUpdate},
			{Name: "scheme delete", NeedsAPI: true, Description: "Delete a scheme", Run: a.cmdSchemeDelete},
			{Name: "scheme export", NeedsAPI: true, Description: "Export a scheme's full graph", Run: a.cmdSchemeExport},
		},
	}

	thoughts := cliapp.CommandGroup{
		Title: "Thoughts",
		Commands: []cliapp.Command{
			{Name: "thought list", NeedsAPI: true, Description: "List thoughts (optionally filtered by scheme)", Run: a.cmdThoughtList},
			{Name: "thought get", NeedsAPI: true, Description: "Get a thought by ID", Run: a.cmdThoughtGet},
			{Name: "thought create", NeedsAPI: true, Description: "Create a new thought", Run: a.cmdThoughtCreate},
			{Name: "thought update", NeedsAPI: true, Description: "Update a thought", Run: a.cmdThoughtUpdate},
			{Name: "thought delete", NeedsAPI: true, Description: "Delete a thought", Run: a.cmdThoughtDelete},
		},
	}

	edges := cliapp.CommandGroup{
		Title: "Edges",
		Commands: []cliapp.Command{
			{Name: "edge list", NeedsAPI: true, Description: "List edges for a thought", Run: a.cmdEdgeList},
			{Name: "edge create", NeedsAPI: true, Description: "Create an edge between thoughts", Run: a.cmdEdgeCreate},
			{Name: "edge delete", NeedsAPI: true, Description: "Delete an edge", Run: a.cmdEdgeDelete},
		},
	}

	info := cliapp.CommandGroup{
		Title: "Information",
		Commands: []cliapp.Command{
			{Name: "info list", NeedsAPI: true, Description: "List information items for a scheme", Run: a.cmdInfoList},
			{Name: "info create", NeedsAPI: true, Description: "Create an information item", Run: a.cmdInfoCreate},
			{Name: "info update", NeedsAPI: true, Description: "Update an information item", Run: a.cmdInfoUpdate},
			{Name: "info delete", NeedsAPI: true, Description: "Delete an information item", Run: a.cmdInfoDelete},
		},
	}

	suggestions := cliapp.CommandGroup{
		Title: "Suggestions",
		Commands: []cliapp.Command{
			{Name: "provider list", NeedsAPI: true, Description: "List LLM providers and their status", Run: a.cmdProviderList},
			{Name: "suggestion generate", NeedsAPI: true, Description: "Generate suggestions for a scheme", Run: a.cmdSuggestionGenerate},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, schemes, thoughts, edges, info, suggestions, config}
}

func (a *App) apiPath(v1Path string) string {
	v1Path = strings.TrimSpace(v1Path)
	if v1Path == "" {
		return ""
	}
	if !strings.HasPrefix(v1Path, "/") {
		v1Path = "/" + v1Path
	}
	base := strings.TrimRight(strings.TrimSpace(a.core.HTTPClient.BaseURL()), "/")
	if strings.HasSuffix(base, "/api/v1") {
		return v1Path
	}
	return "/api/v1" + v1Path
}

// --- Health ---

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

func (a *App) cmdStatus(_ []string) error {
	jsonFalse := false
	return a.getResource("/health", &jsonFalse, func(body []byte) error {
		var parsed healthResponse
		if err := unmarshalBody(body, &parsed); err != nil || parsed.Status == "" {
			cliutil.PrintJSON(body)
			return nil
		}
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
	})
}
