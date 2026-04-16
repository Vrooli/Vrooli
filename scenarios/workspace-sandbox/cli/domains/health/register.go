package health

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"workspace-sandbox/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Check API health and driver status", Run: func(args []string) error { return runStatus(deps, args) }},
		},
	}
}

func runStatus(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := deps.ScenarioApp().Get("/health", nil)
	if err != nil {
		return err
	}

	var parsed support.HealthResponse
	if err := json.Unmarshal(body, &parsed); err != nil || parsed.Status == "" {
		if *jsonOut {
			cliutil.PrintJSON(body)
			return nil
		}
		return fmt.Errorf("failed to parse health response")
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Service: %s", firstNonEmpty(parsed.Service, "workspace-sandbox")),
			fmt.Sprintf("Status: %s", parsed.Status),
			fmt.Sprintf("Ready: %t", parsed.Readiness),
		},
		Triage: []cliapp.TriageGroup{},
		NextSteps: []string{
			support.CLIName + " sandbox list",
			support.CLIName + " maintenance driver",
			support.CLIName + " process list <sandbox-id>",
		},
	}
	if parsed.Version != "" {
		report.Status = append(report.Status, "Version: "+parsed.Version)
	}
	if parsed.Timestamp != "" {
		report.Status = append(report.Status, "Timestamp: "+parsed.Timestamp)
	}
	generalItems := make([]string, 0, 2)
	if parsed.Message != "" {
		generalItems = append(generalItems, "Message: "+parsed.Message)
	}
	if parsed.Error != "" {
		generalItems = append(generalItems, "Error: "+parsed.Error)
	}
	if len(generalItems) > 0 {
		report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: "General", Items: generalItems})
	}

	if len(parsed.Deps) > 0 {
		keys := make([]string, 0, len(parsed.Deps))
		for key := range parsed.Deps {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		items := make([]string, 0, len(keys))
		for _, key := range keys {
			items = append(items, fmt.Sprintf("%s: %s", key, parsed.Deps[key]))
		}
		if len(items) > 0 {
			report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: "Dependencies", Items: items})
		}
	}
	operationItems := support.SortedMapLines(parsed.Operations)
	if len(operationItems) > 0 {
		report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: "Operations", Items: operationItems})
	}

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
