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

// toURLValues converts a map[string]string to url.Values
func toURLValues(params map[string]string) url.Values {
	if len(params) == 0 {
		return nil
	}
	v := url.Values{}
	for key, val := range params {
		v.Set(key, val)
	}
	return v
}

const (
	appName        = "lifestyle-dashboard"
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
		Description:       "Lifestyle Dashboard CLI",
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

	events := cliapp.CommandGroup{
		Title: "Events",
		Commands: []cliapp.Command{
			{Name: "event create", NeedsAPI: true, Description: "Create a new event", Run: a.cmdEventCreate},
			{Name: "event list", NeedsAPI: true, Description: "List events with optional filters", Run: a.cmdEventList},
			{Name: "event get", NeedsAPI: true, Description: "Get event by ID", Run: a.cmdEventGet},
		},
	}

	domains := cliapp.CommandGroup{
		Title: "Domains",
		Commands: []cliapp.Command{
			{Name: "domain register", NeedsAPI: true, Description: "Register a new domain", Run: a.cmdDomainRegister},
			{Name: "domain list", NeedsAPI: true, Description: "List all registered domains", Run: a.cmdDomainList},
			{Name: "domain get", NeedsAPI: true, Description: "Get domain by name", Run: a.cmdDomainGet},
			{Name: "domain update", NeedsAPI: true, Description: "Update domain attributes", Run: a.cmdDomainUpdate},
			{Name: "domain health", NeedsAPI: true, Description: "Check domain health status", Run: a.cmdDomainHealth},
		},
	}

	stats := cliapp.CommandGroup{
		Title: "Statistics",
		Commands: []cliapp.Command{
			{Name: "stats timeline", NeedsAPI: true, Description: "Get event timeline", Run: a.cmdStatsTimeline},
			{Name: "stats summary", NeedsAPI: true, Description: "Get aggregated statistics", Run: a.cmdStatsSummary},
			{Name: "stats score", NeedsAPI: true, Description: "Get daily lifestyle score", Run: a.cmdStatsScore},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, events, domains, stats, config}
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

// =============================================================================
// Health Commands
// =============================================================================

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

func (a *App) cmdStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.core.APIClient.Get(a.apiPath("/health"), nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
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

// =============================================================================
// Event Commands
// =============================================================================

type createEventRequest struct {
	Domain         string          `json:"domain"`
	EventType      string          `json:"event_type"`
	Payload        json.RawMessage `json:"payload,omitempty"`
	Timestamp      *string         `json:"timestamp,omitempty"`
	IsIntervention bool            `json:"is_intervention"`
	HypothesisID   *string         `json:"hypothesis_id,omitempty"`
}

type eventResponse struct {
	ID             string          `json:"id"`
	Domain         string          `json:"domain"`
	EventType      string          `json:"event_type"`
	Timestamp      string          `json:"timestamp"`
	Payload        json.RawMessage `json:"payload"`
	IsIntervention bool            `json:"is_intervention"`
	HypothesisID   *string         `json:"hypothesis_id,omitempty"`
	CreatedAt      string          `json:"created_at"`
}

type eventsListResponse struct {
	Events []eventResponse `json:"events"`
	Count  int             `json:"count"`
}

func (a *App) cmdEventCreate(args []string) error {
	fs := flag.NewFlagSet("event create", flag.ContinueOnError)
	domain := fs.String("domain", "", "Domain name (required)")
	eventType := fs.String("type", "", "Event type (required)")
	payload := fs.String("payload", "", "JSON payload (optional)")
	timestamp := fs.String("timestamp", "", "Timestamp in RFC3339 format (optional)")
	isIntervention := fs.Bool("intervention", false, "Mark as intervention event")
	hypothesisID := fs.String("hypothesis", "", "Associated hypothesis ID (optional)")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *domain == "" {
		return fmt.Errorf("--domain is required")
	}
	if *eventType == "" {
		return fmt.Errorf("--type is required")
	}

	req := createEventRequest{
		Domain:         *domain,
		EventType:      *eventType,
		IsIntervention: *isIntervention,
	}
	if *payload != "" {
		req.Payload = json.RawMessage(*payload)
	}
	if *timestamp != "" {
		req.Timestamp = timestamp
	}
	if *hypothesisID != "" {
		req.HypothesisID = hypothesisID
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/events"), nil, req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp eventResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	fmt.Printf("Created event: %s\n", resp.ID)
	fmt.Printf("  Domain: %s\n", resp.Domain)
	fmt.Printf("  Type: %s\n", resp.EventType)
	fmt.Printf("  Timestamp: %s\n", resp.Timestamp)
	return nil
}

func (a *App) cmdEventList(args []string) error {
	fs := flag.NewFlagSet("event list", flag.ContinueOnError)
	domain := fs.String("domain", "", "Filter by domain")
	eventType := fs.String("type", "", "Filter by event type")
	start := fs.String("start", "", "Filter events after this timestamp (RFC3339)")
	end := fs.String("end", "", "Filter events before this timestamp (RFC3339)")
	limit := fs.Int("limit", 0, "Maximum number of events to return")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	params := make(map[string]string)
	if *domain != "" {
		params["domain"] = *domain
	}
	if *eventType != "" {
		params["event_type"] = *eventType
	}
	if *start != "" {
		params["start"] = *start
	}
	if *end != "" {
		params["end"] = *end
	}
	if *limit > 0 {
		params["limit"] = fmt.Sprintf("%d", *limit)
	}

	body, err := a.core.APIClient.Get(a.apiPath("/events"), toURLValues(params))
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp eventsListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("Events: %d\n", resp.Count)
	if resp.Count > 0 {
		fmt.Println()
		for _, e := range resp.Events {
			fmt.Printf("  %s [%s] %s/%s\n", e.ID[:8], e.Timestamp[:10], e.Domain, e.EventType)
		}
	}
	return nil
}

func (a *App) cmdEventGet(args []string) error {
	fs := flag.NewFlagSet("event get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: event get <id> [--json]")
	}
	id := fs.Arg(0)

	body, err := a.core.APIClient.Get(a.apiPath("/events/"+id), nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp eventResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("ID: %s\n", resp.ID)
	fmt.Printf("Domain: %s\n", resp.Domain)
	fmt.Printf("Type: %s\n", resp.EventType)
	fmt.Printf("Timestamp: %s\n", resp.Timestamp)
	fmt.Printf("Intervention: %v\n", resp.IsIntervention)
	if resp.HypothesisID != nil {
		fmt.Printf("Hypothesis: %s\n", *resp.HypothesisID)
	}
	if len(resp.Payload) > 0 && string(resp.Payload) != "null" {
		fmt.Printf("Payload: %s\n", string(resp.Payload))
	}
	return nil
}

// =============================================================================
// Domain Commands
// =============================================================================

type registerDomainRequest struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Description  string   `json:"description,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	HealthURL    string   `json:"health_url,omitempty"`
}

type domainResponse struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Description  string   `json:"description,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Status       string   `json:"status"`
	HealthURL    string   `json:"health_url,omitempty"`
	LastHealthAt *string  `json:"last_health_at,omitempty"`
	RegisteredAt string   `json:"registered_at"`
	UpdatedAt    string   `json:"updated_at"`
}

type domainsListResponse struct {
	Domains []domainResponse `json:"domains"`
	Count   int              `json:"count"`
}

type healthCheckResponse struct {
	Domain    string `json:"domain"`
	Status    string `json:"status"`
	LastCheck string `json:"last_check"`
	Message   string `json:"message,omitempty"`
}

func (a *App) cmdDomainRegister(args []string) error {
	fs := flag.NewFlagSet("domain register", flag.ContinueOnError)
	name := fs.String("name", "", "Domain name (required)")
	displayName := fs.String("display-name", "", "Display name (required)")
	description := fs.String("description", "", "Description (optional)")
	capabilities := fs.String("capabilities", "", "Comma-separated capabilities (optional)")
	healthURL := fs.String("health-url", "", "Health check URL (optional)")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *name == "" {
		return fmt.Errorf("--name is required")
	}
	if *displayName == "" {
		return fmt.Errorf("--display-name is required")
	}

	req := registerDomainRequest{
		Name:        *name,
		DisplayName: *displayName,
		Description: *description,
		HealthURL:   *healthURL,
	}
	if *capabilities != "" {
		req.Capabilities = cliutil.ParseCSV(*capabilities)
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/domains"), nil, req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp domainResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	fmt.Printf("Registered domain: %s\n", resp.Name)
	fmt.Printf("  Display Name: %s\n", resp.DisplayName)
	fmt.Printf("  Status: %s\n", resp.Status)
	return nil
}

func (a *App) cmdDomainList(args []string) error {
	fs := flag.NewFlagSet("domain list", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.core.APIClient.Get(a.apiPath("/domains"), nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp domainsListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("Domains: %d\n", resp.Count)
	if resp.Count > 0 {
		fmt.Println()
		for _, d := range resp.Domains {
			fmt.Printf("  %s (%s) - %s\n", d.Name, d.DisplayName, d.Status)
		}
	}
	return nil
}

func (a *App) cmdDomainGet(args []string) error {
	fs := flag.NewFlagSet("domain get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: domain get <name> [--json]")
	}
	name := fs.Arg(0)

	body, err := a.core.APIClient.Get(a.apiPath("/domains/"+name), nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp domainResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("Name: %s\n", resp.Name)
	fmt.Printf("Display Name: %s\n", resp.DisplayName)
	fmt.Printf("Status: %s\n", resp.Status)
	if resp.Description != "" {
		fmt.Printf("Description: %s\n", resp.Description)
	}
	if len(resp.Capabilities) > 0 {
		fmt.Printf("Capabilities: %s\n", strings.Join(resp.Capabilities, ", "))
	}
	if resp.HealthURL != "" {
		fmt.Printf("Health URL: %s\n", resp.HealthURL)
	}
	if resp.LastHealthAt != nil {
		fmt.Printf("Last Health Check: %s\n", *resp.LastHealthAt)
	}
	fmt.Printf("Registered: %s\n", resp.RegisteredAt)
	fmt.Printf("Updated: %s\n", resp.UpdatedAt)
	return nil
}

func (a *App) cmdDomainUpdate(args []string) error {
	fs := flag.NewFlagSet("domain update", flag.ContinueOnError)
	displayName := fs.String("display-name", "", "New display name")
	description := fs.String("description", "", "New description")
	healthURL := fs.String("health-url", "", "New health check URL")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: domain update <name> [--display-name NAME] [--description DESC] [--health-url URL] [--json]")
	}
	name := fs.Arg(0)

	updates := make(map[string]interface{})
	if *displayName != "" {
		updates["display_name"] = *displayName
	}
	if *description != "" {
		updates["description"] = *description
	}
	if *healthURL != "" {
		updates["health_url"] = *healthURL
	}

	if len(updates) == 0 {
		return fmt.Errorf("no updates specified; use --display-name, --description, or --health-url")
	}

	body, err := a.core.APIClient.Request("PATCH", a.apiPath("/domains/"+name), nil, updates)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp domainResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("Updated domain: %s\n", resp.Name)
	fmt.Printf("  Display Name: %s\n", resp.DisplayName)
	fmt.Printf("  Status: %s\n", resp.Status)
	return nil
}

func (a *App) cmdDomainHealth(args []string) error {
	fs := flag.NewFlagSet("domain health", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: domain health <name> [--json]")
	}
	name := fs.Arg(0)

	body, err := a.core.APIClient.Get(a.apiPath("/domains/"+name+"/health"), nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp healthCheckResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("Domain: %s\n", resp.Domain)
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Last Check: %s\n", resp.LastCheck)
	if resp.Message != "" {
		fmt.Printf("Message: %s\n", resp.Message)
	}
	return nil
}

// =============================================================================
// Stats Commands
// =============================================================================

type timelineEntry struct {
	Day    string `json:"day"`
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

type timelineResponse struct {
	Timeline []timelineEntry `json:"timeline"`
	Days     string          `json:"days"`
}

type domainCount struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

type summaryResponse struct {
	TotalEvents    int           `json:"total_events"`
	ActiveDomains  int           `json:"active_domains"`
	EventsByDomain []domainCount `json:"events_by_domain"`
	LastEventAt    string        `json:"last_event_at"`
}

func (a *App) cmdStatsTimeline(args []string) error {
	fs := flag.NewFlagSet("stats timeline", flag.ContinueOnError)
	days := fs.Int("days", 0, "Number of days to include (default: 7)")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	params := make(map[string]string)
	if *days > 0 {
		params["days"] = fmt.Sprintf("%d", *days)
	}

	body, err := a.core.APIClient.Get(a.apiPath("/stats/timeline"), toURLValues(params))
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp timelineResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("Timeline (%s days)\n", resp.Days)
	if len(resp.Timeline) > 0 {
		fmt.Println()
		currentDay := ""
		for _, e := range resp.Timeline {
			if e.Day != currentDay {
				if currentDay != "" {
					fmt.Println()
				}
				currentDay = e.Day
				fmt.Printf("  %s:\n", e.Day)
			}
			fmt.Printf("    %s: %d events\n", e.Domain, e.Count)
		}
	} else {
		fmt.Println("  No events in this period")
	}
	return nil
}

func (a *App) cmdStatsSummary(args []string) error {
	fs := flag.NewFlagSet("stats summary", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.core.APIClient.Get(a.apiPath("/stats/summary"), nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp summaryResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("Total Events: %d\n", resp.TotalEvents)
	fmt.Printf("Active Domains: %d\n", resp.ActiveDomains)
	if resp.LastEventAt != "" {
		fmt.Printf("Last Event: %s\n", resp.LastEventAt)
	}
	if len(resp.EventsByDomain) > 0 {
		fmt.Println("\nEvents by Domain:")
		for _, dc := range resp.EventsByDomain {
			fmt.Printf("  %s: %d\n", dc.Domain, dc.Count)
		}
	}
	return nil
}

// Score types
// [REQ:LD-UI-SCORE] CLI types for lifestyle score display

type domainScoreEntry struct {
	Domain      string  `json:"domain"`
	DisplayName string  `json:"display_name"`
	Score       int     `json:"score"`
	Weight      float64 `json:"weight"`
	EventCount  int     `json:"event_count"`
}

type lifestyleScore struct {
	Score               int                `json:"score"`
	Date                string             `json:"date"`
	DomainScores        []domainScoreEntry `json:"domain_scores"`
	Trend               string             `json:"trend"`
	ChangeFromYesterday int                `json:"change_from_yesterday"`
	DataQuality         string             `json:"data_quality"`
	Message             string             `json:"message"`
}

type scoreHistoryEntry struct {
	Date  string `json:"date"`
	Score int    `json:"score"`
}

type scoreResponse struct {
	Current lifestyleScore      `json:"current"`
	History []scoreHistoryEntry `json:"history"`
}

// cmdStatsScore handles the "stats score" command
// [REQ:LD-UI-SCORE] CLI command for lifestyle score
func (a *App) cmdStatsScore(args []string) error {
	fs := flag.NewFlagSet("stats score", flag.ContinueOnError)
	historyDays := fs.Int("history", 0, "Number of history days to include (default: 7)")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	params := make(map[string]string)
	if *historyDays > 0 {
		params["history_days"] = fmt.Sprintf("%d", *historyDays)
	}

	body, err := a.core.APIClient.Get(a.apiPath("/stats/score"), toURLValues(params))
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp scoreResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	// Display current score
	fmt.Printf("Lifestyle Score: %d/100\n", resp.Current.Score)
	fmt.Printf("Date: %s\n", resp.Current.Date)
	fmt.Printf("Trend: %s", resp.Current.Trend)
	if resp.Current.ChangeFromYesterday != 0 {
		if resp.Current.ChangeFromYesterday > 0 {
			fmt.Printf(" (+%d)", resp.Current.ChangeFromYesterday)
		} else {
			fmt.Printf(" (%d)", resp.Current.ChangeFromYesterday)
		}
	}
	fmt.Println()
	fmt.Printf("Data Quality: %s\n", resp.Current.DataQuality)
	fmt.Printf("Message: %s\n", resp.Current.Message)

	// Display domain breakdown
	if len(resp.Current.DomainScores) > 0 {
		fmt.Println("\nDomain Scores:")
		for _, ds := range resp.Current.DomainScores {
			fmt.Printf("  %s: %d/100 (%d events, %.0f%% weight)\n",
				ds.DisplayName, ds.Score, ds.EventCount, ds.Weight*100)
		}
	}

	// Display history if available
	if len(resp.History) > 0 {
		fmt.Println("\nRecent History:")
		for _, h := range resp.History {
			fmt.Printf("  %s: %d\n", h.Date, h.Score)
		}
	}

	return nil
}
