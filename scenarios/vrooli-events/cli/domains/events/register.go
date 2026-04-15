package events

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "vrooli-events"

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Events",
		Commands: []cliapp.Command{
			{Name: "ingest", NeedsAPI: true, Description: "Publish an event to the event bus", Run: func(args []string) error { return runIngest(core, args) }},
			{Name: "query", NeedsAPI: true, Description: "Search events by type/source/correlation_id", Run: func(args []string) error { return runQuery(core, args) }},
			{Name: "subscribe", NeedsAPI: true, Description: "Real-time SSE event listener", Run: func(args []string) error { return runSubscribe(core, args) }},
		},
	}
}

func runIngest(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("ingest", flag.ContinueOnError)
	eventID := fs.String("event-id", "", "unique event ID (required)")
	eventType := fs.String("type", "", "event type (required)")
	source := fs.String("source", "", "source scenario name (required)")
	target := fs.String("target", "", "target scenario name")
	corrID := fs.String("correlation-id", "", "correlation ID for tracing")
	payload := fs.String("payload", "", "JSON payload string or @file.json")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *eventID == "" || *eventType == "" || *source == "" {
		return fmt.Errorf("usage: ingest --event-id ID --type TYPE --source SOURCE [--target TARGET] [--correlation-id CID] [--payload JSON] [--json]")
	}

	envelope := map[string]any{
		"eventId":        *eventID,
		"eventType":      *eventType,
		"sourceScenario": *source,
	}
	if *target != "" {
		envelope["targetScenario"] = *target
	}
	if *corrID != "" {
		envelope["correlationId"] = *corrID
	}
	if *payload != "" {
		payloadString := *payload
		if strings.HasPrefix(payloadString, "@") {
			data, err := os.ReadFile(strings.TrimPrefix(payloadString, "@"))
			if err != nil {
				return fmt.Errorf("read payload file: %w", err)
			}
			payloadString = string(data)
		}
		var parsed any
		if err := json.Unmarshal([]byte(payloadString), &parsed); err != nil {
			return fmt.Errorf("invalid payload JSON: %w", err)
		}
		envelope["payload"] = payloadString
	}

	body, err := core.Request("POST", "/events", nil, envelope)
	if err != nil {
		return err
	}

	var response struct {
		ID      any  `json:"id"`
		DryRun  bool `json:"dry_run"`
		EventID any  `json:"eventId"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.MutationReport{
		Result: []string{"Event accepted", "Event ID: " + *eventID},
		Changes: []string{
			"Type: " + *eventType,
			"Source: " + *source,
		},
		NextCommand: []string{
			cliName + " query --correlation-id " + *corrID,
			cliName + " stats",
		},
	}
	if *target != "" {
		report.Changes = append(report.Changes, "Target: "+*target)
	}
	if response.DryRun {
		report.Result = []string{"Dry run accepted", "Event ID: " + *eventID}
	} else if response.ID != nil {
		report.Changes = append(report.Changes, fmt.Sprintf("Store ID: %v", response.ID))
	}
	if *corrID == "" {
		report.NextCommand[0] = cliName + " query --type " + *eventType + " --source " + *source
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runQuery(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("query", flag.ContinueOnError)
	typeFilter := fs.String("type", "", "event type glob pattern")
	source := fs.String("source", "", "source scenario (exact)")
	corrID := fs.String("correlation-id", "", "correlation ID (exact)")
	since := fs.String("since", "", "return events after this ID")
	limit := fs.String("limit", "", "max results")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if *typeFilter != "" {
		query.Set("type", *typeFilter)
	}
	if *source != "" {
		query.Set("source", *source)
	}
	if *corrID != "" {
		query.Set("correlation_id", *corrID)
	}
	if *since != "" {
		query.Set("since", *since)
	}
	if *limit != "" {
		query.Set("limit", *limit)
	}

	body, err := core.Get("/events", query)
	if err != nil {
		return err
	}

	var events []map[string]any
	if err := json.Unmarshal(body, &events); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	summary := []string{fmt.Sprintf("Events returned: %d", len(events))}
	if *typeFilter != "" {
		summary = append(summary, "Type filter: "+*typeFilter)
	}
	if *source != "" {
		summary = append(summary, "Source filter: "+*source)
	}
	report := cliapp.ListReport{
		Summary: summary,
		Results: renderEventRows(events),
		RetrievalHints: []string{
			cliName + " query --limit 20",
			cliName + " subscribe --type " + defaultHint(*typeFilter, "app.*"),
		},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runSubscribe(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("subscribe", flag.ContinueOnError)
	typeFilter := fs.String("type", "", "event type glob pattern")
	source := fs.String("source", "", "source scenario pattern")
	target := fs.String("target", "", "target scenario pattern")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if *typeFilter != "" {
		query.Set("type", *typeFilter)
	}
	if *source != "" {
		query.Set("source", *source)
	}
	if *target != "" {
		query.Set("target", *target)
	}

	endpoint := strings.TrimRight(core.APIBase(), "/") + "/events/subscribe"
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	resp, err := http.Get(endpoint)
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error (%d): %s", resp.StatusCode, body)
	}

	fmt.Fprintln(os.Stderr, "Connected. Listening for events...")

	scanner := bufio.NewScanner(resp.Body)
	var currentEvent string
	var currentData string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, ":") {
			if !*jsonOut {
				fmt.Println(line)
			}
			continue
		}
		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
			continue
		}
		if strings.HasPrefix(line, "data: ") {
			currentData = strings.TrimPrefix(line, "data: ")
			continue
		}
		if line == "" && currentEvent != "" {
			if *jsonOut {
				fmt.Println(currentData)
			} else {
				fmt.Printf("[%s] %s\n", currentEvent, currentData)
			}
			currentEvent = ""
			currentData = ""
		}
	}
	return scanner.Err()
}

func renderEventRows(events []map[string]any) []string {
	lines := make([]string, 0, len(events))
	for _, event := range events {
		lines = append(lines, fmt.Sprintf("%v  %v  %v -> %v  corr=%v",
			value(event, "eventId"),
			value(event, "eventType"),
			value(event, "sourceScenario"),
			value(event, "targetScenario"),
			value(event, "correlationId"),
		))
	}
	return lines
}

func value(m map[string]any, key string) any {
	if value, ok := m[key]; ok {
		return value
	}
	return ""
}

func defaultHint(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
