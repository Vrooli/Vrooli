package resolver

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	resolverv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/resolver"
	resolverconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/resolver/resolver_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client resolverconnect.ResolverServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return handlers{
		core:   core,
		client: resolverconnect.NewResolverServiceClient(httpClient, baseURL),
	}
}

func (h handlers) status(ctx cliapp.RunContext) error {
	resp, err := h.client.GetResolverStatus(context.Background(), connect.NewRequest(&resolverv1.GetResolverStatusRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get resolver status", err, nil)
	}
	return renderStatus(ctx, resp.Msg)
}

func (h handlers) configureAdGuard(ctx cliapp.RunContext) error {
	req := &resolverv1.ConfigureAdGuardHomeRequest{BaseUrl: ctx.Flag("base-url"), Username: ctx.Flag("username"), TokenRef: ctx.Flag("token-ref"), DryRun: ctx.BoolFlag("dry-run")}
	resp, err := h.client.ConfigureAdGuardHome(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("configure AdGuard Home", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{formatStatus(resp.Msg.GetStatus())}, Changes: resp.Msg.GetNextSteps()})
}

func (h handlers) upstreams(ctx cliapp.RunContext) error {
	var upstreams []string
	if v := ctx.Flag("upstream"); v != "" {
		upstreams = []string{v}
	}
	resp, err := h.client.UpdateUpstreams(context.Background(), connect.NewRequest(&resolverv1.UpdateUpstreamsRequest{Upstreams: upstreams, DryRun: ctx.BoolFlag("dry-run")}))
	if err != nil {
		return cliapp.WrapAPIError("update upstreams", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{formatStatus(resp.Msg.GetStatus())}, Changes: resp.Msg.GetChanges()})
}

func (h handlers) health(ctx cliapp.RunContext) error {
	resp, err := h.client.CheckResolverHealth(context.Background(), connect.NewRequest(&resolverv1.CheckResolverHealthRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("check resolver health", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{formatStatus(resp.Msg.GetStatus())}, ResultsHeading: "Checks", Results: resp.Msg.GetChecks()})
}

func renderStatus(ctx cliapp.RunContext, resp *resolverv1.GetResolverStatusResponse) error {
	return cliapp.RenderProtoList(ctx, resp, cliapp.ListReport{Summary: []string{formatStatus(resp.GetStatus())}, ResultsHeading: "Warnings", Results: resp.GetStatus().GetWarnings(), RetrievalHints: []string{"`resolver configure-adguard --dry-run` — preview backend setup"}})
}

func formatStatus(s *resolverv1.ResolverStatus) string {
	if s == nil {
		return "Resolver status unavailable."
	}
	return fmt.Sprintf("backend=%s status=%s filtering=%t", s.GetBackend(), s.GetStatus(), s.GetFilteringEnabled())
}
