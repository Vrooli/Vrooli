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
// # Current Status: Full API Surface Wired
//
// The CLI provides thin wrappers for ideas, scenarios, recommendations,
// settings, and queue endpoints, plus health checks.
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
// # Scenario Commands
//
//	swarm-manager ideas queue         # Queue idea for processing
//	swarm-manager ideas research      # Spawn research agent for an idea
//	swarm-manager scenarios list      # List all scenarios
//	swarm-manager scenarios get <name> # Get scenario details
//	swarm-manager scenarios update <name> <json> # Update scenario metadata
//	swarm-manager scenarios delete <name> [--archive] # Delete or archive scenario
//	swarm-manager recommendations list # List recommendations
//	swarm-manager recommendations refresh # Regenerate recommendations
//	swarm-manager recommendations create <json> # Create manual recommendation
//	swarm-manager recommendations update <id> <status> # Update recommendation status
//	swarm-manager settings get        # Fetch settings
//	swarm-manager settings update <json> # Update settings
//	swarm-manager queue list          # List local queue items
//	swarm-manager queue create <kind> [payload-json] # Enqueue item
//	swarm-manager queue delete <id>   # Remove item from queue
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
	"flag"
	"fmt"
	"net/url"
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
			{Name: "ideas queue", NeedsAPI: true, Description: "Queue an idea for processing (args: <name> [operation])", Run: a.cmdIdeasQueue},
			{Name: "ideas research", NeedsAPI: true, Description: "Spawn research agent for an idea (args: <name> [json])", Run: a.cmdIdeasResearch},
		},
	}

	scenarios := cliapp.CommandGroup{
		Title: "Scenarios",
		Commands: []cliapp.Command{
			{Name: "scenarios list", NeedsAPI: true, Description: "List all scenarios", Run: a.cmdScenariosList},
			{Name: "scenarios get", NeedsAPI: true, Description: "Get scenario details (args: <name>)", Run: a.cmdScenariosGet},
			{Name: "scenarios update", NeedsAPI: true, Description: "Update scenario metadata (args: <name> <json>)", Run: a.cmdScenariosUpdate},
			{Name: "scenarios delete", NeedsAPI: true, Description: "Delete a scenario (args: <name> [--archive])", Run: a.cmdScenariosDelete},
		},
	}

	recommendations := cliapp.CommandGroup{
		Title: "Recommendations",
		Commands: []cliapp.Command{
			{Name: "recommendations list", NeedsAPI: true, Description: "List recommendations", Run: a.cmdRecommendationsList},
			{Name: "recommendations refresh", NeedsAPI: true, Description: "Refresh recommendations", Run: a.cmdRecommendationsRefresh},
			{Name: "recommendations create", NeedsAPI: true, Description: "Create a manual recommendation (args: <json>)", Run: a.cmdRecommendationsCreate},
			{Name: "recommendations update", NeedsAPI: true, Description: "Update recommendation status (args: <id> <status>)", Run: a.cmdRecommendationsUpdate},
		},
	}

	settings := cliapp.CommandGroup{
		Title: "Settings",
		Commands: []cliapp.Command{
			{Name: "settings get", NeedsAPI: true, Description: "Get current settings", Run: a.cmdSettingsGet},
			{Name: "settings update", NeedsAPI: true, Description: "Update settings (args: <json>)", Run: a.cmdSettingsUpdate},
		},
	}

	queue := cliapp.CommandGroup{
		Title: "Queue",
		Commands: []cliapp.Command{
			{Name: "queue list", NeedsAPI: true, Description: "List queue items", Run: a.cmdQueueList},
			{Name: "queue create", NeedsAPI: true, Description: "Create a queue item (args: <kind> [payload-json])", Run: a.cmdQueueCreate},
			{Name: "queue delete", NeedsAPI: true, Description: "Delete a queue item (args: <id>)", Run: a.cmdQueueDelete},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, ideas, scenarios, recommendations, settings, queue, config}
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

// QueueIdeaResponse wraps queue response for ideas.
type QueueIdeaResponse struct {
	Idea   Idea   `json:"idea"`
	TaskID string `json:"task_id"`
}

// ResearchResponse represents research run metadata.
type ResearchResponse struct {
	TaskID  string `json:"taskId"`
	RunID   string `json:"runId"`
	BaseURL string `json:"baseUrl"`
	Created string `json:"created"`
}

// Scenario represents a scenario entry in the catalog.
type Scenario struct {
	Name                   string   `json:"name"`
	DisplayName            string   `json:"display_name"`
	Description            string   `json:"description"`
	Status                 string   `json:"status"`
	Priority               int      `json:"priority"`
	CompletenessScore      *int     `json:"completeness_score,omitempty"`
	IsGreenfield           bool     `json:"is_greenfield"`
	Tags                   []string `json:"tags"`
	RecommendationsEnabled bool     `json:"recommendations_enabled"`
}

// ScenarioResponse wraps scenario responses.
type ScenarioResponse struct {
	Scenario Scenario `json:"scenario"`
}

// ListScenariosResponse wraps scenario list responses.
type ListScenariosResponse struct {
	Scenarios []Scenario `json:"scenarios"`
}

// DeleteScenarioResponse wraps scenario deletion responses.
type DeleteScenarioResponse struct {
	Name     string `json:"name"`
	Archived bool   `json:"archived"`
	Message  string `json:"message"`
}

// Recommendation represents a recommendation item.
type Recommendation struct {
	ID          string `json:"id"`
	Scenario    string `json:"scenarioName"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    int    `json:"priority"`
	Created     string `json:"created"`
	Source      string `json:"source,omitempty"`
}

// RecommendationResponse wraps recommendation responses.
type RecommendationResponse struct {
	Recommendation Recommendation `json:"recommendation"`
}

// ListRecommendationsResponse wraps recommendation list responses.
type ListRecommendationsResponse struct {
	Recommendations []Recommendation `json:"recommendations"`
}

// QueueItem represents a local queue entry.
type QueueItem struct {
	ID      string          `json:"id"`
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Created string          `json:"created"`
}

// QueueListResponse wraps queue list responses.
type QueueListResponse struct {
	Items []QueueItem `json:"items"`
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

// cmdIdeasQueue queues an idea for processing.
func (a *App) cmdIdeasQueue(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: ideas queue <name> [operation]\n\nOperation: generator|improver")
	}
	name := args[0]
	var payload json.RawMessage
	if len(args) > 1 {
		operation := strings.TrimSpace(args[1])
		if operation != "generator" && operation != "improver" {
			return fmt.Errorf("invalid operation %q (expected generator or improver)", operation)
		}
		body := fmt.Sprintf(`{"operation":"%s"}`, operation)
		payload = json.RawMessage(body)
	}

	body, err := a.core.APIClient.Request("POST", a.resolveV1Endpoint("/ideas/"+name+"/queue"), nil, payload)
	if err != nil {
		return err
	}

	var response QueueIdeaResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Queued idea: %s\n", response.Idea.Name)
	fmt.Printf("  Status: %s\n", response.Idea.Status)
	fmt.Printf("  Task ID: %s\n", response.TaskID)
	return nil
}

// cmdIdeasResearch spawns a research agent for an idea.
func (a *App) cmdIdeasResearch(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: ideas research <name> [json]\n\nExample:\n  ideas research my-idea '{\"prompt\":\"Focus on risks\"}'")
	}
	name := args[0]
	var payload json.RawMessage
	if len(args) > 1 {
		jsonStr := strings.Join(args[1:], " ")
		var req map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
		payload = json.RawMessage(jsonStr)
	}

	body, err := a.core.APIClient.Request("POST", a.resolveV1Endpoint("/ideas/"+name+"/research"), nil, payload)
	if err != nil {
		return err
	}

	var response ResearchResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Research started for idea: %s\n", name)
	fmt.Printf("  Task ID: %s\n", response.TaskID)
	fmt.Printf("  Run ID: %s\n", response.RunID)
	fmt.Printf("  Base URL: %s\n", response.BaseURL)
	return nil
}

// cmdScenariosList lists scenarios with optional filters.
func (a *App) cmdScenariosList(args []string) error {
	fs := flag.NewFlagSet("scenarios list", flag.ContinueOnError)
	search := fs.String("search", "", "Filter by name or description")
	status := fs.String("status", "", "Filter by status (running|stopped|error|unknown)")
	tags := fs.String("tags", "", "Filter by tags (comma-separated)")
	sortField := fs.String("sort", "", "Sort by field (priority|name|displayName)")
	order := fs.String("order", "", "Sort order (asc|desc)")
	jsonOut := fs.Bool("json", false, "Output raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	query := url.Values{}
	if strings.TrimSpace(*search) != "" {
		query.Set("search", strings.TrimSpace(*search))
	}
	if strings.TrimSpace(*status) != "" {
		query.Set("status", strings.TrimSpace(*status))
	}
	if strings.TrimSpace(*tags) != "" {
		query.Set("tags", strings.TrimSpace(*tags))
	}
	if strings.TrimSpace(*sortField) != "" {
		query.Set("sort", strings.TrimSpace(*sortField))
	}
	if strings.TrimSpace(*order) != "" {
		query.Set("order", strings.TrimSpace(*order))
	}

	body, err := a.core.APIClient.Get(a.resolveV1Endpoint("/scenarios"), query)
	if err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var response ListScenariosResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Scenarios) == 0 {
		fmt.Println("No scenarios found.")
		return nil
	}

	fmt.Printf("Found %d scenario(s):\n\n", len(response.Scenarios))
	for _, scenario := range response.Scenarios {
		display := scenario.DisplayName
		if display == "" {
			display = scenario.Name
		}
		fmt.Printf("  %s (status: %s, priority: %d)\n", scenario.Name, scenario.Status, scenario.Priority)
		fmt.Printf("    Display: %s\n", display)
		if scenario.Description != "" {
			fmt.Printf("    Description: %s\n", scenario.Description)
		}
		if len(scenario.Tags) > 0 {
			fmt.Printf("    Tags: %s\n", strings.Join(scenario.Tags, ", "))
		}
		fmt.Println()
	}
	return nil
}

// cmdScenariosGet fetches scenario details.
func (a *App) cmdScenariosGet(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: scenarios get <name>")
	}
	name := args[0]

	body, err := a.core.APIClient.Get(a.resolveV1Endpoint("/scenarios/"+name), nil)
	if err != nil {
		return err
	}

	var response ScenarioResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	scenario := response.Scenario

	fmt.Printf("Name: %s\n", scenario.Name)
	fmt.Printf("Display Name: %s\n", scenario.DisplayName)
	fmt.Printf("Description: %s\n", scenario.Description)
	fmt.Printf("Status: %s\n", scenario.Status)
	fmt.Printf("Priority: %d\n", scenario.Priority)
	if scenario.CompletenessScore != nil {
		fmt.Printf("Completeness: %d\n", *scenario.CompletenessScore)
	}
	fmt.Printf("Greenfield: %v\n", scenario.IsGreenfield)
	fmt.Printf("Recommendations Enabled: %v\n", scenario.RecommendationsEnabled)
	if len(scenario.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(scenario.Tags, ", "))
	}
	return nil
}

// cmdScenariosUpdate updates scenario metadata.
func (a *App) cmdScenariosUpdate(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: scenarios update <name> <json>\n\nExample:\n  scenarios update my-scenario '{\"is_greenfield\":true}'")
	}
	name := args[0]
	jsonStr := strings.Join(args[1:], " ")

	var patch map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &patch); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	body, err := a.core.APIClient.Request("PATCH", a.resolveV1Endpoint("/scenarios/"+name), nil, json.RawMessage(jsonStr))
	if err != nil {
		return err
	}

	var response ScenarioResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	scenario := response.Scenario

	fmt.Printf("Updated scenario: %s\n", scenario.Name)
	fmt.Printf("  Greenfield: %v\n", scenario.IsGreenfield)
	fmt.Printf("  Recommendations Enabled: %v\n", scenario.RecommendationsEnabled)
	return nil
}

// cmdScenariosDelete deletes a scenario (optionally archived).
func (a *App) cmdScenariosDelete(args []string) error {
	fs := flag.NewFlagSet("scenarios delete", flag.ContinueOnError)
	archive := fs.Bool("archive", false, "Archive scenario to ideas backlog before deletion")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: scenarios delete <name> [--archive]")
	}
	name := fs.Arg(0)

	query := url.Values{}
	if *archive {
		query.Set("archive", "true")
	}

	body, err := a.core.APIClient.Request("DELETE", a.resolveV1Endpoint("/scenarios/"+name), query, nil)
	if err != nil {
		return err
	}

	var response DeleteScenarioResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println(response.Message)
	return nil
}

// cmdRecommendationsList lists recommendations with optional filters.
func (a *App) cmdRecommendationsList(args []string) error {
	fs := flag.NewFlagSet("recommendations list", flag.ContinueOnError)
	status := fs.String("status", "", "Filter by status (pending|approved|rejected)")
	scenario := fs.String("scenario", "", "Filter by scenario name")
	recType := fs.String("type", "", "Filter by type (test|feature|refactor|docs)")
	refresh := fs.Bool("refresh", false, "Force refresh before listing")
	jsonOut := fs.Bool("json", false, "Output raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	query := url.Values{}
	if strings.TrimSpace(*status) != "" {
		query.Set("status", strings.TrimSpace(*status))
	}
	if strings.TrimSpace(*scenario) != "" {
		query.Set("scenario", strings.TrimSpace(*scenario))
	}
	if strings.TrimSpace(*recType) != "" {
		query.Set("type", strings.TrimSpace(*recType))
	}
	if *refresh {
		query.Set("refresh", "true")
	}

	body, err := a.core.APIClient.Get(a.resolveV1Endpoint("/recommendations"), query)
	if err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var response ListRecommendationsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Recommendations) == 0 {
		fmt.Println("No recommendations found.")
		return nil
	}

	fmt.Printf("Found %d recommendation(s):\n\n", len(response.Recommendations))
	for _, rec := range response.Recommendations {
		fmt.Printf("  %s (%s, %s)\n", rec.ID, rec.Status, rec.Type)
		fmt.Printf("    Scenario: %s\n", rec.Scenario)
		fmt.Printf("    Priority: %d\n", rec.Priority)
		fmt.Printf("    Description: %s\n", rec.Description)
		fmt.Println()
	}
	return nil
}

// cmdRecommendationsRefresh regenerates recommendations.
func (a *App) cmdRecommendationsRefresh(_ []string) error {
	body, err := a.core.APIClient.Request("POST", a.resolveV1Endpoint("/recommendations/refresh"), nil, nil)
	if err != nil {
		return err
	}

	var response ListRecommendationsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Refreshed recommendations (%d total).\n", len(response.Recommendations))
	return nil
}

// cmdRecommendationsCreate creates a manual recommendation.
func (a *App) cmdRecommendationsCreate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: recommendations create <json>")
	}
	jsonStr := strings.Join(args, " ")
	var req map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	body, err := a.core.APIClient.Request("POST", a.resolveV1Endpoint("/recommendations"), nil, json.RawMessage(jsonStr))
	if err != nil {
		return err
	}

	var response RecommendationResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	rec := response.Recommendation

	fmt.Printf("Created recommendation: %s\n", rec.ID)
	fmt.Printf("  Scenario: %s\n", rec.Scenario)
	fmt.Printf("  Status: %s\n", rec.Status)
	return nil
}

// cmdRecommendationsUpdate updates recommendation status.
func (a *App) cmdRecommendationsUpdate(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: recommendations update <id> <status>")
	}
	id := args[0]
	status := strings.TrimSpace(args[1])
	if status != "pending" && status != "approved" && status != "rejected" {
		return fmt.Errorf("invalid status %q (expected pending|approved|rejected)", status)
	}

	payload := fmt.Sprintf(`{"status":"%s"}`, status)
	body, err := a.core.APIClient.Request("PATCH", a.resolveV1Endpoint("/recommendations/"+id), nil, json.RawMessage(payload))
	if err != nil {
		return err
	}

	var response RecommendationResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	rec := response.Recommendation

	fmt.Printf("Updated recommendation: %s\n", rec.ID)
	fmt.Printf("  Status: %s\n", rec.Status)
	return nil
}

// cmdSettingsGet fetches settings.
func (a *App) cmdSettingsGet(_ []string) error {
	body, err := a.core.APIClient.Get(a.resolveV1Endpoint("/settings"), nil)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

// cmdSettingsUpdate updates settings.
func (a *App) cmdSettingsUpdate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: settings update <json>")
	}
	jsonStr := strings.Join(args, " ")
	var patch map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &patch); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	body, err := a.core.APIClient.Request("PUT", a.resolveV1Endpoint("/settings"), nil, json.RawMessage(jsonStr))
	if err != nil {
		return err
	}

	cliutil.PrintJSON(body)
	return nil
}

// cmdQueueList lists local queue items.
func (a *App) cmdQueueList(_ []string) error {
	body, err := a.core.APIClient.Get(a.resolveV1Endpoint("/queue"), nil)
	if err != nil {
		return err
	}

	var response QueueListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Items) == 0 {
		fmt.Println("No queue items found.")
		return nil
	}

	fmt.Printf("Found %d queue item(s):\n\n", len(response.Items))
	for _, item := range response.Items {
		fmt.Printf("  %s (%s)\n", item.ID, item.Kind)
		fmt.Printf("    Created: %s\n", item.Created)
		if len(item.Payload) > 0 {
			fmt.Printf("    Payload: %s\n", string(item.Payload))
		}
		fmt.Println()
	}
	return nil
}

// cmdQueueCreate creates a new queue item.
func (a *App) cmdQueueCreate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: queue create <kind> [payload-json]")
	}
	kind := strings.TrimSpace(args[0])
	if kind == "" {
		return fmt.Errorf("kind is required")
	}

	payload := map[string]any{
		"kind": kind,
	}
	if len(args) > 1 {
		raw := strings.Join(args[1:], " ")
		var parsed json.RawMessage
		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			return fmt.Errorf("invalid payload JSON: %w", err)
		}
		payload["payload"] = json.RawMessage(raw)
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	body, err := a.core.APIClient.Request("POST", a.resolveV1Endpoint("/queue"), nil, json.RawMessage(requestBody))
	if err != nil {
		return err
	}

	cliutil.PrintJSON(body)
	return nil
}

// cmdQueueDelete removes a queue item.
func (a *App) cmdQueueDelete(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: queue delete <id>")
	}
	id := strings.TrimSpace(args[0])
	if id == "" {
		return fmt.Errorf("id is required")
	}

	_, err := a.core.APIClient.Request("DELETE", a.resolveV1Endpoint("/queue/"+id), nil, nil)
	if err != nil {
		return err
	}

	fmt.Printf("Deleted queue item: %s\n", id)
	return nil
}
