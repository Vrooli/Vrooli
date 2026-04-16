package ai

import (
	"fmt"
	"os"

	"web-console/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `ai` subcommand group for AI generation, suggestion,
// provider config, and provider health surfaces under /ai.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "ai",
		Description: "AI command generation, suggestions, provider config, and health",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "generate", Description: "Generate a shell command (--body-file PATH)", Run: func(args []string) error { return runGenerate(core, args) }},
			{Name: "suggest", Description: "Get AI suggestions (--body-file PATH)", Run: func(args []string) error { return runSuggest(core, args) }},
			{Name: "config-get", Aliases: []string{"config"}, Description: "Show AI provider config", Run: func(args []string) error { return runConfigGet(core, args) }},
			{Name: "config-set", Description: "Update AI provider config (--body-file PATH)", Run: func(args []string) error { return runConfigSet(core, args) }},
			{Name: "health", Description: "Check configured AI providers", Run: func(args []string) error { return runHealth(core, args) }},
		},
	}
}

func runGenerate(core *cliapp.ScenarioApp, args []string) error {
	return runBodyMutation(core, args, "ai generate", "POST", "/ai/generate", "Generated command")
}

func runSuggest(core *cliapp.ScenarioApp, args []string) error {
	return runBodyMutation(core, args, "ai suggest", "POST", "/ai/suggest", "AI suggestion")
}

func runBodyMutation(core *cliapp.ScenarioApp, args []string, name, method, path, heading string) error {
	fs := support.NewFlagSet(name)
	bodyFile := fs.String("body-file", "", "Path to a JSON request body (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	body, err := core.Request(method, path, nil, payload)
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	_ = support.Decode(body, &resp)

	report := cliapp.ListReport{
		Summary:        []string{heading},
		ResultsHeading: "Response",
		Results:        support.MapRows(resp),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runConfigGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("ai config-get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/ai/config", nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"AI provider config"},
		ResultsHeading: "Values",
		Results:        support.MapRows(payload),
		RetrievalHints: []string{fmt.Sprintf("%s ai config-set --body-file config.json", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runConfigSet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("ai config-set")
	bodyFile := fs.String("body-file", "", "Path to JSON body with provider config (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	if _, err := core.Request("PUT", "/ai/config", nil, payload); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Updated AI provider config"},
		NextCommand: []string{fmt.Sprintf("%s ai config-get", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runHealth(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("ai health")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/ai/health", nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	status := "unknown"
	if v, ok := payload["status"].(string); ok && v != "" {
		status = v
	}

	report := cliapp.OperationalReport{
		Status:    []string{fmt.Sprintf("AI provider status: %s", status)},
		Triage:    []cliapp.TriageGroup{{Heading: "Findings", Items: support.MapRows(payload)}},
		NextSteps: []string{fmt.Sprintf("%s ai config-get", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}
