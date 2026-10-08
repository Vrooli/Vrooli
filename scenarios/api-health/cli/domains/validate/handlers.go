package validate

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

type handlers struct {
	client scenariovalidationconnect.ScenarioValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL)}
}

func (h *handlers) validateScenarioCall(ctx cliapp.OperationContext) (*scenariovalidationv1.ValidateScenarioResponse, error) {
	name := ctx.Positional("name")
	resp, err := h.client.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         name,
		Path:             ctx.Flag("path"),
		IncludeExecution: ctx.BoolFlag("include-execution"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("validate scenario %q", name), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no validation response")
	}
	return resp.Msg, nil
}

func (h *handlers) validateScenarioReport(_ cliapp.OperationContext, msg *scenariovalidationv1.ValidateScenarioResponse) cliapp.ListReport {
	assessment := msg.GetAssessment()
	results := make([]string, 0, len(assessment.GetFindings()))
	for _, finding := range assessment.GetFindings() {
		results = append(results, fmt.Sprintf("%s %s %s", finding.GetSeverity(), finding.GetCode(), finding.GetMessage()))
	}
	return cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Validated %s - status=%s errors=%d warnings=%d infos=%d",
				msg.GetScenario(),
				statusLabel(msg.GetStatus()),
				severityCount(assessment, "SEVERITY_ERROR"),
				severityCount(assessment, "SEVERITY_WARNING"),
				severityCount(assessment, "SEVERITY_INFO"),
			),
		},
		ResultsHeading: "Findings",
		Results:        results,
	}
}

func (h *handlers) validateScenarioOutcome(msg *scenariovalidationv1.ValidateScenarioResponse) error {
	if msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		assessment := msg.GetAssessment()
		return fmt.Errorf("scenario %s did not pass API Health validation (%d error finding(s), %d warning finding(s))",
			msg.GetScenario(),
			severityCount(assessment, "SEVERITY_ERROR"),
			severityCount(assessment, "SEVERITY_WARNING"),
		)
	}
	return nil
}

func (h *handlers) previewFixCall(ctx cliapp.OperationContext) (*scenariovalidationv1.FixResponse, error) {
	return h.fixCall(ctx, false)
}

func (h *handlers) applyFixCall(ctx cliapp.OperationContext) (*scenariovalidationv1.FixResponse, error) {
	return h.fixCall(ctx, true)
}

func (h *handlers) fixCall(ctx cliapp.OperationContext, apply bool) (*scenariovalidationv1.FixResponse, error) {
	name := ctx.Positional("name")
	req := &scenariovalidationv1.FixRequest{
		Scenario: name,
		Path:     ctx.Flag("path"),
		RuleIds:  ctx.Positionals("rule_ids"),
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
	verb := "preview fixes"
	if apply {
		verb = "apply fixes"
	}
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("%s %q", verb, name), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no fix response")
	}
	return resp.Msg, nil
}

func (h *handlers) fixReport(apply bool) func(cliapp.OperationContext, *scenariovalidationv1.FixResponse) cliapp.ListReport {
	return func(_ cliapp.OperationContext, msg *scenariovalidationv1.FixResponse) cliapp.ListReport {
		verb := "preview fixes"
		if apply {
			verb = "apply fixes"
		}
		results := fixResults(msg)
		return cliapp.ListReport{
			Summary: []string{
				fmt.Sprintf("%s %s - candidates=%d applied=%t", verb, msg.GetScenario(), len(msg.GetCandidates()), msg.GetApplied()),
			},
			ResultsHeading: "Candidates",
			Results:        results,
		}
	}
}

func fixResults(msg *scenariovalidationv1.FixResponse) []string {
	results := make([]string, 0, len(msg.GetCandidates())+len(msg.GetMessages()))
	for _, candidate := range msg.GetCandidates() {
		state := "preview"
		if candidate.GetApplied() {
			state = "applied"
		}
		results = append(results, fmt.Sprintf("%s %s %s", state, candidate.GetRuleId(), candidate.GetFilePath()))
	}
	for _, message := range msg.GetMessages() {
		results = append(results, message)
	}
	return results
}

func statusLabel(status scenariovalidationv1.ValidationStatus) string {
	switch status {
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED:
		return "passed"
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED:
		return "failed"
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED:
		return "degraded"
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR:
		return "error"
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_SKIPPED:
		return "skipped"
	default:
		return "unspecified"
	}
}

func severityCount(a interface{ GetFindingsBySeverity() map[string]int32 }, severity string) int {
	if a == nil {
		return 0
	}
	return int(a.GetFindingsBySeverity()[severity])
}
