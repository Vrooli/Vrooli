package homeassistant

import (
	"fmt"
	"os"
	"strings"

	"home-automation/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "home-assistant",
		Description: "Inspect, configure, test, and auto-provision the Home Assistant integration",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "status", Description: "Show Home Assistant integration health", Run: func(args []string) error { return runStatus(core, args) }},
			{Name: "set", Description: "Persist Home Assistant base URL / token / mock mode", Run: func(args []string) error { return runSet(core, args, false) }},
			{Name: "test", Description: "Test Home Assistant settings without saving", Run: func(args []string) error { return runSet(core, args, true) }},
			{Name: "provision", Description: "Auto-provision a long-lived Home Assistant token", Run: func(args []string) error { return runProvision(core, args) }},
		},
	}
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("home-assistant status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/integrations/home-assistant/config", nil)
	if err != nil {
		return err
	}

	var response support.HomeAssistantConfigResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Status: %s", firstNonEmpty(response.Status, "unknown")),
			fmt.Sprintf("Base URL: %s", firstNonEmpty(response.BaseURL, "unconfigured")),
			fmt.Sprintf("Target: %s", firstNonEmpty(response.Target, "unknown")),
			fmt.Sprintf("Token configured: %t", response.TokenConfigured),
			fmt.Sprintf("Mock mode: %t", response.MockMode),
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Diagnostics", Items: []string{
				"Message: " + firstNonEmpty(response.Message, "none"),
				"Error: " + firstNonEmpty(response.Error, "none"),
				"Action required: " + firstNonEmpty(response.ActionRequired, "none"),
				fmt.Sprintf("Token type: %s", firstNonEmpty(response.TokenType, "unknown")),
				fmt.Sprintf("Status checked at: %s", support.HumanTime(response.StatusCheckedAt)),
				fmt.Sprintf("Updated at: %s", support.HumanTime(response.UpdatedAt)),
			}},
		},
		NextSteps: []string{
			"home-automation home-assistant test --base-url http://localhost:8123 --token <token>",
			"home-automation home-assistant provision --base-url http://localhost:8123",
		},
	}
	if *jsonOutput {
		return support.PrintJSONReport(true, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func runSet(core *cliapp.ScenarioApp, args []string, testOnly bool) error {
	commandName := "set"
	if testOnly {
		commandName = "test"
	}
	fs := support.NewFlagSet("home-assistant " + commandName)
	baseURL := fs.String("base-url", "", "Home Assistant base URL")
	token := fs.String("token", "", "Home Assistant long-lived token")
	mock := fs.Bool("mock", false, "Enable mock mode")
	live := fs.Bool("live", false, "Disable mock mode")
	clearToken := fs.Bool("clear-token", false, "Clear any saved token")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *mock && *live {
		return fmt.Errorf("--mock and --live cannot be used together")
	}

	payload := map[string]interface{}{
		"base_url":    strings.TrimSpace(*baseURL),
		"clear_token": *clearToken,
		"test_only":   testOnly,
	}
	if strings.TrimSpace(*token) != "" {
		payload["token"] = strings.TrimSpace(*token)
	}
	if *mock || *live {
		payload["mock_mode"] = *mock
	}

	body, err := core.Request("POST", "/integrations/home-assistant/config", nil, payload)
	if err != nil {
		return err
	}

	var response support.HomeAssistantConfigResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	resultVerb := "tested"
	if response.Saved {
		resultVerb = "saved"
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Home Assistant settings %s.", resultVerb),
			fmt.Sprintf("Status: %s", firstNonEmpty(response.Status, "unknown")),
			fmt.Sprintf("Base URL: %s", firstNonEmpty(response.BaseURL, "unconfigured")),
		},
		Changes: []string{
			fmt.Sprintf("Token configured: %t", response.TokenConfigured),
			fmt.Sprintf("Mock mode: %t", response.MockMode),
			"Message: " + firstNonEmpty(response.Message, "none"),
			"Error: " + firstNonEmpty(response.Error, "none"),
			"Action required: " + firstNonEmpty(response.ActionRequired, "none"),
		},
		NextCommand: []string{
			"home-automation home-assistant status",
			"home-automation devices list --filter light",
		},
	}
	if *jsonOutput {
		return support.PrintJSONReport(true, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runProvision(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("home-assistant provision")
	baseURL := fs.String("base-url", "", "Home Assistant base URL")
	clientName := fs.String("client-name", "Home Automation Intelligence", "Client name for the generated token")
	lifespanDays := fs.Int("lifespan-days", 365, "Long-lived token lifespan in days")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Request("POST", "/integrations/home-assistant/provision", nil, map[string]interface{}{
		"base_url":      strings.TrimSpace(*baseURL),
		"client_name":   strings.TrimSpace(*clientName),
		"lifespan_days": *lifespanDays,
	})
	if err != nil {
		return err
	}

	var response support.HomeAssistantConfigResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Auto-provisioned: %t", response.AutoProvisioned),
			fmt.Sprintf("Saved: %t", response.Saved),
			fmt.Sprintf("Status: %s", firstNonEmpty(response.Status, "unknown")),
			fmt.Sprintf("Base URL: %s", firstNonEmpty(response.BaseURL, "unconfigured")),
		},
		Changes: []string{
			"Message: " + firstNonEmpty(response.Message, "none"),
			fmt.Sprintf("Token configured: %t", response.TokenConfigured),
			fmt.Sprintf("Mock mode: %t", response.MockMode),
			"Error: " + firstNonEmpty(response.Error, "none"),
		},
		NextCommand: []string{
			"home-automation home-assistant status",
			"home-automation devices list",
		},
	}
	if *jsonOutput {
		return support.PrintJSONReport(true, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
