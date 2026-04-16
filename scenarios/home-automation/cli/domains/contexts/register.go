package contexts

import (
	"fmt"
	"os"
	"strings"
	"time"

	"home-automation/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "contexts",
		Description: "Inspect and drive schedule-aware home contexts",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "current", Description: "Show the active context", Run: func(args []string) error { return runCurrent(core, args) }},
			{Name: "activate", Description: "Activate one context", Run: func(args []string) error { return runActivate(core, args, true) }},
			{Name: "deactivate", Description: "Deactivate one context", Run: func(args []string) error { return runActivate(core, args, false) }},
			{Name: "trigger", Description: "Simulate a calendar event and let the scheduler detect context", Run: func(args []string) error { return runTrigger(core, args) }},
		},
	}
}

func runCurrent(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("contexts current")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/contexts/current", nil)
	if err != nil {
		return err
	}

	var response support.CurrentContext
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Active context: %s", firstNonEmpty(response.ContextName, "unknown")),
			fmt.Sprintf("Active since: %s", support.HumanTime(response.ActiveSince)),
			fmt.Sprintf("Triggered by: %s", firstNonEmpty(response.TriggeredBy, "unknown")),
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Configuration", Items: []string{
				fmt.Sprintf("Scene ID: %s", firstNonEmpty(response.Configuration.SceneID, "none")),
				"Description: " + firstNonEmpty(response.Configuration.Description, "none"),
				fmt.Sprintf("Automation overrides: %d", len(response.Configuration.AutomationOverrides)),
			}},
			{Heading: "Active Devices", Items: []string{support.FormatMapInline(response.ActiveDevices)}},
		},
		NextSteps: []string{
			fmt.Sprintf("home-automation contexts deactivate %s", firstNonEmpty(response.ContextName, "default_mode")),
			"home-automation contexts trigger --title \"Meeting\"",
		},
	}
	if *jsonOutput {
		return support.PrintJSONReport(true, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func runActivate(core *cliapp.ScenarioApp, args []string, active bool) error {
	name := "activate"
	status := "activated"
	if !active {
		name = "deactivate"
		status = "deactivated"
	}

	fs := support.NewFlagSet("contexts " + name)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: home-automation contexts %s <context> [--json]", name)
	}

	contextName := strings.TrimSpace(fs.Arg(0))
	methodPath := "/contexts/" + contextName + "/" + name
	body, err := core.Request("POST", methodPath, nil, map[string]interface{}{})
	if err != nil {
		return err
	}

	var response struct {
		Success bool   `json:"success"`
		Context string `json:"context"`
		Status  string `json:"status"`
	}
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Context change success: %t", response.Success),
			fmt.Sprintf("Context: %s", firstNonEmpty(response.Context, contextName)),
			fmt.Sprintf("Status: %s", firstNonEmpty(response.Status, status)),
		},
		Changes: []string{
			fmt.Sprintf("Scheduler marked context %s.", status),
		},
		NextCommand: []string{
			"home-automation contexts current",
			fmt.Sprintf("home-automation contexts %s %s", opposite(active), firstNonEmpty(response.Context, contextName)),
		},
	}
	if *jsonOutput {
		return support.PrintJSONReport(true, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runTrigger(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("contexts trigger")
	eventID := fs.String("event-id", "", "Calendar event ID")
	eventType := fs.String("event-type", "starting", "Event type: starting, ending, or updated")
	title := fs.String("title", "", "Calendar title used for context detection")
	description := fs.String("description", "", "Calendar description")
	startTime := fs.String("start", time.Now().UTC().Format(time.RFC3339), "Event start time")
	endTime := fs.String("end", time.Now().UTC().Add(time.Hour).Format(time.RFC3339), "Event end time")
	location := fs.String("location", "", "Event location")
	priority := fs.String("priority", "", "Event priority")
	jsonOutput := cliutil.JSONFlag(fs)
	var participants cliutil.StringList
	fs.Var(&participants, "participant", "Participant name or ID (repeatable)")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*title) == "" {
		return fmt.Errorf("usage: home-automation contexts trigger --title <text> [--event-id id] [--event-type starting|ending|updated] [--start ts] [--end ts] [--participant name] [--json]")
	}
	if strings.TrimSpace(*eventID) == "" {
		*eventID = fmt.Sprintf("cli-%d", time.Now().Unix())
	}

	body, err := core.Request("POST", "/calendar/trigger", nil, map[string]interface{}{
		"event_id":   strings.TrimSpace(*eventID),
		"event_type": strings.TrimSpace(*eventType),
		"event_data": map[string]interface{}{
			"title":        strings.TrimSpace(*title),
			"description":  strings.TrimSpace(*description),
			"start_time":   strings.TrimSpace(*startTime),
			"end_time":     strings.TrimSpace(*endTime),
			"participants": normalizeList(participants.Values()),
			"location":     strings.TrimSpace(*location),
			"priority":     strings.TrimSpace(*priority),
		},
	})
	if err != nil {
		return err
	}

	var response support.CalendarEventResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	changeRows := make([]string, 0, len(response.DeviceChanges))
	for _, change := range response.DeviceChanges {
		changeRows = append(changeRows, fmt.Sprintf("%s | action=%s | success=%t | params=%s | %s", change.DeviceID, change.Action, change.Success, support.FormatMapInline(change.Parameters), firstNonEmpty(change.Message, "no message")))
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Calendar trigger success: %t", response.Success),
			fmt.Sprintf("Event: %s", response.EventID),
			fmt.Sprintf("Detected context: %s", firstNonEmpty(response.DetectedContext, "unknown")),
			fmt.Sprintf("Context activated: %t", response.ContextActivated),
		},
		Changes: []string{
			"Message: " + firstNonEmpty(response.Message, "No message returned"),
			"Device changes: " + joinOrNone(changeRows),
			fmt.Sprintf("Processed at: %s", support.HumanTime(response.ProcessingTimestamp)),
		},
		NextCommand: []string{
			"home-automation contexts current",
			fmt.Sprintf("home-automation contexts deactivate %s", firstNonEmpty(response.DetectedContext, "default_mode")),
		},
	}
	if *jsonOutput {
		return support.PrintJSONReport(true, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func normalizeList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "(none)"
	}
	return strings.Join(values, "; ")
}

func opposite(active bool) string {
	if active {
		return "deactivate"
	}
	return "activate"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
