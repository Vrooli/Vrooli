package deployment

import (
	"fmt"

	"secrets-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "deployment",
		Description: "Deployment manifest generation and readiness snapshots",
		Subcommands: []cliapp.Command{
			{Name: "plan", Aliases: []string{"export"}, NeedsAPI: true, Description: "Generate a deployment manifest", Run: func(args []string) error { return runPlan(core, args) }},
			{Name: "readiness", NeedsAPI: true, Description: "Show a lightweight deployment readiness snapshot", Run: func(args []string) error { return runReadiness(core, args) }},
		},
	}
}

func runPlan(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deployment plan")
	scenario := fs.String("scenario", "", "Scenario name")
	tier := fs.String("tier", "tier-2-desktop", "Deployment tier")
	includeOptional := fs.Bool("include-optional", false, "Include optional secrets")
	var resources cliutil.StringList
	fs.Var(&resources, "resource", "Filter to a resource; repeatable")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *scenario == "" {
		return fmt.Errorf("--scenario is required")
	}

	payload := map[string]any{
		"scenario":         *scenario,
		"tier":             *tier,
		"resources":        resources.Values(),
		"include_optional": *includeOptional,
	}
	var resp support.DeploymentManifestResponse
	if err := support.RequestJSON(core, "POST", "/deployment/secrets", nil, payload, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Secrets))
	for _, secret := range resp.Secrets {
		results = append(results, fmt.Sprintf("%s/%s | %s | %s | required=%t | prompt=%t",
			secret.ResourceName, secret.SecretKey, secret.Classification, secret.HandlingStrategy, secret.Required, secret.RequiresUserInput))
	}
	for _, item := range resp.Summary.BlockingSecrets {
		results = append(results, "Blocking: "+item)
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenario: %s", resp.Scenario),
			fmt.Sprintf("Tier: %s", resp.Tier),
			fmt.Sprintf("Secrets: %d", resp.Summary.TotalSecrets),
			fmt.Sprintf("Strategized: %d", resp.Summary.StrategizedSecrets),
			fmt.Sprintf("Requires action: %d", resp.Summary.RequiresAction),
		},
		ResultsHeading: "Manifest Secrets",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " deployment readiness --scenario " + resp.Scenario, support.CLIName + " overrides effective " + resp.Scenario + " --tier " + resp.Tier},
	}
	return support.PrintList(*jsonOutput, resp, report)
}

func runReadiness(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deployment readiness")
	scenario := fs.String("scenario", "", "Scenario name")
	tier := fs.String("tier", "tier-2-desktop", "Deployment tier")
	includeOptional := fs.Bool("include-optional", false, "Include optional secrets")
	var resources cliutil.StringList
	fs.Var(&resources, "resource", "Filter to a resource; repeatable")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *scenario == "" {
		return fmt.Errorf("--scenario is required")
	}

	payload := map[string]any{
		"scenario":         *scenario,
		"tier":             *tier,
		"resources":        resources.Values(),
		"include_optional": *includeOptional,
	}
	var resp support.DeploymentReadinessResponse
	if err := support.RequestJSON(core, "POST", "/deployment/readiness", nil, payload, &resp); err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Scenario: %s", resp.Scenario),
			fmt.Sprintf("Tier: %s", resp.Tier),
			fmt.Sprintf("Strategized secrets: %d/%d", resp.Summary.StrategizedSecrets, resp.Summary.TotalSecrets),
			fmt.Sprintf("Requires action: %d", resp.Summary.RequiresAction),
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Blocking Secrets", Items: resp.Summary.BlockingSecrets},
		},
		NextSteps: []string{
			support.CLIName + " deployment plan --scenario " + resp.Scenario + " --tier " + resp.Tier,
			support.CLIName + " overrides effective " + resp.Scenario + " --tier " + resp.Tier,
		},
	}
	return support.PrintOperational(*jsonOutput, resp, report)
}
