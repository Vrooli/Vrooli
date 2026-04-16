package ai

import (
	"fmt"
	"os"
	"strings"

	"react-component-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `ai` subcommand group exposing the chat and refactor
// endpoints backed by OpenRouter.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "ai",
		Description: "AI-assisted chat and code refactoring",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "chat", Description: "Send a chat message to the AI assistant", Run: func(args []string) error { return runChat(core, args) }},
			{Name: "refactor", Description: "Refactor a code snippet via the AI service", Run: func(args []string) error { return runRefactor(core, args) }},
		},
	}
}

func runChat(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("ai chat")
	message := fs.String("message", "", "Prompt to send (required unless --body-file is provided)")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with {message, context} overriding inline flags")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if strings.TrimSpace(*message) == "" {
			return fmt.Errorf("--message or --body-file is required")
		}
		payload = map[string]interface{}{"message": *message}
	}

	body, err := core.Request("POST", "/ai/chat", nil, payload)
	if err != nil {
		return err
	}
	var resp support.AIResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := []string{fmt.Sprintf("Response: %s", resp.Response)}
	if len(resp.Suggestions) > 0 {
		results = append(results, "Suggestions:")
		for _, s := range resp.Suggestions {
			results = append(results, "- "+s)
		}
	}

	report := cliapp.ListReport{
		Summary:        []string{"AI chat response"},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("%s ai refactor --body-file ./refactor.json", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runRefactor(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("ai refactor")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with {code, instruction} (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/ai/refactor", nil, payload)
	if err != nil {
		return err
	}
	var resp support.AIRefactorResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := []string{
		"Refactored code:",
		resp.RefactoredCode,
	}
	if strings.TrimSpace(resp.Diff) != "" {
		results = append(results, "", "Diff:", resp.Diff)
	}

	report := cliapp.ListReport{
		Summary:        []string{"AI refactor result"},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("%s ai chat --message 'Explain the refactor above'", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
