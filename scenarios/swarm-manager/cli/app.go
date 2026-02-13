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
// # Current Status: Backlog/Scenarios/Settings/Queue Surface Wired
//
// The CLI provides thin wrappers for backlog, scenarios, settings, and queue
// endpoints, plus health checks.
//
// # Usage
//
//	swarm-manager status              # Check API health
//	swarm-manager backlog list              # List backlog items
//	swarm-manager backlog get <kind> <name> # Get a single backlog item
//	swarm-manager backlog create <json>     # Create backlog item
//	swarm-manager backlog update <kind> <name> <json> # Update backlog item
//	swarm-manager backlog delete <kind> <name> # Delete a backlog item
//	swarm-manager configure           # Set API base URL and token
//	swarm-manager --help              # Show all commands
//
// # Scenario Commands
//
//	swarm-manager backlog queue       # Queue backlog item for processing
//	swarm-manager backlog research    # Spawn research agent for a backlog item
//	swarm-manager backlog convert     # Convert backlog item to another kind
//	swarm-manager scenarios list      # List all scenarios
//	swarm-manager scenarios get <name> # Get scenario details
//	swarm-manager scenarios update <name> <json> # Update scenario metadata
//	swarm-manager scenarios delete <name> [--archive] # Delete or archive scenario
//	swarm-manager settings get        # Fetch settings
//	swarm-manager settings update <json> # Update settings
//	swarm-manager execution list      # List execution runs
//	swarm-manager execution get <execution-id> # Get execution details
//	swarm-manager execution policy get # Show execution defaults
//	swarm-manager execution policy update --mode manual|scheduled|yolo [--delay-seconds N]
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
// Related PRD targets: OT-P0-002 (backlog CRUD), OT-P0-005 (scenario catalog)
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

	backlog := cliapp.CommandGroup{
		Title: "Backlog",
		Commands: []cliapp.Command{
			{Name: "backlog list", NeedsAPI: true, Description: "List backlog items (args: [kinds])", Run: a.cmdBacklogList},
			{Name: "backlog get", NeedsAPI: true, Description: "Get a backlog item (args: <kind> <name>)", Run: a.cmdBacklogGet},
			{Name: "backlog create", NeedsAPI: true, Description: "Create a backlog item (args: <json>)", Run: a.cmdBacklogCreate},
			{Name: "backlog update", NeedsAPI: true, Description: "Update a backlog item (args: <kind> <name> <json>)", Run: a.cmdBacklogUpdate},
			{Name: "backlog delete", NeedsAPI: true, Description: "Delete a backlog item (args: <kind> <name>)", Run: a.cmdBacklogDelete},
			{Name: "backlog queue", NeedsAPI: true, Description: "Queue a backlog item (args: <kind> <name> [--mode ... --delay-seconds ...])", Run: a.cmdBacklogQueue},
			{Name: "backlog research", NeedsAPI: true, Description: "Spawn research agent for a backlog item (args: <kind> <name> [json])", Run: a.cmdBacklogResearch},
			{Name: "backlog convert", NeedsAPI: true, Description: "Convert a backlog item (args: <kind> <name> <target-kind> [target-name])", Run: a.cmdBacklogConvert},
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

	execution := cliapp.CommandGroup{
		Title: "Execution",
		Commands: []cliapp.Command{
			{Name: "execution list", NeedsAPI: true, Description: "List execution runs", Run: a.cmdExecutionList},
			{Name: "execution get", NeedsAPI: true, Description: "Get execution details (args: <execution-id>)", Run: a.cmdExecutionGet},
			{Name: "execution policy get", NeedsAPI: true, Description: "Get execution policy defaults", Run: a.cmdExecutionPolicyGet},
			{Name: "execution policy update", NeedsAPI: true, Description: "Update execution policy defaults (flags: --mode --delay-seconds)", Run: a.cmdExecutionPolicyUpdate},
			{Name: "execution start", NeedsAPI: true, Description: "Start an execution (args: <execution-id>)", Run: a.cmdExecutionStart},
			{Name: "execution cancel", NeedsAPI: true, Description: "Cancel an execution (args: <execution-id>)", Run: a.cmdExecutionCancel},
			{Name: "execution retry", NeedsAPI: true, Description: "Retry a failed execution (args: <execution-id>)", Run: a.cmdExecutionRetry},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, backlog, scenarios, settings, queue, execution, config}
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

// BacklogItem represents a tracked unit of work for the swarm.
// [REQ:REQ-P0-003] Backlog data structure for CLI display
type BacklogItem struct {
	Name           string   `json:"name"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Status         string   `json:"status"`
	Priority       int      `json:"priority"`
	Tags           []string `json:"tags"`
	Created        string   `json:"created"`
	Updated        string   `json:"updated"`
	Kind           string   `json:"kind"`
	ResearchTarget string   `json:"research_target,omitempty"`
}

// BacklogItemResponse wraps a single backlog item response.
type BacklogItemResponse struct {
	Item BacklogItem `json:"item"`
}

// ListBacklogResponse wraps backlog list responses.
type ListBacklogResponse struct {
	Items []BacklogItem `json:"items"`
}

// CreateBacklogRequest is the payload for creating a new backlog item.
type CreateBacklogRequest struct {
	Name           string   `json:"name"`
	Title          string   `json:"title"`
	Description    string   `json:"description,omitempty"`
	Priority       int      `json:"priority,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	Kind           string   `json:"kind"`
	ResearchTarget string   `json:"researchTarget,omitempty"`
}

// QueueBacklogResponse wraps queue response for backlog items.
type QueueBacklogResponse struct {
	Item    BacklogItem `json:"item"`
	TaskID  string      `json:"task_id"`
	RunID   string      `json:"run_id"`
	BaseURL string      `json:"base_url"`
	Created string      `json:"created"`
}

// ResearchResponse represents research run metadata.
type ResearchResponse struct {
	TaskID  string `json:"task_id"`
	RunID   string `json:"run_id"`
	BaseURL string `json:"base_url"`
	Created string `json:"created"`
}

// Scenario represents a scenario entry in the catalog.
type Scenario struct {
	Name              string   `json:"name"`
	DisplayName       string   `json:"display_name"`
	Description       string   `json:"description"`
	Status            string   `json:"status"`
	Priority          int      `json:"priority"`
	CompletenessScore *int     `json:"completeness_score,omitempty"`
	IsGreenfield      bool     `json:"is_greenfield"`
	Tags              []string `json:"tags"`
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

// ExecutionRecord represents an execution-control record.
type ExecutionRecord struct {
	ExecutionID   string `json:"execution_id"`
	BacklogKind   string `json:"backlog_kind"`
	BacklogName   string `json:"backlog_name"`
	TaskID        string `json:"task_id"`
	RunID         string `json:"run_id"`
	Status        string `json:"status"`
	Mode          string `json:"mode"`
	ScheduledAt   string `json:"scheduled_at"`
	StartedAt     string `json:"started_at"`
	FinishedAt    string `json:"finished_at"`
	FailureReason string `json:"failure_reason"`
	StartedBy     string `json:"started_by"`
	Operation     string `json:"operation"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// ExecutionListResponse wraps execution list response payloads.
type ExecutionListResponse struct {
	Items []ExecutionRecord `json:"items"`
}

// ExecutionItemResponse wraps a single execution response.
type ExecutionItemResponse struct {
	Execution ExecutionRecord `json:"execution"`
}

// ExecutionPolicy stores default execution behavior.
type ExecutionPolicy struct {
	DefaultMode         string `json:"default_mode"`
	DefaultDelaySeconds int64  `json:"default_delay_seconds"`
}

// ExecutionPolicyResponse wraps policy payloads.
type ExecutionPolicyResponse struct {
	Policy ExecutionPolicy `json:"policy"`
}

// cmdBacklogList lists backlog items.
// [REQ:REQ-P0-003] CLI backlog list command
func (a *App) cmdBacklogList(args []string) error {
	query := url.Values{}
	if len(args) > 0 {
		kinds := strings.Join(args, ",")
		if strings.TrimSpace(kinds) != "" {
			query.Set("kinds", kinds)
		}
	}

	body, err := a.core.APIClient.Get(a.resolveV1Endpoint("/backlog"), query)
	if err != nil {
		return err
	}

	var response ListBacklogResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Items) == 0 {
		fmt.Println("No backlog items found.")
		return nil
	}

	fmt.Printf("Found %d backlog item(s):\n\n", len(response.Items))
	for _, item := range response.Items {
		fmt.Printf("  [%s] %s (priority: %d, status: %s)\n", item.Kind, item.Name, item.Priority, item.Status)
		fmt.Printf("    Title: %s\n", item.Title)
		if len(item.Tags) > 0 {
			fmt.Printf("    Tags: %s\n", strings.Join(item.Tags, ", "))
		}
		if item.Kind == "research" && item.ResearchTarget != "" {
			fmt.Printf("    Target: %s\n", item.ResearchTarget)
		}
		fmt.Println()
	}
	return nil
}

// cmdBacklogGet retrieves a single backlog item.
// [REQ:REQ-P0-003] CLI backlog get command
func (a *App) cmdBacklogGet(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: backlog get <kind> <name>")
	}
	kind := args[0]
	name := args[1]

	body, err := a.core.APIClient.Get(a.resolveV1Endpoint("/backlog/"+kind+"/"+name), nil)
	if err != nil {
		return err
	}

	var response BacklogItemResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	item := response.Item

	fmt.Printf("Name: %s\n", item.Name)
	fmt.Printf("Kind: %s\n", item.Kind)
	fmt.Printf("Title: %s\n", item.Title)
	fmt.Printf("Description: %s\n", item.Description)
	fmt.Printf("Status: %s\n", item.Status)
	fmt.Printf("Priority: %d\n", item.Priority)
	if len(item.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(item.Tags, ", "))
	}
	if item.ResearchTarget != "" {
		fmt.Printf("Research Target: %s\n", item.ResearchTarget)
	}
	fmt.Printf("Created: %s\n", item.Created)
	fmt.Printf("Updated: %s\n", item.Updated)
	return nil
}

// cmdBacklogCreate creates a new backlog item.
// [REQ:REQ-P0-003] CLI backlog create command
func (a *App) cmdBacklogCreate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: backlog create <json>\n\nExample:\n  backlog create '{\"name\":\"my-idea\",\"title\":\"My Idea\",\"kind\":\"idea\"}'")
	}

	jsonStr := strings.Join(args, " ")
	var req CreateBacklogRequest
	if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if req.Name == "" || req.Title == "" || req.Kind == "" {
		return fmt.Errorf("name, title, and kind are required fields")
	}

	body, err := a.core.APIClient.Request("POST", a.resolveV1Endpoint("/backlog"), nil, json.RawMessage(jsonStr))
	if err != nil {
		return err
	}

	var response BacklogItemResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	item := response.Item

	fmt.Printf("Created backlog item: %s\n", item.Name)
	fmt.Printf("  Kind: %s\n", item.Kind)
	fmt.Printf("  Status: %s\n", item.Status)
	fmt.Printf("  Priority: %d\n", item.Priority)
	return nil
}

// cmdBacklogUpdate updates an existing backlog item.
// [REQ:REQ-P0-003] CLI backlog update command
func (a *App) cmdBacklogUpdate(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: backlog update <kind> <name> <json>\n\nExample:\n  backlog update idea my-idea '{\"title\":\"Updated Title\",\"status\":\"ready\"}'")
	}

	kind := args[0]
	name := args[1]
	jsonStr := strings.Join(args[2:], " ")

	var update map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &update); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	body, err := a.core.APIClient.Request("PUT", a.resolveV1Endpoint("/backlog/"+kind+"/"+name), nil, json.RawMessage(jsonStr))
	if err != nil {
		return err
	}

	var response BacklogItemResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	item := response.Item

	fmt.Printf("Updated backlog item: %s\n", item.Name)
	fmt.Printf("  Kind: %s\n", item.Kind)
	fmt.Printf("  Status: %s\n", item.Status)
	fmt.Printf("  Priority: %d\n", item.Priority)
	return nil
}

// cmdBacklogDelete deletes a backlog item.
// [REQ:REQ-P0-003] CLI backlog delete command
func (a *App) cmdBacklogDelete(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: backlog delete <kind> <name>")
	}
	kind := args[0]
	name := args[1]

	_, err := a.core.APIClient.Request("DELETE", a.resolveV1Endpoint("/backlog/"+kind+"/"+name), nil, nil)
	if err != nil {
		return err
	}

	fmt.Printf("Deleted backlog item: %s (%s)\n", name, kind)
	return nil
}

// cmdBacklogQueue queues a backlog item for processing.
func (a *App) cmdBacklogQueue(args []string) error {
	fs := flag.NewFlagSet("backlog queue", flag.ContinueOnError)
	mode := fs.String("mode", "", "Execution mode override: manual|scheduled|yolo (default uses execution policy)")
	delaySeconds := fs.Int64("delay-seconds", 0, "Schedule delay in seconds (scheduled mode)")
	operation := fs.String("operation", "generator", "Operation hint: generator|improver")
	startedBy := fs.String("started-by", "swarm-manager", "Started-by attribution label")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: backlog queue <kind> <name> [--mode manual|scheduled|yolo] [--delay-seconds N] [--operation generator|improver] [--started-by NAME]")
	}
	kind := fs.Arg(0)
	name := fs.Arg(1)
	modeValue := strings.ToLower(strings.TrimSpace(*mode))
	if modeValue != "" && modeValue != "manual" && modeValue != "scheduled" && modeValue != "yolo" {
		return fmt.Errorf("invalid mode %q (expected manual, scheduled, or yolo)", modeValue)
	}
	operationValue := strings.ToLower(strings.TrimSpace(*operation))
	if operationValue != "generator" && operationValue != "improver" {
		return fmt.Errorf("invalid operation %q (expected generator or improver)", operationValue)
	}
	if *delaySeconds < 0 {
		return fmt.Errorf("delay-seconds must be >= 0")
	}
	payload, err := json.Marshal(map[string]any{
		"operation":     operationValue,
		"mode":          modeValue,
		"delay_seconds": *delaySeconds,
		"started_by":    strings.TrimSpace(*startedBy),
	})
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	body, err := a.core.APIClient.Request("POST", a.resolveV1Endpoint("/backlog/"+kind+"/"+name+"/queue"), nil, json.RawMessage(payload))
	if err != nil {
		return err
	}

	var response QueueBacklogResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Queued backlog item: %s\n", response.Item.Name)
	fmt.Printf("  Kind: %s\n", response.Item.Kind)
	fmt.Printf("  Status: %s\n", response.Item.Status)
	fmt.Printf("  Task ID: %s\n", response.TaskID)
	if response.RunID != "" {
		fmt.Printf("  Run ID: %s\n", response.RunID)
	}
	fmt.Printf("  Mode: %s\n", modeValue)
	if modeValue == "scheduled" {
		fmt.Printf("  Delay Seconds: %d\n", *delaySeconds)
	}
	return nil
}

// cmdBacklogResearch spawns a research agent for a backlog item.
func (a *App) cmdBacklogResearch(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: backlog research <kind> <name> [json]\n\nExample:\n  backlog research idea my-idea '{\"prompt\":\"Focus on risks\"}'")
	}
	kind := args[0]
	name := args[1]
	var payload json.RawMessage
	if len(args) > 2 {
		jsonStr := strings.Join(args[2:], " ")
		var req map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
		payload = json.RawMessage(jsonStr)
	}

	body, err := a.core.APIClient.Request("POST", a.resolveV1Endpoint("/backlog/"+kind+"/"+name+"/research"), nil, payload)
	if err != nil {
		return err
	}

	var response ResearchResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Research started for backlog item: %s\n", name)
	fmt.Printf("  Task ID: %s\n", response.TaskID)
	fmt.Printf("  Run ID: %s\n", response.RunID)
	fmt.Printf("  Base URL: %s\n", response.BaseURL)
	return nil
}

// cmdBacklogConvert converts a backlog item to another kind.
func (a *App) cmdBacklogConvert(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: backlog convert <kind> <name> <target-kind> [target-name]")
	}
	kind := args[0]
	name := args[1]
	targetKind := args[2]
	targetName := ""
	if len(args) > 3 {
		targetName = strings.Join(args[3:], " ")
	}

	payload := map[string]string{
		"targetKind": targetKind,
	}
	if strings.TrimSpace(targetName) != "" {
		payload["targetName"] = targetName
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	body, err := a.core.APIClient.Request("POST", a.resolveV1Endpoint("/backlog/"+kind+"/"+name+"/convert"), nil, json.RawMessage(bodyBytes))
	if err != nil {
		return err
	}

	var response BacklogItemResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Converted backlog item: %s → %s/%s\n", name, response.Item.Kind, response.Item.Name)
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
	return nil
}

// cmdScenariosDelete deletes a scenario (optionally archived).
func (a *App) cmdScenariosDelete(args []string) error {
	fs := flag.NewFlagSet("scenarios delete", flag.ContinueOnError)
	archive := fs.Bool("archive", false, "Archive scenario to backlog (idea) before deletion")
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

// cmdExecutionList lists execution records with optional filters.
func (a *App) cmdExecutionList(args []string) error {
	fs := flag.NewFlagSet("execution list", flag.ContinueOnError)
	status := fs.String("status", "", "Filter by status")
	mode := fs.String("mode", "", "Filter by mode")
	backlogKind := fs.String("backlog-kind", "", "Filter by backlog kind")
	backlogName := fs.String("backlog-name", "", "Filter by backlog name")
	startedBy := fs.String("started-by", "", "Filter by started_by/source team")
	createdFrom := fs.String("created-from", "", "Filter by created_at lower bound (RFC3339)")
	createdTo := fs.String("created-to", "", "Filter by created_at upper bound (RFC3339)")
	jsonOut := fs.Bool("json", false, "Output raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	query := url.Values{}
	if strings.TrimSpace(*status) != "" {
		query.Set("status", strings.TrimSpace(*status))
	}
	if strings.TrimSpace(*mode) != "" {
		query.Set("mode", strings.TrimSpace(*mode))
	}
	if strings.TrimSpace(*backlogKind) != "" {
		query.Set("backlog_kind", strings.TrimSpace(*backlogKind))
	}
	if strings.TrimSpace(*backlogName) != "" {
		query.Set("backlog_name", strings.TrimSpace(*backlogName))
	}
	if strings.TrimSpace(*startedBy) != "" {
		query.Set("started_by", strings.TrimSpace(*startedBy))
	}
	if strings.TrimSpace(*createdFrom) != "" {
		query.Set("created_from", strings.TrimSpace(*createdFrom))
	}
	if strings.TrimSpace(*createdTo) != "" {
		query.Set("created_to", strings.TrimSpace(*createdTo))
	}

	body, err := a.core.APIClient.Get(a.resolveV1Endpoint("/execution"), query)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var response ExecutionListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	if len(response.Items) == 0 {
		fmt.Println("No execution runs found.")
		return nil
	}

	fmt.Printf("Found %d execution run(s):\n\n", len(response.Items))
	for _, item := range response.Items {
		fmt.Printf("  %s (%s)\n", item.ExecutionID, item.Status)
		fmt.Printf("    Backlog: %s/%s\n", item.BacklogKind, item.BacklogName)
		fmt.Printf("    Mode: %s\n", item.Mode)
		if item.RunID != "" {
			fmt.Printf("    Run ID: %s\n", item.RunID)
		}
		if item.TaskID != "" {
			fmt.Printf("    Task ID: %s\n", item.TaskID)
		}
		if item.FailureReason != "" {
			fmt.Printf("    Failure: %s\n", item.FailureReason)
		}
		fmt.Println()
	}
	return nil
}

// cmdExecutionGet fetches one execution record.
func (a *App) cmdExecutionGet(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: execution get <execution-id>")
	}
	executionID := strings.TrimSpace(args[0])
	body, err := a.core.APIClient.Get(a.resolveV1Endpoint("/execution/"+executionID), nil)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func (a *App) cmdExecutionPolicyGet(_ []string) error {
	body, err := a.core.APIClient.Get(a.resolveV1Endpoint("/execution/policy"), nil)
	if err != nil {
		return err
	}
	var response ExecutionPolicyResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	fmt.Printf("Default mode: %s\n", response.Policy.DefaultMode)
	fmt.Printf("Default delay seconds: %d\n", response.Policy.DefaultDelaySeconds)
	return nil
}

func (a *App) cmdExecutionPolicyUpdate(args []string) error {
	fs := flag.NewFlagSet("execution policy update", flag.ContinueOnError)
	mode := fs.String("mode", "", "Default mode: manual|scheduled|yolo")
	delay := fs.Int64("delay-seconds", 300, "Default delay seconds for scheduled mode")
	if err := fs.Parse(args); err != nil {
		return err
	}
	modeValue := strings.ToLower(strings.TrimSpace(*mode))
	if modeValue != "manual" && modeValue != "scheduled" && modeValue != "yolo" {
		return fmt.Errorf("mode is required and must be manual, scheduled, or yolo")
	}
	if *delay < 0 {
		return fmt.Errorf("delay-seconds must be >= 0")
	}
	payload, err := json.Marshal(map[string]any{
		"default_mode":          modeValue,
		"default_delay_seconds": *delay,
	})
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}
	body, err := a.core.APIClient.Request("PUT", a.resolveV1Endpoint("/execution/policy"), nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	var response ExecutionPolicyResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	fmt.Printf("Updated execution policy:\n")
	fmt.Printf("  Default mode: %s\n", response.Policy.DefaultMode)
	fmt.Printf("  Default delay seconds: %d\n", response.Policy.DefaultDelaySeconds)
	return nil
}

// cmdExecutionStart starts an execution now.
func (a *App) cmdExecutionStart(args []string) error {
	return a.runExecutionMutation(args, "start")
}

// cmdExecutionCancel cancels an execution.
func (a *App) cmdExecutionCancel(args []string) error {
	return a.runExecutionMutation(args, "cancel")
}

// cmdExecutionRetry retries a failed execution.
func (a *App) cmdExecutionRetry(args []string) error {
	return a.runExecutionMutation(args, "retry")
}

func (a *App) runExecutionMutation(args []string, action string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: execution %s <execution-id>", action)
	}
	executionID := strings.TrimSpace(args[0])
	body, err := a.core.APIClient.Request("POST", a.resolveV1Endpoint("/execution/"+executionID+"/"+action), nil, nil)
	if err != nil {
		return err
	}

	var response ExecutionItemResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Execution %s: %s\n", action, response.Execution.ExecutionID)
	fmt.Printf("  Status: %s\n", response.Execution.Status)
	fmt.Printf("  Backlog: %s/%s\n", response.Execution.BacklogKind, response.Execution.BacklogName)
	return nil
}
