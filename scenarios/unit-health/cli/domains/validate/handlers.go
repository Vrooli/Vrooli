package validate

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation/validation_v1connect"
)

type handlers struct {
	client validationconnect.ValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: validationconnect.NewValidationServiceClient(httpClient, baseURL)}
}

func (h *handlers) validateScenario(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.ValidateScenario(context.Background(), connect.NewRequest(&validationv1.ValidateScenarioRequest{
		Scenario:         scenario,
		Path:             firstFlag(ctx.FlagValues("path")),
		Workspaces:       splitCSV(ctx.FlagValues("workspace")),
		IncludeExecution: ctx.BoolFlag("execution"),
		UseCache:         true,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("validate scenario %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validation response")
	}
	msg := resp.Msg

	results := make([]string, 0, len(msg.GetFindings()))
	for _, f := range msg.GetFindings() {
		results = append(results, fmt.Sprintf("[%s] %s %s: %s", f.GetSeverity(), f.GetCode(), f.GetFilePath(), f.GetMessage()))
	}
	if len(results) == 0 {
		results = append(results, "No test-maturity findings.")
	}

	summary := []string{
		fmt.Sprintf("%s (%s): %d error(s), %d warning(s), %d info(s) across %d workspace(s); maturity %s",
			msg.GetScenario(), msg.GetStatus(),
			msg.GetCounts().GetErrors(), msg.GetCounts().GetWarnings(), msg.GetCounts().GetInfos(),
			msg.GetCounts().GetWorkspaces(), msg.GetMaturity().GetLabel()),
	}
	if reason := strings.TrimSpace(msg.GetDegradedReason()); reason != "" {
		summary = append(summary, "Degraded: "+reason)
	}

	human := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Findings",
		Results:        results,
		RetrievalHints: msg.GetNextSteps(),
	}
	if maturity := cliapp.BuildMaturityListReport(msg.GetAssessment()); maturity.Summary != nil {
		human.Summary = append(human.Summary, maturity.Summary...)
		human.RetrievalHints = append(human.RetrievalHints, maturity.RetrievalHints...)
	}
	if err := cliapp.RenderProtoList(ctx, msg, human); err != nil {
		return err
	}
	if msg.GetStatus() == "failed" {
		return fmt.Errorf("unit-health validation failed with %d error finding(s)", msg.GetCounts().GetErrors())
	}
	return nil
}

func firstFlag(values []string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func splitCSV(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" || seen[part] {
				continue
			}
			seen[part] = true
			out = append(out, part)
		}
	}
	return out
}
