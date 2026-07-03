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

func (h *handlers) validateScenario(ctx cliapp.RunContext) error {
	name := ctx.Positional("name")
	resp, err := h.client.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         name,
		Path:             ctx.Flag("path"),
		IncludeExecution: ctx.BoolFlag("include-execution"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("validate scenario %q", name), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validation response")
	}
	msg := resp.Msg
	assessment := msg.GetAssessment()
	results := make([]string, 0, len(assessment.GetFindings()))
	for _, finding := range assessment.GetFindings() {
		results = append(results, fmt.Sprintf("%s %s %s", finding.GetSeverity(), finding.GetCode(), finding.GetMessage()))
	}
	if err := cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
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
	}); err != nil {
		return err
	}
	if msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		return fmt.Errorf("scenario %s did not pass API Health validation (%d error finding(s), %d warning finding(s))",
			msg.GetScenario(),
			severityCount(assessment, "SEVERITY_ERROR"),
			severityCount(assessment, "SEVERITY_WARNING"),
		)
	}
	return nil
}

func (h *handlers) previewFix(ctx cliapp.RunContext) error {
	return h.fix(ctx, false)
}

func (h *handlers) applyFix(ctx cliapp.RunContext) error {
	return h.fix(ctx, true)
}

func (h *handlers) fix(ctx cliapp.RunContext, apply bool) error {
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
		return cliapp.WrapAPIError(fmt.Sprintf("%s %q", verb, name), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no fix response")
	}
	msg := resp.Msg
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
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("%s %s - candidates=%d applied=%t", verb, msg.GetScenario(), len(msg.GetCandidates()), msg.GetApplied()),
		},
		ResultsHeading: "Candidates",
		Results:        results,
	})
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
