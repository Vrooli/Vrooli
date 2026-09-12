package contracts

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

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListContracts(context.Background(), connect.NewRequest(&auditv1.ListContractsRequest{
		Language:    ctx.Flag("language"),
		Framework:   ctx.Flag("framework"),
		SurfaceKind: ctx.Flag("surface-kind"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list contracts", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetContracts()))
	for _, c := range resp.Msg.GetContracts() {
		results = append(results, fmt.Sprintf("%s — %s (%s)", c.GetId(), c.GetTitle(), c.GetSeverity()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d Quality Health contract(s).", len(resp.Msg.GetContracts()))},
		ResultsHeading: "Contracts",
		Results:        results,
	})
}
