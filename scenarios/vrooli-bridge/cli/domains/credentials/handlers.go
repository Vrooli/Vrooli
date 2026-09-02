package credentials

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	grantv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/credentialgrant"
	grantconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/credentialgrant/credentialgrant_v1connect"
	"vrooli-bridge/cli/internal/session"
)

type handlers struct {
	client grantconnect.CredentialGrantServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := session.NewConnectHTTPClient(core)
	return &handlers{client: grantconnect.NewCredentialGrantServiceClient(httpClient, baseURL)}
}

func (h *handlers) grant(ctx cliapp.RunContext) error {
	resp, err := h.client.CreateGrant(context.Background(), connect.NewRequest(&grantv1.CreateGrantRequest{
		NodeId: ctx.Flag("node-id"), LogicalId: ctx.Flag("logical-id"), Field: ctx.Flag("field"),
		Class: ctx.Flag("class"), Retention: ctx.Flag("retention"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("create credential grant", err, nil)
	}
	return renderGrantMutation(ctx, resp, "Created credential grant")
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListGrants(context.Background(), connect.NewRequest(&grantv1.ListGrantsRequest{NodeId: ctx.Flag("node-id")}))
	if err != nil {
		return cliapp.WrapAPIError("list credential grants", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no credential grant response")
	}
	results := make([]string, 0, len(resp.Msg.Grants))
	for _, grant := range resp.Msg.Grants {
		results = append(results, formatGrant(grant))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("%d credential grant(s).", len(results))}, ResultsHeading: "Credential grants", Results: results})
}

func (h *handlers) revoke(ctx cliapp.RunContext) error {
	resp, err := h.client.RevokeGrant(context.Background(), connect.NewRequest(&grantv1.RevokeGrantRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return cliapp.WrapAPIError("revoke credential grant", err, nil)
	}
	return renderGrantMutation(ctx, resp, "Revoked credential grant")
}

func (h *handlers) rotate(ctx cliapp.RunContext) error {
	resp, err := h.client.RotateAddress(context.Background(), connect.NewRequest(&grantv1.RotateAddressRequest{LogicalId: ctx.Flag("logical-id"), Field: ctx.Flag("field")}))
	if err != nil {
		return cliapp.WrapAPIError("rotate credential address", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no rotation response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Rotated %s/%s to generation %d.", resp.Msg.LogicalId, resp.Msg.Field, resp.Msg.Generation)}})
}

func (h *handlers) sync(ctx cliapp.RunContext) error {
	resp, err := h.client.SyncNodeGrants(context.Background(), connect.NewRequest(&grantv1.SyncNodeGrantsRequest{NodeId: ctx.Flag("node-id")}))
	if err != nil {
		return cliapp.WrapAPIError("sync credential grants", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no credential grant response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Synchronized %d credential grant(s).", len(resp.Msg.Grants))}})
}

func renderGrantMutation(ctx cliapp.RunContext, resp *connect.Response[grantv1.CredentialGrant], action string) error {
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no credential grant")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("%s %s.", action, resp.Msg.Id)}, Changes: []string{formatGrant(resp.Msg)}})
}

func formatGrant(grant *grantv1.CredentialGrant) string {
	if grant == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s node=%s address=%s/%s class=%s retention=%s generation=%d acked=%d receipt=%t", grant.Id, grant.NodeId, grant.LogicalId, grant.Field, grant.Class, grant.Retention, grant.Generation, grant.AckedGeneration, grant.ReceiptAccepted)
}
