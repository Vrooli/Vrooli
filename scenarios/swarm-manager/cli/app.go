// Package main provides the Swarm Manager CLI.
//
// DOC: docs/internal/INTENT.md#cli-components
// DOC: docs/internal/INTENT.md#flow-3-cli-health-check
// DOC: docs/internal/SEAMS.md#change-axes
//
// # Purpose
//
// The CLI provides terminal access to Swarm Manager functionality, primarily
// for operators and automation scripts. It communicates with the API server
// and provides human-readable output.
//
// # Current Status: Ideas CRUD Implemented
//
// The CLI supports full idea management (list, get, create, update, delete)
// and basic health checking via the status command.
//
// # Usage
//
//	swarm-manager status              # Check API health
//	swarm-manager ideas list          # List all ideas
//	swarm-manager ideas get <name>    # Get a single idea
//	swarm-manager ideas create <json> # Create new idea
//	swarm-manager ideas update <name> <json> # Update an idea
//	swarm-manager ideas delete <name> # Delete an idea
//	swarm-manager configure           # Set API base URL and token
//	swarm-manager --help              # Show all commands
//
// # Future Commands (P0)
//
//	swarm-manager ideas queue         # Queue idea for processing
//	swarm-manager scenarios list      # List all scenarios
//	swarm-manager scenarios status    # Show scenario status
//
// # API Discovery
//
// The CLI automatically discovers the API URL via:
//  1. --api-base flag
//  2. SWARM_MANAGER_API_BASE env var
//  3. vrooli scenario port detection
//
// Related PRD targets: OT-P0-002 (ideas CRUD), OT-P0-005 (scenario catalog)
package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	appName        = "swarm-manager"
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
		ExtraAPIEnvVars:     []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		ExtraAPIPortEnvVars: []string{"API_PORT"},
	})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:              appName,
		Version:           appVersion,
		Description:       "Swarm Manager CLI",
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
			{Name: "status", NeedsAPI: true, Description: "Check API health", Run: a.cmdStatus},
		},
	}

	ideas := cliapp.CommandGroup{
		Title: "Ideas",
		Commands: []cliapp.Command{
			{Name: "ideas list", NeedsAPI: true, Description: "List all ideas", Run: a.cmdIdeasList},
			{Name: "ideas get", NeedsAPI: true, Description: "Get a single idea by name (args: <name>)", Run: a.cmdIdeasGet},
			{Name: "ideas create", NeedsAPI: true, Description: "Create a new idea (args: <json>)", Run: a.cmdIdeasCreate},
			{Name: "ideas update", NeedsAPI: true, Description: "Update an existing idea (args: <name> <json>)", Run: a.cmdIdeasUpdate},
			{Name: "ideas delete", NeedsAPI: true, Description: "Delete an idea (args: <name>)", Run: a.cmdIdeasDelete},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, ideas, config}
}

// resolveV1Endpoint converts a relative endpoint path to the full API v1 path.
//
// This handles the case where the base URL may or may not include /api/v1.
// For example:
//   - Base: "http://localhost:3000", path: "/health" → "/api/v1/health"
//   - Base: "http://localhost:3000/api/v1", path: "/health" → "/health"
//
// The function ensures commands can use simple paths like "/health" without
// worrying about the configured base URL format.
func (a *App) resolveV1Endpoint(endpointPath string) string {
	endpointPath = strings.TrimSpace(endpointPath)
	if endpointPath == "" {
		return ""
	}
	if !strings.HasPrefix(endpointPath, "/") {
		endpointPath = "/" + endpointPath
	}
	base := strings.TrimRight(strings.TrimSpace(a.core.HTTPClient.BaseURL()), "/")
	if strings.HasSuffix(base, "/api/v1") {
		return endpointPath
	}
	return "/api/v1" + endpointPath
}

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

// cmdStatus checks the API health and displays the result.
//
// Output includes service status, version, readiness, and dependency health.
// For unstructured responses, it falls back to printing raw JSON.
func (a *App) cmdStatus(_ []string) error {
	body, err := a.core.APIClient.Get(a.resolveV1Endpoint("/health"), nil)
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

// Idea represents a proposal for a new scenario.
// [REQ:REQ-P0-003] Idea data structure for CLI display
type Idea struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Priority    int      `json:"priority"`
	Tags        []string `json:"tags"`
	Created     string   `json:"created"`
	Updated     string   `json:"updated"`
}

// IdeaResponse wraps a single idea response.
type IdeaResponse struct {
	Idea Idea `json:"idea"`
}

