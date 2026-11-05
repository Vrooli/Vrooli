package explain

import (
	"context"
	"fmt"

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

func (h *handlers) explain(ctx cliapp.RunContext) error {
	id := ctx.Positional("finding-id")
	resp, err := h.client.ExplainFinding(context.Background(), connect.NewRequest(&auditv1.ExplainFindingRequest{
		FindingId: id,
		RuleId:    ctx.Flag("rule"),
		Scenario:  ctx.Flag("scenario"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("explain finding", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s: %s", resp.Msg.GetContract().GetId(), resp.Msg.GetWhyItMatters())},
		ResultsHeading: "Remediation",
		Results:        []string{resp.Msg.GetRemediation()},
		RetrievalHints: resp.Msg.GetNextSteps(),
	})
}
