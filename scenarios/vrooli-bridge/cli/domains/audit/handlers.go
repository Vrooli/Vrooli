package audit

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/audit"
	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/audit/audit_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"vrooli-bridge/cli/internal/session"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client auditconnect.AuditServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := session.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: auditconnect.NewAuditServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListAuditRecords(context.Background(), connect.NewRequest(&auditv1.ListAuditRecordsRequest{
		NodeId: ctx.Flag("node"),
		RunId:  ctx.Flag("run"),
		Limit:  int32(parseInt(ctx.Flag("limit"))),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list audit records", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no audit response")
	}
	results := make([]string, 0, len(resp.Msg.Records))
	for _, r := range resp.Msg.Records {
		results = append(results, formatRecord(r))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d audit record(s).", len(resp.Msg.Records))},
		ResultsHeading: "Audit trail",
		Results:        results,
		RetrievalHints: []string{
			"`audit list --node <id>` — records for one node",
			"`audit list --run <id>` — records for one run",
		},
	})
}

func formatRecord(r *auditv1.AuditRecord) string {
	if r == nil {
		return "(nil)"
	}
	when := ""
	if r.RecordedAt != nil {
		when = r.RecordedAt.AsTime().Format(time.RFC3339)
	}
	cmd := strings.TrimSpace(r.Verb + " " + r.Scenario)
	return fmt.Sprintf("%s — %s %s node=%s %q actor=%s%s [%s]",
		when, actionLabel(r.Action), outcomeLabel(r.Outcome), r.NodeId, cmd, r.Actor, runSuffix(r.RunId), detail(r.Detail))
}

func runSuffix(runID string) string {
	if runID == "" {
		return ""
	}
	return " run=" + runID
}

func detail(d string) string {
	if d == "" {
		return "-"
	}
	return d
}

func actionLabel(a auditv1.AuditAction) string {
	switch a {
	case auditv1.AuditAction_AUDIT_ACTION_DISPATCH:
		return "dispatch"
	case auditv1.AuditAction_AUDIT_ACTION_PROVISION:
		return "provision"
	default:
		return "unspecified"
	}
}

func outcomeLabel(o auditv1.AuditOutcome) string {
	switch o {
	case auditv1.AuditOutcome_AUDIT_OUTCOME_ACCEPTED:
		return "accepted"
	case auditv1.AuditOutcome_AUDIT_OUTCOME_REJECTED:
		return "rejected"
	case auditv1.AuditOutcome_AUDIT_OUTCOME_COMPLETED:
		return "completed"
	case auditv1.AuditOutcome_AUDIT_OUTCOME_FAILED:
		return "failed"
	default:
		return "unspecified"
	}
}

func parseInt(raw string) int {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	return v
}
