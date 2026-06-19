package routes

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	routesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/routes"
	routesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/routes/routes_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client routesconnect.RoutesServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: routesconnect.NewRoutesServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	tier, err := tierFlag(ctx.Flag("tier"))
	if err != nil {
		return err
	}
	resp, err := h.client.ListRoutes(context.Background(), connect.NewRequest(&routesv1.ListRoutesRequest{Tier: tier}))
	if err != nil {
		return cliapp.WrapAPIError("list routes", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no routes response")
	}
	results := make([]string, 0, len(resp.Msg.Routes))
	for _, r := range resp.Msg.Routes {
		results = append(results, formatRoute(r))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d route(s).", len(resp.Msg.Routes))},
		ResultsHeading: "Routes",
		Results:        results,
		RetrievalHints: []string{
			"`routes get <id>` — show a single route",
			"`routes create --subdomain <s> --scenario <n> --local-port <p>` — add a route",
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetRoute(context.Background(), connect.NewRequest(&routesv1.GetRouteRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get route %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Route == nil {
		return fmt.Errorf("server returned no route")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched route %s.", resp.Msg.Route.Id)},
		ResultsHeading: "Route",
		Results:        []string{formatRoute(resp.Msg.Route)},
	})
}

func (h *handlers) create(ctx cliapp.RunContext) error {
	port, err := strconv.ParseInt(strings.TrimSpace(ctx.Flag("local-port")), 10, 32)
	if err != nil {
		return fmt.Errorf("--local-port must be an integer: %w", err)
	}
	tier, err := tierFlag(ctx.Flag("tier"))
	if err != nil {
		return err
	}
	resp, err := h.client.CreateRoute(context.Background(), connect.NewRequest(&routesv1.CreateRouteRequest{
		Subdomain:  ctx.Flag("subdomain"),
		Scenario:   ctx.Flag("scenario"),
		Domain:     ctx.Flag("domain"),
		LocalPort:  int32(port),
		Tier:       tier,
		HealthPath: ctx.Flag("health-path"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("create route", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Route == nil {
		return fmt.Errorf("server returned no route")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Created route %s.", resp.Msg.Route.Id)},
		Changes: []string{formatRoute(resp.Msg.Route)},
		NextCommand: []string{
			fmt.Sprintf("`routes get %s` — show this route", resp.Msg.Route.Id),
			"`routes list` — show all routes",
		},
	})
}

func (h *handlers) update(ctx cliapp.RunContext) error {
	req := &routesv1.UpdateRouteRequest{
		Id:         ctx.Positional("id"),
		Subdomain:  ctx.Flag("subdomain"),
		Scenario:   ctx.Flag("scenario"),
		Domain:     ctx.Flag("domain"),
		HealthPath: ctx.Flag("health-path"),
	}
	if v := strings.TrimSpace(ctx.Flag("local-port")); v != "" {
		port, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return fmt.Errorf("--local-port must be an integer: %w", err)
		}
		req.LocalPort = int32(port)
	}
	tier, err := tierFlag(ctx.Flag("tier"))
	if err != nil {
		return err
	}
	req.Tier = tier
	if v := strings.TrimSpace(ctx.Flag("enabled")); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("--enabled must be true or false: %w", err)
		}
		req.Enabled = &enabled
	}
	resp, err := h.client.UpdateRoute(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("update route", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Route == nil {
		return fmt.Errorf("server returned no route")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Updated route %s.", resp.Msg.Route.Id)},
		Changes: []string{formatRoute(resp.Msg.Route)},
	})
}

func (h *handlers) delete(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.DeleteRoute(context.Background(), connect.NewRequest(&routesv1.DeleteRouteRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("delete route %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no delete response")
	}
	msg := fmt.Sprintf("Deleted route %s.", id)
	if !resp.Msg.Deleted {
		msg = fmt.Sprintf("No route with id %s existed.", id)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{msg},
	})
}

// tierFlag maps a --tier value (core|leased|"") to the proto enum. Empty
// returns TIER_UNSPECIFIED (list = all; create = server default leased).
func tierFlag(v string) (routesv1.Tier, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "":
		return routesv1.Tier_TIER_UNSPECIFIED, nil
	case "core":
		return routesv1.Tier_TIER_CORE, nil
	case "leased":
		return routesv1.Tier_TIER_LEASED, nil
	default:
		return routesv1.Tier_TIER_UNSPECIFIED, fmt.Errorf("unknown tier %q (use core or leased)", v)
	}
}

func formatRoute(r *routesv1.Route) string {
	if r == nil {
		return "(nil)"
	}
	state := "enabled"
	if !r.Enabled {
		state = "disabled"
	}
	return fmt.Sprintf("%s — %s → %s :%d [%s, %s, id=%s]",
		r.Subdomain, r.Scenario, r.PublicUrl, r.LocalPort, strings.ToLower(r.Tier.String()), state, r.Id)
}
