package audit

import (
	"encoding/json"
	"fmt"
	"os"

	"tunnel-manager/cli/internal/flags"
	"tunnel-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type response struct {
	Results []struct {
		Subdomain    string `json:"subdomain"`
		ScenarioName string `json:"scenario_name"`
		ExpectedPort int    `json:"expected_port"`
		ActualPort   int    `json:"actual_port,omitempty"`
		Status       string `json:"status"`
		Detail       string `json:"detail,omitempty"`
	} `json:"results"`
	Total      int `json:"total"`
	Violations int `json:"violations"`
	Compliant  int `json:"compliant"`
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "audit",
		Description: "Run tunnel-manager compliance audits",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "ports", NeedsAPI: true, Description: "Check port compliance and report violations", Run: func(args []string) error { return run(deps, args) }},
		},
	}
}

func run(deps support.Dependencies, args []string) error {
	body, err := deps.ScenarioApp().Get("/audit/ports", nil)
	if err != nil {
		return err
	}
	if flags.HasJSONOutput(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp response
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Routes audited: %d", resp.Total),
			fmt.Sprintf("Compliant: %d", resp.Compliant),
			fmt.Sprintf("Violations: %d", resp.Violations),
		},
		NextSteps: []string{
			"tunnel-manager route list",
			"tunnel-manager health detailed",
		},
	}
	if len(resp.Results) > 0 {
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Port Findings",
			Items:   formatResults(resp.Results),
		})
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func formatResults(results []struct {
	Subdomain    string `json:"subdomain"`
	ScenarioName string `json:"scenario_name"`
	ExpectedPort int    `json:"expected_port"`
	ActualPort   int    `json:"actual_port,omitempty"`
	Status       string `json:"status"`
	Detail       string `json:"detail,omitempty"`
},
) []string {
	if len(results) == 0 {
		return []string{"No routes to audit."}
	}
	lines := make([]string, 0, len(results))
	for _, result := range results {
		line := fmt.Sprintf("%s | %s | expected %d", result.Subdomain, result.ScenarioName, result.ExpectedPort)
		if result.ActualPort > 0 {
			line += fmt.Sprintf(" | actual %d", result.ActualPort)
		}
		line += " | " + result.Status
		if result.Detail != "" {
			line += " | " + result.Detail
		}
		lines = append(lines, line)
	}
	return lines
}
