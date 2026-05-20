package validate

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/validation/validation_v1connect"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each RunCtx
// func has typed access to the generated Connect client without
// re-resolving it.
type handlers struct {
	core   *cliapp.ScenarioApp
	client validationconnect.ValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: validationconnect.NewValidationServiceClient(httpClient, baseURL),
	}
}

// validateScenario calls ValidationService.ValidateScenario, renders the
// returned Findings to human / JSON output, and returns a non-nil error when
// the report contains any SEVERITY_ERROR finding so shells get a non-zero
// exit code without a duplicated stderr noise line.
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
		sev := severityLabel(f.Severity)
		line := fmt.Sprintf("[%s] %s — %s (%s)", sev, f.Code, f.Message, f.Location)
		if f.Suggestion != "" {
			line += "\n    suggestion: " + f.Suggestion
		}
		results = append(results, line)
	}
	summary := fmt.Sprintf("Validated %s — passed=%v errors=%d warnings=%d infos=%d",
		msg.Scenario, msg.Passed,
		int(msg.GetSummary().GetErrors()),
		int(msg.GetSummary().GetWarnings()),
		int(msg.GetSummary().GetInfos()),
	)
	if err := cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Findings",
		Results:        results,
	}); err != nil {
		return err
	}
	if !msg.Passed {
		return fmt.Errorf("scenario %s did not pass validation (%d error finding(s))", msg.Scenario, msg.GetSummary().GetErrors())
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
