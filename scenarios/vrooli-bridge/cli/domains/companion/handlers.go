package companion

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	companionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/companion"
	companionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/companion/companionv1connect"
	"vrooli-bridge/cli/internal/session"
)

type handlers struct {
	client companionconnect.CompanionServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := session.NewConnectHTTPClient(core)
	return &handlers{client: companionconnect.NewCompanionServiceClient(httpClient, baseURL)}
}

func (h *handlers) install(ctx cliapp.RunContext) error {
	resp, err := h.client.Install(context.Background(), connect.NewRequest(&companionv1.InstallRequest{
		NodeId: ctx.Positional("node-id"), Version: ctx.Flag("version"), DisplayId: ctx.Flag("display"),
		ArtifactSourceRef: ctx.Flag("artifact-source"), ArtifactName: ctx.Flag("artifact-name"), ArtifactDestinationPath: ctx.Flag("artifact-dest"),
	}))
	return render(ctx, "install companion", resp, err)
}

func (h *handlers) inspect(ctx cliapp.RunContext) error {
	resp, err := h.client.Inspect(context.Background(), connect.NewRequest(&companionv1.InspectRequest{NodeId: ctx.Positional("node-id")}))
	return render(ctx, "inspect companion", resp, err)
}

func (h *handlers) upgrade(ctx cliapp.RunContext) error {
	resp, err := h.client.Upgrade(context.Background(), connect.NewRequest(&companionv1.UpgradeRequest{
		NodeId: ctx.Positional("node-id"), Version: ctx.Flag("version"),
		ArtifactSourceRef: ctx.Flag("artifact-source"), ArtifactName: ctx.Flag("artifact-name"), ArtifactDestinationPath: ctx.Flag("artifact-dest"),
	}))
	return render(ctx, "upgrade companion", resp, err)
}

func (h *handlers) revoke(ctx cliapp.RunContext) error {
	resp, err := h.client.Revoke(context.Background(), connect.NewRequest(&companionv1.RevokeRequest{NodeId: ctx.Positional("node-id"), Reason: ctx.Flag("reason")}))
	return render(ctx, "revoke companion", resp, err)
}

func (h *handlers) remove(ctx cliapp.RunContext) error {
	resp, err := h.client.Remove(context.Background(), connect.NewRequest(&companionv1.RemoveRequest{NodeId: ctx.Positional("node-id")}))
	return render(ctx, "remove companion", resp, err)
}

func render(ctx cliapp.RunContext, verb string, resp *connect.Response[companionv1.CompanionOperation], err error) error {
	if err != nil {
		return cliapp.WrapAPIError(verb, err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no companion operation")
	}
	op := resp.Msg
	return cliapp.RenderProtoMutation(ctx, op, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("%s %s — state=%s reason=%s recovery=%s", verb, op.GetOperationId(), op.GetState().String(), op.GetReasonCode(), op.GetRecovery())},
		Changes: []string{fmt.Sprintf("node=%s version=%s", op.GetNodeId(), op.GetVersion())},
	})
}
