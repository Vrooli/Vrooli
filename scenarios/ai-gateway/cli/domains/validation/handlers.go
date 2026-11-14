package validation

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	client validationconnect.ScenarioValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: validationconnect.NewScenarioValidationServiceClient(httpClient, baseURL)}
}

func (h *handlers) validate(ctx cliapp.RunContext) error {
	resp, err := h.client.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         ctx.Flag("scenario"),
		Path:             ctx.Flag("path"),
		IncludeExecution: ctx.BoolFlag("include-execution"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("validate scenario AI conformance", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validation response")
	}
	results := []string{fmt.Sprintf("status=%s", resp.Msg.GetStatus().String())}
	if assessment := resp.Msg.GetAssessment(); assessment != nil {
		level := ""
		if assessment.GetLocal() != nil {
			level = assessment.GetLocal().GetCurrentLevel()
		}
		results = append(results, fmt.Sprintf("assessment=%s level=%s findings=%d", assessment.GetPhase(), level, len(assessment.GetFindings())))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Shared AI conformance validation for %s: %s.", resp.Msg.GetScenario(), resp.Msg.GetStatus().String())},
		ResultsHeading: "Validation",
		Results:        results,
	})
}

func (h *handlers) describeProvider(ctx cliapp.RunContext) error {
	resp, err := h.client.DescribeProvider(context.Background(), connect.NewRequest(&scenariovalidationv1.DescribeProviderRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("describe AI conformance provider", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no provider descriptor")
	}
	results := []string{fmt.Sprintf("provider=%s phase=%s contract=%s", resp.Msg.GetProvider(), resp.Msg.GetPhase(), resp.Msg.GetContract())}
	if caps := resp.Msg.GetCapabilities(); caps != nil {
		results = append(results, fmt.Sprintf("supports_execution=%t supports_fixes=%t delivery=%s", caps.GetSupportsExecution(), caps.GetSupportsFixes(), caps.GetDeliveryMode()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{"AI Gateway validation provider descriptor."}, ResultsHeading: "Provider", Results: results})
}

func (h *handlers) previewFix(ctx cliapp.RunContext) error {
	return h.fix(ctx, false)
}

func (h *handlers) applyFix(ctx cliapp.RunContext) error {
	return h.fix(ctx, true)
}

func (h *handlers) fix(ctx cliapp.RunContext, apply bool) error {
	req := &scenariovalidationv1.FixRequest{
		Scenario: ctx.Flag("scenario"),
		Path:     ctx.Flag("path"),
		RuleIds:  splitCSV(ctx.Flag("rule-ids")),
	}
	var (
		resp *connect.Response[scenariovalidationv1.FixResponse]
		err  error
	)
	if apply {
		resp, err = h.client.ApplyFix(context.Background(), connect.NewRequest(req))
	} else {
		resp, err = h.client.PreviewFix(context.Background(), connect.NewRequest(req))
	}
	if err != nil {
		return cliapp.WrapAPIError("AI conformance fix", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no fix response")
	}
	results := append([]string{}, resp.Msg.GetMessages()...)
	for _, candidate := range resp.Msg.GetCandidates() {
		results = append(results, fmt.Sprintf("%s %s applied=%t: %s", candidate.GetRuleId(), candidate.GetFilePath(), candidate.GetApplied(), candidate.GetDescription()))
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("AI conformance fix applied=%t.", resp.Msg.GetApplied())},
		Changes: results,
	})
}

func splitCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
