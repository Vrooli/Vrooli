package audit

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/quality-health/v1/audit"
	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/quality-health/v1/audit/audit_v1connect"
)

type handlers struct {
	client auditconnect.AuditServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: auditconnect.NewAuditServiceClient(httpClient, baseURL)}
}

func (h *handlers) run(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.AuditQuality(context.Background(), connect.NewRequest(&auditv1.AuditQualityRequest{
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
	results := make([]string, 0, len(msg.GetFindings()))
	for _, f := range msg.GetFindings() {
		results = append(results, fmt.Sprintf("[%s] %s %s: %s", f.GetSeverity(), f.GetRuleId(), f.GetFilePath(), f.GetMessage()))
	}
	if len(results) == 0 {
		results = append(results, "No static-quality findings.")
	}
	err = cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("%s (%s): %d error(s), %d warning(s), maturity %s", msg.GetScenario(), msg.GetStatus(), msg.GetCounts().GetErrors(), msg.GetCounts().GetWarnings(), msg.GetMaturity().GetLabel()),
		},
		ResultsHeading: "Findings",
		Results:        results,
		RetrievalHints: msg.GetNextSteps(),
	})
	if err != nil {
		return err
	}
	if msg.GetStatus() == "failed" {
		return fmt.Errorf("quality audit failed with %d error finding(s)", msg.GetCounts().GetErrors())
	}
	return nil
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
