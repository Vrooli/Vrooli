package recovery

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"tunnel-manager/cli/internal/flags"
	"tunnel-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "recovery",
		Description: "Inspect and drive tunnel recovery workflows",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "state", NeedsAPI: true, Description: "Show current recovery engine state", Run: func(args []string) error { return state(deps, args) }},
			{Name: "trigger", NeedsAPI: true, Description: "Trigger a recovery action", Run: func(args []string) error { return trigger(deps, args) }},
			{Name: "events", NeedsAPI: true, Description: "List recent recovery events", Run: func(args []string) error { return events(deps, args) }},
			{Name: "circuit-reset", NeedsAPI: true, Description: "Reset the recovery circuit breaker", Run: func(args []string) error { return circuitReset(deps, args) }},
		},
	}
}

func state(deps support.Dependencies, args []string) error {
	body, err := deps.ScenarioApp().Get("/recovery/state", nil)
	if err != nil {
		return err
	}
	if flags.HasJSONOutput(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var parsed struct {
		Status         string `json:"status"`
		Failures       int    `json:"failures"`
		BackoffSeconds int    `json:"backoff_seconds"`
		CircuitState   string `json:"circuit_state"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{
		Status: []string{
			"Recovery status: " + parsed.Status,
			fmt.Sprintf("Circuit: %s", parsed.CircuitState),
		},
		Triage: []cliapp.TriageGroup{
			{
				Heading: "Backoff",
				Items: []string{
					fmt.Sprintf("Failures: %d", parsed.Failures),
					fmt.Sprintf("Backoff: %ds", parsed.BackoffSeconds),
				},
			},
		},
		NextSteps: []string{
			"tunnel-manager recovery events",
			"tunnel-manager recovery trigger --force",
		},
	})
}

func trigger(deps support.Dependencies, args []string) error {
	body, err := deps.ScenarioApp().Request("POST", "/recovery/trigger", nil, map[string]any{
		"force": flags.BoolValue(args, "force"),
	})
	if err != nil {
		return err
	}
	if flags.HasJSONOutput(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	changes := make([]string, 0, len(result))
	for key, value := range result {
		changes = append(changes, fmt.Sprintf("%s: %v", key, value))
	}

	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result:      []string{"Triggered recovery action"},
		Changes:     changes,
		NextCommand: []string{"tunnel-manager recovery state", "tunnel-manager recovery events"},
	})
}

func events(deps support.Dependencies, args []string) error {
	limit := "50"
	if v, ok := flags.StringValue(args, "limit"); ok {
		limit = v
	}

	body, err := deps.ScenarioApp().Get("/recovery/events", urlValues("limit", limit))
	if err != nil {
		return err
	}
	if flags.HasJSONOutput(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var events []struct {
		Timestamp string `json:"timestamp"`
		Trigger   string `json:"trigger"`
		Action    string `json:"action"`
		Outcome   string `json:"outcome"`
		Details   string `json:"details,omitempty"`
	}
	if err := json.Unmarshal(body, &events); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	results := make([]string, 0, len(events))
	for _, event := range events {
		line := fmt.Sprintf("%s | %s | %s | %s", event.Timestamp, event.Trigger, event.Action, event.Outcome)
		if event.Details != "" {
			line += " | " + event.Details
		}
		results = append(results, line)
	}

	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Recovery events: %d", len(events)),
		},
		Results:        results,
		RetrievalHints: []string{"tunnel-manager recovery state", "tunnel-manager recovery trigger --force"},
	})
}

func circuitReset(deps support.Dependencies, args []string) error {
	body, err := deps.ScenarioApp().Request("POST", "/recovery/circuit/reset", nil, nil)
	if err != nil {
		return err
	}
	if flags.HasJSONOutput(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result:      []string{"Recovery circuit breaker reset"},
		NextCommand: []string{"tunnel-manager recovery state"},
	})
}

func urlValues(key, value string) url.Values {
	if value == "" {
		return nil
	}
	query := url.Values{}
	query.Set(key, value)
	return query
}
