package overview

import (
	"fmt"

	"secrets-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type statusPayload struct {
	Health     support.HealthResponse           `json:"health"`
	Coverage   support.CredentialCoverageStatus `json:"credential_coverage"`
	Compliance support.ComplianceResponse       `json:"compliance"`
}

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Overview",
		Commands: []cliapp.Command{
			{
				Name:        "status",
				NeedsAPI:    true,
				Description: "Show health, credential coverage, and compliance posture",
				Run: func(args []string) error {
					return runStatus(core, args)
				},
			},
			core.StandardStatusCommand(cliapp.StatusCommandOptions{
				Name:        "health",
				Description: "Check API health and dependency readiness",
			}),
		},
	}
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload statusPayload
	if err := support.GetRootJSON(core, "/health", nil, &payload.Health); err != nil {
		return err
	}
	if err := support.GetJSON(core, "/credentials/secrets/status", nil, &payload.Coverage); err != nil {
		return err
	}
	if err := support.GetJSON(core, "/security/compliance", nil, &payload.Compliance); err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Service: %s", payload.Health.Service),
			fmt.Sprintf("Health: %s", payload.Health.Status),
			fmt.Sprintf("Readiness: %s", support.BoolLabel(payload.Health.Readiness, "ready", "not ready")),
			fmt.Sprintf("Overall compliance: %d", payload.Compliance.OverallScore),
			fmt.Sprintf("Credential coverage: %d/%d resources configured", payload.Coverage.ConfiguredResources, payload.Coverage.TotalResources),
			fmt.Sprintf("Total vulnerabilities: %d", payload.Compliance.TotalVulnerabilities),
		},
		Triage: []cliapp.TriageGroup{
			{
				Heading: "Database",
				Items: []string{
					fmt.Sprintf("Connected: %s", support.BoolLabel(payload.Health.Dependencies.Database.Connected, "yes", "no")),
					fmt.Sprintf("Latency: %.0fms", payload.Health.Dependencies.Database.LatencyMS),
				},
			},
			{
				Heading: "Credential Coverage",
				Items: []string{
					fmt.Sprintf("Missing credentials: %d", len(payload.Coverage.MissingSecrets)),
					fmt.Sprintf("Last updated: %s", support.FormatTime(payload.Coverage.LastUpdated)),
				},
			},
			{
				Heading: "Security",
				Items: []string{
					fmt.Sprintf("Security score: %d", payload.Compliance.RemediationProgress.SecurityScore),
					fmt.Sprintf("Critical issues: %d", payload.Compliance.RemediationProgress.CriticalIssues),
					fmt.Sprintf("High issues: %d", payload.Compliance.RemediationProgress.HighIssues),
				},
			},
		},
		NextSteps: []string{
			support.CLIName + " credentials status",
			support.CLIName + " security vulnerabilities --severity critical",
			support.CLIName + " deployment readiness --scenario <scenario>",
		},
	}
	if payload.Health.Dependencies.Database.Error != nil && payload.Health.Dependencies.Database.Error.Message != "" {
		report.Triage[0].Items = append(report.Triage[0].Items, "Error: "+payload.Health.Dependencies.Database.Error.Message)
	}
	if len(payload.Health.StatusNotes) > 0 {
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Status Notes",
			Items:   payload.Health.StatusNotes,
		})
	}

	return support.PrintOperational(*jsonOutput, payload, report)
}
