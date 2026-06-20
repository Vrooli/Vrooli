package validate

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/validation/validation_v1connect"
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
		Scenario: scenario,
		Path:     firstFlag(ctx.FlagValues("path")),
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
		fix := ""
		if f.GetAutofixAvailable() {
			fix = " (auto-fixable)"
		}
		results = append(results, fmt.Sprintf("[%s] %s %s: %s%s", f.GetSeverity(), f.GetCode(), f.GetLocation(), f.GetMessage(), fix))
	}
	if len(results) == 0 {
		results = append(results, "No structure findings.")
	}

	summary := []string{
		fmt.Sprintf("%s (%s): %s", msg.GetScenario(), msg.GetStatus(), msg.GetSummary()),
	}
	if p := msg.GetProfile(); p != nil {
		recognized := "recognized"
		if !p.GetRecognized() {
			recognized = "unrecognized → advisory rules"
		}
		summary = append(summary, fmt.Sprintf("Profile: %s [%s]", p.GetId(), recognized))
	}
	if reason := strings.TrimSpace(msg.GetDegradedReason()); reason != "" {
		summary = append(summary, "Degraded: "+reason)
	}
	summary = append(summary, surfaceLines(msg.GetSurfaces())...)

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
	if strings.EqualFold(msg.GetStatus(), "failed") {
		return fmt.Errorf("structure-health validation failed for %q", scenario)
	}
	return nil
}

// surfaceLines renders the declared-vs-actual reconcile per surface.
func surfaceLines(surfaces []*validationv1.SurfaceReconcile) []string {
	if len(surfaces) == 0 {
		return nil
	}
	lines := []string{fmt.Sprintf("Surfaces (%d):", len(surfaces))}
	for _, s := range surfaces {
		state := "declared+actual"
		switch {
		case s.GetDeclared() && !s.GetActual():
			state = "declared, NOT detected"
		case !s.GetDeclared() && s.GetActual():
			state = "detected, NOT declared"
		}
		lines = append(lines, fmt.Sprintf("  • %s [%s] — %s", s.GetSurface(), s.GetKind(), state))
	}
	return lines
}

func firstFlag(values []string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
