package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"connectrpc.com/connect"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/ai/ai_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"web-console/cli/internal/support"
)

// Register builds the `ai` subcommand group for AI generation, suggestion,
// provider config, and provider health surfaces.
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

func newClient(core *cliapp.ScenarioApp) aiconnect.AIServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return aiconnect.NewAIServiceClient(httpClient, baseURL)
}

type generateBody struct {
	Prompt  string `json:"prompt"`
	Context string `json:"context,omitempty"`
}

func runGenerate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("ai generate")
	bodyFile := fs.String("body-file", "", "Path to a JSON request body (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	var body generateBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}

	resp, err := newClient(core).Generate(context.Background(), connect.NewRequest(&aiv1.GenerateRequest{
		Prompt:  body.Prompt,
		Context: body.Context,
	}))
	if err != nil {
		return cliapp.WrapAPIError("ai generate", err, nil)
	}

	report := cliapp.ListReport{
		Summary:        []string{"Generated command"},
		ResultsHeading: "Response",
		Results: []string{
			fmt.Sprintf("command: %s", resp.Msg.GetCommand()),
			fmt.Sprintf("provider: %s", resp.Msg.GetProvider()),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runSuggest(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("ai suggest")
	bodyFile := fs.String("body-file", "", "Path to a JSON request body (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	var body generateBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}

	resp, err := newClient(core).Suggest(context.Background(), connect.NewRequest(&aiv1.SuggestRequest{
		Prompt:  body.Prompt,
		Context: body.Context,
	}))
	if err != nil {
		return cliapp.WrapAPIError("ai suggest", err, nil)
	}

	rows := []string{fmt.Sprintf("provider: %s", resp.Msg.GetProvider())}
	for i, c := range resp.Msg.GetCommands() {
		rows = append(rows, fmt.Sprintf("  [%d] %s", i+1, c))
	}
	report := cliapp.ListReport{
		Summary:        []string{"AI suggestion"},
		ResultsHeading: "Commands",
		Results:        rows,
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

	resp, err := newClient(core).GetConfig(context.Background(), connect.NewRequest(&aiv1.GetConfigRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("ai config-get", err, nil)
	}

	rows := []string{}
	for _, p := range resp.Msg.GetProviders() {
		rows = append(rows, fmt.Sprintf("provider %s | enabled=%t | priority=%d | timeout=%ds | retries=%d",
			p.GetName(), p.GetEnabled(), p.GetPriority(), p.GetTimeoutSec(), p.GetMaxRetries()))
	}
	for _, h := range resp.Msg.GetHealth() {
		rows = append(rows, fmt.Sprintf("health   %s | available=%t | success=%d | errors=%d | err_rate=%.2f",
			h.GetName(), h.GetAvailable(), h.GetSuccessCount(), h.GetErrorCount(), h.GetErrorRate()))
	}
	report := cliapp.ListReport{
		Summary:        []string{"AI provider config"},
		ResultsHeading: "Values",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("%s ai config-set --body-file config.json", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// configUpdateBody mirrors UpdateConfigRequest. Each pointer-field present
// in the JSON flips the corresponding has_* flag server-side.
type configUpdateBody struct {
	Name       string `json:"name"`
	Enabled    *bool  `json:"enabled,omitempty"`
	Priority   *int32 `json:"priority,omitempty"`
	TimeoutSec *int32 `json:"timeout_sec,omitempty"`
	MaxRetries *int32 `json:"max_retries,omitempty"`
}

func runConfigSet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("ai config-set")
	bodyFile := fs.String("body-file", "", "Path to JSON body with provider config (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	var body configUpdateBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}
	if body.Name == "" {
		return fmt.Errorf("--body-file must include a provider name")
	}

	req := &aiv1.UpdateConfigRequest{Name: body.Name}
	if body.Enabled != nil {
		req.Enabled = *body.Enabled
		req.HasEnabled = true
	}
	if body.Priority != nil {
		req.Priority = *body.Priority
		req.HasPriority = true
	}
	if body.TimeoutSec != nil {
		req.TimeoutSec = *body.TimeoutSec
		req.HasTimeoutSec = true
	}
	if body.MaxRetries != nil {
		req.MaxRetries = *body.MaxRetries
		req.HasMaxRetries = true
	}

	if _, err := newClient(core).UpdateConfig(context.Background(), connect.NewRequest(req)); err != nil {
		return cliapp.WrapAPIError("ai config-set", err, nil)
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated AI provider config for %s", body.Name)},
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

	resp, err := newClient(core).GetHealth(context.Background(), connect.NewRequest(&aiv1.GetHealthRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("ai health", err, nil)
	}

	healthy := 0
	rows := []string{}
	for _, h := range resp.Msg.GetHealth() {
		if h.GetAvailable() {
			healthy++
		}
		rows = append(rows, fmt.Sprintf("%s | available=%t | success=%d | errors=%d | last_check=%s",
			h.GetName(), h.GetAvailable(), h.GetSuccessCount(), h.GetErrorCount(), h.GetLastCheck()))
	}

	report := cliapp.OperationalReport{
		Status:    []string{fmt.Sprintf("AI provider status: %d/%d healthy", healthy, len(resp.Msg.GetHealth()))},
		Triage:    []cliapp.TriageGroup{{Heading: "Providers", Items: rows}},
		NextSteps: []string{fmt.Sprintf("%s ai config-get", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}
