package status

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

type statusReport struct {
	cliapp.OperationalReport
	API struct {
		Running bool   `json:"running"`
		URL     string `json:"url"`
	} `json:"api"`
	Workflows struct {
		Count int `json:"count"`
	} `json:"workflows"`
}

type workflowListResponse struct {
	Workflows []struct {
		ID string `json:"id"`
	} `json:"workflows"`
}

func Commands(ctx *appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{
				Name:        "status",
				NeedsAPI:    false,
				Description: "Show operational status and resource health",
				Run: func(args []string) error {
					return runStatus(ctx, args)
				},
			},
		},
	}
}

func runStatus(ctx *appctx.Context, args []string) error {
	jsonOutput := false
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		default:
			return fmt.Errorf("unknown option: %s", arg)
		}
	}

	report := statusReport{}
	report.API.URL = ctx.Core.APIBase()
	report.API.Running = apiRunning(ctx)
	report.Workflows.Count = workflowCount(ctx, report.API.Running)
	report.OperationalReport = buildOperationalReport(report)

	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report.OperationalReport)
}

func apiRunning(ctx *appctx.Context) bool {
	if ctx == nil || ctx.Core == nil {
		return false
	}
	_, err := ctx.Core.GetRoot(ctx.Core.HealthPath(), nil)
	return err == nil
}

func workflowCount(ctx *appctx.Context, apiRunning bool) int {
	if !apiRunning || ctx == nil || ctx.Core == nil {
		return 0
	}
	body, err := ctx.Core.Get("/workflows", nil)
	if err != nil {
		return 0
	}
	var parsed workflowListResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0
	}
	return len(parsed.Workflows)
}

func buildOperationalReport(report statusReport) cliapp.OperationalReport {
	statusLines := []string{
		fmt.Sprintf("API: %s", statusLabel(report.API.Running, "running", "down")),
		fmt.Sprintf("Workflows indexed: %d", report.Workflows.Count),
	}
	if strings.TrimSpace(report.API.URL) != "" {
		statusLines = append(statusLines, fmt.Sprintf("API URL: %s", report.API.URL))
	}

	triage := []cliapp.TriageGroup{
		{
			Heading: "API",
			Items:   apiTriage(report),
		},
	}

	return cliapp.OperationalReport{
		Status:    statusLines,
		Triage:    triage,
		NextSteps: nextSteps(report),
	}
}

func apiTriage(report statusReport) []string {
	if report.API.Running {
		return []string{"Health endpoint responded successfully."}
	}
	items := []string{"API health endpoint is not responding."}
	if strings.TrimSpace(report.API.URL) == "" {
		items = append(items, "No API base is configured for the CLI.")
	}
	return items
}

func nextSteps(report statusReport) []string {
	steps := []string{}
	if !report.API.Running {
		steps = append(steps, "cd scenarios/browser-automation-studio && make start")
		steps = append(steps, "browser-automation-studio configure api_base <url>")
	}
	if report.API.Running {
		steps = append(steps, "browser-automation-studio workflow list")
	}
	return steps
}

func statusLabel(ok bool, whenOK, whenBad string) string {
	if ok {
		return whenOK
	}
	return whenBad
}
