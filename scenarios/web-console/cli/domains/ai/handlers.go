package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/ai/ai_v1connect"

	"github.com/vrooli/cli-core/cliapp"

	"web-console/cli/internal/support"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client aiconnect.AIServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: aiconnect.NewAIServiceClient(httpClient, baseURL),
	}
}

type generateBody struct {
	Prompt  string `json:"prompt"`
	Context string `json:"context,omitempty"`
}

func (h *handlers) generate(ctx cliapp.RunContext) error {
	raw, err := support.ReadJSONFile(ctx.Flag("body-file"), true)
	if err != nil {
		return err
	}
	var body generateBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}

	resp, err := h.client.Generate(context.Background(), connect.NewRequest(&aiv1.GenerateRequest{
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
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderListReport(ctx.Stdout(), report)
}

func (h *handlers) suggest(ctx cliapp.RunContext) error {
	raw, err := support.ReadJSONFile(ctx.Flag("body-file"), true)
	if err != nil {
		return err
	}
	var body generateBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}

	resp, err := h.client.Suggest(context.Background(), connect.NewRequest(&aiv1.SuggestRequest{
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
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderListReport(ctx.Stdout(), report)
}

func (h *handlers) configGet(ctx cliapp.RunContext) error {
	resp, err := h.client.GetConfig(context.Background(), connect.NewRequest(&aiv1.GetConfigRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("ai config-get", err, nil)
	}

	rows := []string{}
	for _, p := range resp.Msg.GetProviders() {
		rows = append(rows, fmt.Sprintf("provider %s | enabled=%t | priority=%d | timeout=%ds | retries=%d",
			p.GetName(), p.GetEnabled(), p.GetPriority(), p.GetTimeoutSec(), p.GetMaxRetries()))
	}
	for _, hh := range resp.Msg.GetHealth() {
		rows = append(rows, fmt.Sprintf("health   %s | available=%t | success=%d | errors=%d | err_rate=%.2f",
			hh.GetName(), hh.GetAvailable(), hh.GetSuccessCount(), hh.GetErrorCount(), hh.GetErrorRate()))
	}
	report := cliapp.ListReport{
		Summary:        []string{"AI provider config"},
		ResultsHeading: "Values",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("%s ai config-set --body-file config.json", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderListReport(ctx.Stdout(), report)
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

func (h *handlers) configSet(ctx cliapp.RunContext) error {
	raw, err := support.ReadJSONFile(ctx.Flag("body-file"), true)
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

	if _, err := h.client.UpdateConfig(context.Background(), connect.NewRequest(req)); err != nil {
		return cliapp.WrapAPIError("ai config-set", err, nil)
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated AI provider config for %s", body.Name)},
		NextCommand: []string{fmt.Sprintf("%s ai config-get", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

func (h *handlers) health(ctx cliapp.RunContext) error {
	resp, err := h.client.GetHealth(context.Background(), connect.NewRequest(&aiv1.GetHealthRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("ai health", err, nil)
	}

	healthy := 0
	rows := []string{}
	for _, hh := range resp.Msg.GetHealth() {
		if hh.GetAvailable() {
			healthy++
		}
		rows = append(rows, fmt.Sprintf("%s | available=%t | success=%d | errors=%d | last_check=%s",
			hh.GetName(), hh.GetAvailable(), hh.GetSuccessCount(), hh.GetErrorCount(), hh.GetLastCheck()))
	}

	report := cliapp.OperationalReport{
		Status:    []string{fmt.Sprintf("AI provider status: %d/%d healthy", healthy, len(resp.Msg.GetHealth()))},
		Triage:    []cliapp.TriageGroup{{Heading: "Providers", Items: rows}},
		NextSteps: []string{fmt.Sprintf("%s ai config-get", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderOperationalReport(ctx.Stdout(), report)
}
