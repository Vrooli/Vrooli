package progress

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `progress` subcommand group for onboarding progress
// tracking (get, update, complete). Because updates and completion take
// nested JSON shapes (completed_steps array, config_data object), mutating
// commands accept a --body-file path rather than exposing each nested field
// as a flag.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "progress",
		Description: "Inspect and update onboarding progress",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "get", Aliases: []string{"show"}, Description: "Show onboarding progress for a user", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "update", Aliases: []string{"set"}, Description: "Update onboarding progress (see --body-file)", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "complete", Description: "Mark onboarding complete for a user", Run: func(args []string) error { return runComplete(core, args) }},
		},
	}
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("progress get")
	userID := fs.String("user-id", "", "User ID (defaults to server-side 'default')")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{"user_id": *userID})
	body, err := core.Get("/progress", query)
	if err != nil {
		return err
	}
	var p support.OnboardingProgress
	if err := support.Decode(body, &p); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %d", p.ID),
		fmt.Sprintf("User: %s", p.UserID),
		fmt.Sprintf("Current step: %d", p.CurrentStep),
		fmt.Sprintf("Completed steps: %s", rawOrEmpty(p.CompletedSteps, "[]")),
		fmt.Sprintf("Config data: %s", rawOrEmpty(p.ConfigData, "{}")),
		fmt.Sprintf("Updated at: %s", support.FormatTimeValue(p.UpdatedAt)),
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Progress for %s: step %d", p.UserID, p.CurrentStep)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s progress update --body-file progress.json", support.CLIName),
			fmt.Sprintf("%s progress complete --user-id %s", support.CLIName, p.UserID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("progress update")
	userID := fs.String("user-id", "", "User ID (overrides any user_id in the body)")
	currentStep := fs.Int("current-step", -1, "Current onboarding step (omit to leave unset)")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full progress update body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := buildUpdateBody(*userID, *currentStep, *bodyFile)
	if err != nil {
		return err
	}

	body, err := core.Request("PUT", "/progress", nil, payload)
	if err != nil {
		return err
	}
	var p support.OnboardingProgress
	if err := support.Decode(body, &p); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Progress updated for %s: step %d", p.UserID, p.CurrentStep)},
		Changes: []string{
			fmt.Sprintf("current_step -> %d", p.CurrentStep),
			fmt.Sprintf("updated_at -> %s", support.FormatTimeValue(p.UpdatedAt)),
		},
		NextCommand: []string{
			fmt.Sprintf("%s progress get --user-id %s", support.CLIName, p.UserID),
			fmt.Sprintf("%s progress complete --user-id %s", support.CLIName, p.UserID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runComplete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("progress complete")
	userID := fs.String("user-id", "", "User ID (defaults to server-side 'default')")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload := map[string]string{}
	if strings.TrimSpace(*userID) != "" {
		payload["user_id"] = strings.TrimSpace(*userID)
	}

	body, err := core.Request("POST", "/complete", nil, payload)
	if err != nil {
		return err
	}
	var resp support.CompleteResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Onboarding complete for %s", resp.UserID)},
		Changes: []string{
			fmt.Sprintf("completed_at -> %s", support.FormatTime(resp.CompletedAt)),
			fmt.Sprintf("config_path -> %s", resp.ConfigPath),
		},
		NextCommand: []string{
			fmt.Sprintf("%s progress get --user-id %s", support.CLIName, resp.UserID),
			fmt.Sprintf("%s resources health", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

// buildUpdateBody composes the PUT /progress body. --body-file supplies the
// nested completed_steps/config_data; --user-id and --current-step override
// any values in the file when provided.
func buildUpdateBody(userID string, currentStep int, bodyFile string) (map[string]json.RawMessage, error) {
	payload := map[string]json.RawMessage{}
	if strings.TrimSpace(bodyFile) != "" {
		raw, err := support.ReadJSONFile(bodyFile, true)
		if err != nil {
			return nil, err
		}
		var fileBody map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fileBody); err != nil {
			return nil, fmt.Errorf("parse %s as JSON object: %w", bodyFile, err)
		}
		for k, v := range fileBody {
			payload[k] = v
		}
	}

	if strings.TrimSpace(userID) != "" {
		encoded, _ := json.Marshal(strings.TrimSpace(userID))
		payload["user_id"] = encoded
	}
	if currentStep >= 0 {
		encoded, _ := json.Marshal(currentStep)
		payload["current_step"] = encoded
	}

	if len(payload) == 0 {
		return nil, fmt.Errorf("nothing to update: supply --body-file and/or --user-id/--current-step")
	}
	return payload, nil
}

func rawOrEmpty(raw json.RawMessage, fallback string) string {
	if len(raw) == 0 {
		return fallback
	}
	return string(raw)
}