// ListIdeasResponse wraps idea list responses.
type ListIdeasResponse struct {
	Ideas []Idea `json:"ideas"`
}

// CreateIdeaRequest is the payload for creating a new idea.
type CreateIdeaRequest struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Priority    int      `json:"priority,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// cmdIdeasList lists all ideas.
// [REQ:REQ-P0-003] CLI ideas list command
func (a *App) cmdIdeasList(_ []string) error {
	body, err := a.core.APIClient.Get(a.resolveV1Endpoint("/ideas"), nil)
	if err != nil {
		return err
	}

	var response ListIdeasResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Ideas) == 0 {
		fmt.Println("No ideas found.")
		return nil
	}

	fmt.Printf("Found %d idea(s):\n\n", len(response.Ideas))
	for _, idea := range response.Ideas {
		fmt.Printf("  %s (priority: %d, status: %s)\n", idea.Name, idea.Priority, idea.Status)
		fmt.Printf("    Title: %s\n", idea.Title)
		if len(idea.Tags) > 0 {
			fmt.Printf("    Tags: %s\n", strings.Join(idea.Tags, ", "))
		}
		fmt.Println()
	}
	return nil
}

// cmdIdeasGet retrieves a single idea by name.
// [REQ:REQ-P0-003] CLI ideas get command
func (a *App) cmdIdeasGet(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: ideas get <name>")
	}
	name := args[0]

	body, err := a.core.APIClient.Get(a.resolveV1Endpoint("/ideas/"+name), nil)
	if err != nil {
		return err
	}

	var response IdeaResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	idea := response.Idea

	fmt.Printf("Name: %s\n", idea.Name)
	fmt.Printf("Title: %s\n", idea.Title)
	fmt.Printf("Description: %s\n", idea.Description)
	fmt.Printf("Status: %s\n", idea.Status)
	fmt.Printf("Priority: %d\n", idea.Priority)
	if len(idea.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(idea.Tags, ", "))
	}
	fmt.Printf("Created: %s\n", idea.Created)
	fmt.Printf("Updated: %s\n", idea.Updated)
	return nil
}

// cmdIdeasCreate creates a new idea.
// [REQ:REQ-P0-003] CLI ideas create command
func (a *App) cmdIdeasCreate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: ideas create <json>\n\nExample:\n  ideas create '{\"name\":\"my-idea\",\"title\":\"My Idea\",\"description\":\"Description here\"}'")
	}

	jsonStr := strings.Join(args, " ")
	var req CreateIdeaRequest
	if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if req.Name == "" || req.Title == "" {
		return fmt.Errorf("name and title are required fields")
	}

	body, err := a.core.APIClient.Request("POST", a.resolveV1Endpoint("/ideas"), nil, json.RawMessage(jsonStr))
	if err != nil {
		return err
	}

	var response IdeaResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	idea := response.Idea

	fmt.Printf("Created idea: %s\n", idea.Name)
	fmt.Printf("  Title: %s\n", idea.Title)
	fmt.Printf("  Status: %s\n", idea.Status)
	fmt.Printf("  Priority: %d\n", idea.Priority)
	return nil
}

// cmdIdeasUpdate updates an existing idea.
// [REQ:REQ-P0-003] CLI ideas update command
func (a *App) cmdIdeasUpdate(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: ideas update <name> <json>\n\nExample:\n  ideas update my-idea '{\"title\":\"Updated Title\",\"status\":\"ready\"}'")
	}

	name := args[0]
	jsonStr := strings.Join(args[1:], " ")

	// Validate JSON
	var update map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &update); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	body, err := a.core.APIClient.Request("PUT", a.resolveV1Endpoint("/ideas/"+name), nil, json.RawMessage(jsonStr))
	if err != nil {
		return err
	}

	var response IdeaResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	idea := response.Idea

	fmt.Printf("Updated idea: %s\n", idea.Name)
	fmt.Printf("  Title: %s\n", idea.Title)
	fmt.Printf("  Status: %s\n", idea.Status)
	fmt.Printf("  Priority: %d\n", idea.Priority)
	return nil
}

// cmdIdeasDelete deletes an idea.
// [REQ:REQ-P0-003] CLI ideas delete command
func (a *App) cmdIdeasDelete(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: ideas delete <name>")
	}
	name := args[0]

	_, err := a.core.APIClient.Request("DELETE", a.resolveV1Endpoint("/ideas/"+name), nil, nil)
	if err != nil {
		return err
	}

	fmt.Printf("Deleted idea: %s\n", name)
	return nil
}
