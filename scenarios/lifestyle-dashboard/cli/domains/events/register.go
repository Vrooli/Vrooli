package events

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"lifestyle-dashboard/cli/internal/query"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "lifestyle-dashboard"

type CreateEventRequest struct {
	Domain         string          `json:"domain"`
	EventType      string          `json:"event_type"`
	Payload        json.RawMessage `json:"payload,omitempty"`
	Timestamp      *string         `json:"timestamp,omitempty"`
	IsIntervention bool            `json:"is_intervention"`
	HypothesisID   *string         `json:"hypothesis_id,omitempty"`
}

type EventResponse struct {
	ID             string          `json:"id"`
	Domain         string          `json:"domain"`
	EventType      string          `json:"event_type"`
	Timestamp      string          `json:"timestamp"`
	Payload        json.RawMessage `json:"payload"`
	IsIntervention bool            `json:"is_intervention"`
	HypothesisID   *string         `json:"hypothesis_id,omitempty"`
	CreatedAt      string          `json:"created_at"`
}

type EventsListResponse struct {
	Events []EventResponse `json:"events"`
	Count  int             `json:"count"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "event",
		Description: "Create, list, and inspect lifestyle events",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "create", NeedsAPI: true, Description: "Create a new event", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "list", NeedsAPI: true, Description: "List events with optional filters", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", NeedsAPI: true, Description: "Get event by ID", Run: func(args []string) error { return runGet(core, args) }},
		},
	}
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
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

	req := CreateEventRequest{
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

	body, err := core.Request("POST", "/events", nil, req)
	if err != nil {
		return err
	}
	var resp EventResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.MutationReport{
		Result: []string{
			"Event created",
			"Event ID: " + resp.ID,
		},
		Changes: []string{
			"Domain: " + resp.Domain,
			"Type: " + resp.EventType,
			"Timestamp: " + resp.Timestamp,
			fmt.Sprintf("Intervention: %v", resp.IsIntervention),
		},
		NextCommand: []string{
			cliName + " event get " + resp.ID,
			cliName + " stats summary",
		},
	}
	if resp.HypothesisID != nil {
		report.Changes = append(report.Changes, "Hypothesis: "+*resp.HypothesisID)
	}
	if len(resp.Payload) > 0 && string(resp.Payload) != "null" {
		report.Changes = append(report.Changes, "Payload: "+string(resp.Payload))
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runList(core *cliapp.ScenarioApp, args []string) error {
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

	params := map[string]string{}
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

	body, err := core.Get("/events", query.ToURLValues(params))
	if err != nil {
		return err
	}
	var resp EventsListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Events: %d", resp.Count),
		},
		Results:        renderEventRows(resp.Events),
		RetrievalHints: []string{cliName + " event get <event-id>", cliName + " stats timeline"},
	}
	if *domain != "" {
		report.Summary = append(report.Summary, "Domain filter: "+*domain)
	}
	if *eventType != "" {
		report.Summary = append(report.Summary, "Type filter: "+*eventType)
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("event get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: event get <id> [--json]")
	}

	id := fs.Arg(0)
	body, err := core.Get("/events/"+id, nil)
	if err != nil {
		return err
	}
	var resp EventResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.ListReport{
		Summary:        []string{"Event: " + resp.ID, "Domain: " + resp.Domain},
		ResultsHeading: "Details",
		Results: []string{
			"Type: " + resp.EventType,
			"Timestamp: " + resp.Timestamp,
			fmt.Sprintf("Intervention: %v", resp.IsIntervention),
		},
		RetrievalHints: []string{cliName + " event list --domain " + resp.Domain, cliName + " stats summary"},
	}
	if resp.HypothesisID != nil {
		report.Results = append(report.Results, "Hypothesis: "+*resp.HypothesisID)
	}
	if len(resp.Payload) > 0 && string(resp.Payload) != "null" {
		report.Results = append(report.Results, "Payload: "+compactPayload(resp.Payload))
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func renderEventRows(events []EventResponse) []string {
	if len(events) == 0 {
		return nil
	}
	rows := make([]string, 0, len(events))
	for _, event := range events {
		day := event.Timestamp
		if len(day) >= 10 {
			day = day[:10]
		}
		rows = append(rows, fmt.Sprintf("%s | %s | %s/%s", shortID(event.ID), day, event.Domain, event.EventType))
	}
	return rows
}

func compactPayload(payload json.RawMessage) string {
	text := strings.TrimSpace(string(payload))
	if len(text) <= 120 {
		return text
	}
	return text[:117] + "..."
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
