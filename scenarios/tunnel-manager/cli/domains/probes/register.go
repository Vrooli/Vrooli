package probes

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

type probeResponse struct {
	Results []struct {
		Subdomain  string `json:"subdomain"`
		ProbeType  string `json:"probe_type"`
		Status     string `json:"status"`
		LatencyMs  int    `json:"latency_ms"`
		StatusCode int    `json:"status_code,omitempty"`
		ErrorMsg   string `json:"error_msg,omitempty"`
	} `json:"results"`
	Summary struct {
		Total int `json:"total"`
		Up    int `json:"up"`
		Down  int `json:"down"`
	} `json:"summary"`
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "probe",
		Description: "Run live probes and inspect probe history",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "run", NeedsAPI: true, Description: "Run all internal and external probes", Run: func(args []string) error { return run(deps, args) }},
			{Name: "history", NeedsAPI: true, Description: "Show probe history", Run: func(args []string) error { return history(deps, args) }},
		},
	}
}

func run(deps support.Dependencies, args []string) error {
	body, err := deps.ScenarioApp().Request("POST", "/probes", nil, nil)
	if err != nil {
		return err
	}
	if flags.HasJSONOutput(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp probeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Routes probed: %d", resp.Summary.Total),
			fmt.Sprintf("Up: %d", resp.Summary.Up),
			fmt.Sprintf("Down: %d", resp.Summary.Down),
		},
		Results:        formatProbeResults(resp.Results),
		RetrievalHints: []string{"tunnel-manager probe history", "tunnel-manager health detailed"},
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func history(deps support.Dependencies, args []string) error {
	limit := "100"
	if v, ok := flags.StringValue(args, "limit"); ok {
		limit = v
	}

	body, err := deps.ScenarioApp().Get("/probes/history", urlValues("limit", limit))
	if err != nil {
		return err
	}
	if flags.HasJSONOutput(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var entries []struct {
		Timestamp string `json:"timestamp"`
		Route     string `json:"route"`
		ProbeType string `json:"probe_type"`
		Status    string `json:"status"`
		LatencyMs int    `json:"latency_ms"`
		ErrorMsg  string `json:"error_msg,omitempty"`
	}
	if err := json.Unmarshal(body, &entries); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Probe history entries: %d", len(entries)),
		},
		Results:        formatHistory(entries),
		RetrievalHints: []string{"tunnel-manager probe run", "tunnel-manager route list"},
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func formatProbeResults(results []struct {
	Subdomain  string `json:"subdomain"`
	ProbeType  string `json:"probe_type"`
	Status     string `json:"status"`
	LatencyMs  int    `json:"latency_ms"`
	StatusCode int    `json:"status_code,omitempty"`
	ErrorMsg   string `json:"error_msg,omitempty"`
},
) []string {
	if len(results) == 0 {
		return []string{"No routes to probe."}
	}
	lines := make([]string, 0, len(results))
	for _, result := range results {
		line := fmt.Sprintf("%s | %s | %s", result.Subdomain, result.ProbeType, result.Status)
		if result.LatencyMs > 0 {
			line += fmt.Sprintf(" | %dms", result.LatencyMs)
		}
		if result.StatusCode > 0 {
			line += fmt.Sprintf(" | HTTP %d", result.StatusCode)
		}
		if result.ErrorMsg != "" {
			line += " | " + result.ErrorMsg
		}
		lines = append(lines, line)
	}
	return lines
}

func formatHistory(entries []struct {
	Timestamp string `json:"timestamp"`
	Route     string `json:"route"`
	ProbeType string `json:"probe_type"`
	Status    string `json:"status"`
	LatencyMs int    `json:"latency_ms"`
	ErrorMsg  string `json:"error_msg,omitempty"`
},
) []string {
	if len(entries) == 0 {
		return []string{"No probe history."}
	}
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		line := fmt.Sprintf("%s | %s | %s | %s", entry.Timestamp, entry.Route, entry.ProbeType, entry.Status)
		if entry.LatencyMs > 0 {
			line += fmt.Sprintf(" | %dms", entry.LatencyMs)
		}
		if entry.ErrorMsg != "" {
			line += " | " + entry.ErrorMsg
		}
		lines = append(lines, line)
	}
	return lines
}

func urlValues(key, value string) url.Values {
	if value == "" {
		return nil
	}
	query := url.Values{}
	query.Set(key, value)
	return query
}
