package health

import (
	"encoding/json"
	"fmt"
	"os"

	"visited-tracker/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "visited-tracker"

func Register(core *cliapp.ScenarioApp, campaignID *string) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{
				Name:        "status",
				NeedsAPI:    true,
				Description: "Check API health and coverage",
				Run: func(args []string) error {
					return runStatus(core, campaignID, args)
				},
			},
		},
	}
}

func runStatus(core *cliapp.ScenarioApp, campaignID *string, args []string) error {
	fs, jsonOutput, err := support.ParseFlags("status", args)
	if err != nil {
		return err
	}
	location := fs.String("location", "", "Campaign location")
	tag := fs.String("tag", "", "Campaign tag")
	pattern := fs.String("pattern", "", "File pattern")
	name := fs.String("name", "", "Campaign name")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := core.GetRoot("/health", nil)
	if err != nil {
		return err
	}

	var health support.HealthResponse
	_ = jsonOutput
	if err := json.Unmarshal(body, &health); err != nil {
		return fmt.Errorf("parse health response: %w", err)
	}

	report := cliapp.OperationalReport{
		Status: []string{
			"Service: " + health.Service,
			"Version: " + health.Version,
			"Status: " + health.Status,
			fmt.Sprintf("Ready: %t", health.Readiness),
		},
	}
	if health.Metrics.UptimeSeconds > 0 {
		report.Status = append(report.Status, fmt.Sprintf("Uptime: %ds", int(health.Metrics.UptimeSeconds)))
	}

	resolver := support.Resolver{Core: core, CampaignID: campaignID}
	resolved, resolveErr := resolver.ResolveCampaignID(support.CampaignAutoOptions{
		Location: *location,
		Tag:      *tag,
		Pattern:  *pattern,
		Name:     *name,
	}, true)
	if resolveErr != nil {
		report.Triage = []cliapp.TriageGroup{{
			Heading: "Coverage",
			Items:   []string{"Coverage data not available (no campaigns found)"},
		}}
		report.NextSteps = []string{cliName + " campaigns create --name \"...\" --pattern \"**/*\"", cliName + " campaigns list"}
	} else {
		coverageBody, err := core.Get("/campaigns/"+resolved+"/coverage", nil)
		if err != nil {
			report.Triage = []cliapp.TriageGroup{{
				Heading: "Coverage",
				Items:   []string{"Coverage data not available (campaign request failed)"},
			}}
		} else {
			var coverage support.CoverageResponse
			if err := json.Unmarshal(coverageBody, &coverage); err != nil {
				report.Triage = []cliapp.TriageGroup{{
					Heading: "Coverage",
					Items:   []string{"Coverage data not available (invalid response)"},
				}}
			} else {
				report.Triage = []cliapp.TriageGroup{{
					Heading: "Coverage",
					Items: []string{
						fmt.Sprintf("Campaign ID: %s", resolved),
						fmt.Sprintf("Total files: %d", coverage.TotalFiles),
						fmt.Sprintf("Visited: %d (%.0f%%)", coverage.VisitedFiles, coverage.CoveragePercent),
						fmt.Sprintf("Unvisited: %d", coverage.UnvisitedFiles),
					},
				}}
			}
		}
		report.NextSteps = []string{cliName + " coverage", cliName + " least-visited"}
	}

	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}
