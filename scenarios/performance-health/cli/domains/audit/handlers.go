package audit

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/audit"
	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/audit/audit_v1connect"
)

type handlers struct {
	client auditconnect.AuditServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: auditconnect.NewAuditServiceClient(httpClient, baseURL)}
}

// run orchestrates a profile-mode perf capture of a scenario.
func (h *handlers) run(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.RunAudit(context.Background(), connect.NewRequest(&auditv1.RunAuditRequest{
		Scenario: scenario,
		Path:     firstFlag(ctx.FlagValues("path")),
		Workflow: firstFlag(ctx.FlagValues("workflow")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("run audit for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no audit response")
	}
	msg := resp.Msg
	result := []string{fmt.Sprintf("%s: %s.", msg.GetScenario(), outcomeLabel(msg.GetOutcome()))}
	if r := strings.TrimSpace(msg.GetReason()); r != "" {
		result = append(result, "Reason: "+r)
	}
	changes := []string{}
	if a := strings.TrimSpace(msg.GetTraceArtifact()); a != "" {
		changes = append(changes, "trace: "+a)
	}
	if a := strings.TrimSpace(msg.GetWebVitalsArtifact()); a != "" {
		changes = append(changes, "web-vitals: "+a)
	}
	return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{Result: result, Changes: changes})
}

func outcomeLabel(o auditv1.AuditOutcome) string {
	switch o {
	case auditv1.AuditOutcome_AUDIT_OUTCOME_CAPTURED:
		return "captured"
	case auditv1.AuditOutcome_AUDIT_OUTCOME_SKIPPED:
		return "skipped"
	case auditv1.AuditOutcome_AUDIT_OUTCOME_FAILED:
		return "failed"
	default:
		return "unspecified"
	}
}

func firstFlag(values []string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
