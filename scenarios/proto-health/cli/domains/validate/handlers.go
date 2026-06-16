package validate

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/validation/validation_v1connect"
)

type handlers struct {
	client validationconnect.ProtoHealthServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		client: validationconnect.NewProtoHealthServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) validateScenario(ctx cliapp.RunContext) error {
	name := ctx.Positional("name")
	resp, err := h.client.ValidateScenario(context.Background(), connect.NewRequest(&validationv1.ValidateScenarioRequest{
		Scenario: name,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("validate scenario %q", name), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validation response")
	}
	msg := resp.Msg
	results := make([]string, 0, len(msg.Findings))
	for _, f := range msg.Findings {
		line := fmt.Sprintf("[%s] %s - %s (%s)", severityLabel(f.Severity), f.Code, f.Message, f.Location)
		if f.Suggestion != "" {
			line += "\n    suggestion: " + f.Suggestion
		}
		results = append(results, line)
	}
	summary := fmt.Sprintf("Validated %s - passed=%v errors=%d warnings=%d infos=%d",
		msg.Scenario, msg.Passed,
		int(msg.GetSummary().GetErrors()),
		int(msg.GetSummary().GetWarnings()),
		int(msg.GetSummary().GetInfos()),
	)
	summaryLines := []string{summary}
	if assessmentReport := cliapp.BuildMaturityListReport(msg.GetAssessment()); len(assessmentReport.Summary) > 0 {
		summaryLines = append(summaryLines, assessmentReport.Summary...)
		if len(assessmentReport.Results) > 0 {
			results = append(results, "")
			results = append(results, assessmentReport.Results...)
		}
	}
	if err := cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summaryLines,
		ResultsHeading: "Findings",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("`describe scenario %s` - inspect the proto surface facts", name),
		},
	}); err != nil {
		return err
	}
	if !msg.Passed {
		return fmt.Errorf("scenario %s did not pass proto validation (%d error finding(s))", msg.Scenario, msg.GetSummary().GetErrors())
	}
	return nil
}

func severityLabel(s validationv1.Severity) string {
	switch s {
	case validationv1.Severity_SEVERITY_ERROR:
		return "ERROR"
	case validationv1.Severity_SEVERITY_WARNING:
		return "WARN"
	case validationv1.Severity_SEVERITY_INFO:
		return "INFO"
	default:
		return "UNSPECIFIED"
	}
}
