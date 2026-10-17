package abtests

import (
	"fmt"
	"os"

	"ai-chatbot-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register wires `ai-chatbot-manager abtest ...` covering create/start/results.
// Create requires --body-file because the request body includes two nested
// variant objects plus a traffic split; --body-file is the cleanest way to
// pass that without hand-assembling JSON in the CLI.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "abtest",
		Description: "Manage A/B tests for chatbots",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "create", Description: "Create an A/B test (--body-file PATH required)", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "start", Description: "Start a draft A/B test", Run: func(args []string) error { return runStart(core, args) }},
			{Name: "results", Description: "Fetch A/B test results", Run: func(args []string) error { return runResults(core, args) }},
		},
	}
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("abtest create")
	bodyFile := fs.String("body-file", "", "Path to A/B test request JSON (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: abtest create <chatbot-id> --body-file PATH")
	}
	chatbotID := fs.Arg(0)
	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/chatbots/"+chatbotID+"/ab-tests", nil, payload)
	if err != nil {
		return err
	}

	var resp struct {
		ABTest  map[string]interface{} `json:"ab_test"`
		Message string                 `json:"message"`
	}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	message := resp.Message
	if message == "" {
		message = "A/B test created"
	}
	result := []string{message}
	result = append(result, support.MapRows(resp.ABTest)...)
	testID, _ := resp.ABTest["id"].(string)

	report := cliapp.MutationReport{
		Result:  result,
		Changes: []string{fmt.Sprintf("Created A/B test %s for chatbot %s", testID, chatbotID)},
		NextCommand: []string{
			fmt.Sprintf("%s abtest start %s", support.CLIName, testID),
			fmt.Sprintf("%s abtest results %s", support.CLIName, testID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runStart(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("abtest start")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: abtest start <test-id>")
	}
	id := fs.Arg(0)

	body, err := core.Request("POST", "/ab-tests/"+id+"/start", nil, nil)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("A/B test %s started", id)
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     []string{fmt.Sprintf("A/B test %s -> running", id)},
		NextCommand: []string{fmt.Sprintf("%s abtest results %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runResults(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("abtest results")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: abtest results <test-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/ab-tests/"+id+"/results", nil)
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := support.MapRows(resp)

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("A/B test %s results", id)},
		ResultsHeading: "Results",
		Results:        results,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
