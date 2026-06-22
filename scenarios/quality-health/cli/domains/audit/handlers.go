package audit

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	maturityreport "github.com/vrooli/maturity-go/report"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/quality-health/v1/audit"
	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/quality-health/v1/audit/audit_v1connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/anypb"
)

type handlers struct {
	auditClient      auditconnect.AuditServiceClient
	validationClient scenariovalidationconnect.ScenarioValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		auditClient:      auditconnect.NewAuditServiceClient(httpClient, baseURL),
		validationClient: scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) run(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	if len(splitCSV(ctx.FlagValues("rule"))) > 0 || len(splitCSV(ctx.FlagValues("surface"))) > 0 || ctx.BoolFlag("autofix-preview") {
		return h.runDomainAudit(ctx, scenario)
	}
	resp, err := h.validationClient.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         scenario,
		IncludeExecution: ctx.BoolFlag("commands"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("audit %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no audit response")
	}
	msg := resp.Msg
	native, err := unpackNativeAudit(msg.GetNativeDetail())
	if err != nil {
		return err
	}
	human := auditListReport(native)
	if maturity := maturityreport.BuildMaturityListReport(msg.GetAssessment()); maturity.Summary != nil {
		human.Summary = append(human.Summary, maturity.Summary...)
		human.RetrievalHints = append(human.RetrievalHints, maturity.RetrievalHints...)
	}
	if err := cliapp.RenderProtoList(ctx, msg, human); err != nil {
		return err
	}
	if msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		return fmt.Errorf("quality audit failed with %d error finding(s)", native.GetCounts().GetErrors())
	}
	if msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR {
		return fmt.Errorf("quality audit errored")
	}
	return nil
}

func (h *handlers) runDomainAudit(ctx cliapp.RunContext, scenario string) error {
	resp, err := h.auditClient.AuditQuality(context.Background(), connect.NewRequest(&auditv1.AuditQualityRequest{
		Scenario:                scenario,
		RuleIds:                 splitCSV(ctx.FlagValues("rule")),
		Surfaces:                splitCSV(ctx.FlagValues("surface")),
		IncludeCommandExecution: ctx.BoolFlag("commands"),
		IncludeAutofixPreview:   ctx.BoolFlag("autofix-preview"),
		UseCache:                true,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("audit %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no audit response")
	}
	msg := resp.Msg
	human := auditListReport(msg)
	if maturity := maturityreport.BuildMaturityListReport(msg.GetAssessment()); maturity.Summary != nil {
		human.Summary = append(human.Summary, maturity.Summary...)
		human.RetrievalHints = append(human.RetrievalHints, maturity.RetrievalHints...)
	}
	if err := cliapp.RenderProtoList(ctx, msg, human); err != nil {
		return err
	}
	if msg.GetStatus() == "failed" {
		return fmt.Errorf("quality audit failed with %d error finding(s)", msg.GetCounts().GetErrors())
	}
	return nil
}

func unpackNativeAudit(detail *anypb.Any) (*auditv1.AuditQualityResponse, error) {
	if detail == nil {
		return nil, fmt.Errorf("server returned no native audit detail")
	}
	msg := &auditv1.AuditQualityResponse{}
	if err := detail.UnmarshalTo(msg); err != nil {
		return nil, fmt.Errorf("unpack native audit detail: %w", err)
	}
	return msg, nil
}

func auditListReport(msg *auditv1.AuditQualityResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.GetFindings()))
	for _, f := range msg.GetFindings() {
		tag := ""
		if f.GetAutofixAvailable() {
			tag = " (autofixable)"
		}
		results = append(results, fmt.Sprintf("[%s] %s%s %s: %s", f.GetSeverity(), f.GetRuleId(), tag, f.GetFilePath(), f.GetMessage()))
	}
	if len(results) == 0 {
		results = append(results, "No static-quality findings.")
	}
	return cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("%s (%s): %d error(s), %d warning(s), %d autofixable, maturity %s", msg.GetScenario(), msg.GetStatus(), msg.GetCounts().GetErrors(), msg.GetCounts().GetWarnings(), msg.GetCounts().GetAutofixableCount(), msg.GetMaturity().GetLabel()),
		},
		ResultsHeading: "Findings",
		Results:        results,
		RetrievalHints: msg.GetNextSteps(),
	}
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
