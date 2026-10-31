package audit

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/audit"
	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/audit/audit_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client auditconnect.AuditServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: auditconnect.NewAuditServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) runCall(_ cliapp.OperationContext) (*auditv1.RunAuditResponse, error) {
	resp, err := h.client.RunAudit(context.Background(), connect.NewRequest(&auditv1.RunAuditRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("run audit", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no audit response")
	}
	return resp.Msg, nil
}

func (h *handlers) runReport(_ cliapp.OperationContext, message *auditv1.RunAuditResponse) cliapp.ListReport {
	results := make([]string, 0, len(message.Results))
	for _, r := range message.Results {
		results = append(results, formatResult(r))
	}
	summary := []string{
		fmt.Sprintf("Audited %d route(s); %d violation(s).", len(message.Results), message.ViolationCount),
	}
	return cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Findings",
		Results:        results,
		RetrievalHints: []string{
			"`routes list` — show the manifest the audit checks",
		},
	}
}

// formatResult renders one finding as a single human-readable line.
func formatResult(r *auditv1.PortAuditResult) string {
	if r == nil {
		return "(nil)"
	}
	status := strings.TrimPrefix(strings.ToLower(r.Status.String()), "audit_status_")
	line := fmt.Sprintf("%s — %s [%s] expected :%d", r.Subdomain, r.Scenario, status, r.ExpectedPort)
	if r.ActualPort != 0 {
		line += fmt.Sprintf(", actual :%d", r.ActualPort)
	}
	if strings.TrimSpace(r.Detail) != "" {
		line += fmt.Sprintf(" — %s", r.Detail)
	}
	return line
}
