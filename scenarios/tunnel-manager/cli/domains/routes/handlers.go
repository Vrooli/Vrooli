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

func (h *handlers) listCall(ctx cliapp.OperationContext) (*routesv1.ListRoutesResponse, error) {
	tier, err := tierFlag(ctx.Flag("tier"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.ListRoutes(context.Background(), connect.NewRequest(&routesv1.ListRoutesRequest{Tier: tier}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list routes", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no routes response")
	}
	return resp.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, message *routesv1.ListRoutesResponse) cliapp.ListReport {
	results := make([]string, 0, len(message.Routes))
	for _, r := range message.Routes {
		results = append(results, formatRoute(r))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d route(s).", len(message.Routes))},
		ResultsHeading: "Routes",
		Results:        results,
		RetrievalHints: []string{
			"`routes get <id>` — show a single route",
			"`routes create --subdomain <s> --scenario <n> --local-port <p>` — add a route",
		},
	}
}

func (h *handlers) getCall(ctx cliapp.OperationContext) (*routesv1.GetRouteResponse, error) {
	id := ctx.Positional("id")
	resp, err := h.client.GetRoute(context.Background(), connect.NewRequest(&routesv1.GetRouteRequest{Id: id}))
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("get route %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Route == nil {
		return nil, fmt.Errorf("server returned no route")
	}
	return resp.Msg, nil
}

func (h *handlers) getReport(_ cliapp.OperationContext, message *routesv1.GetRouteResponse) cliapp.ListReport {
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched route %s.", message.Route.Id)},
		ResultsHeading: "Route",
		Results:        []string{formatRoute(message.Route)},
	}
}

func (h *handlers) createCall(ctx cliapp.OperationContext) (*routesv1.CreateRouteResponse, error) {
	tier, err := tierFlag(ctx.Flag("tier"))
	if err != nil {
		return nil, err
	}
	external := false
	if v := strings.TrimSpace(ctx.Flag("external")); v != "" {
		external, err = strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("--external must be true or false: %w", err)
		}
	}
	target := strings.TrimSpace(ctx.Flag("target"))
	if target != "" {
		external = true
	}

	exposure, err := publicExposureFlag(ctx.Flag("public-exposure"))
	if err != nil {
		return nil, err
	}
	req := &routesv1.CreateRouteRequest{
		Subdomain:      ctx.Flag("subdomain"),
		Scenario:       ctx.Flag("scenario"),
		Domain:         ctx.Flag("domain"),
		Tier:           tier,
		HealthPath:     ctx.Flag("health-path"),
		PublicExposure: exposure,
	}
	if external {
		if target == "" {
			return nil, fmt.Errorf("--target is required with --external (e.g. http://127.0.0.1:9000)")
		}
		req.Source = routesv1.RouteSource_ROUTE_SOURCE_EXTERNAL
		req.ServiceTarget = target
	} else {
		port, err := strconv.ParseInt(strings.TrimSpace(ctx.Flag("local-port")), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("--local-port must be an integer for scenario routes (or use --external --target): %w", err)
		}
		req.LocalPort = int32(port)
	}
	resp, err := h.client.CreateRoute(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("create route", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Route == nil {
		return nil, fmt.Errorf("server returned no route")
	}
	return resp.Msg, nil
}

func (h *handlers) createReport(_ cliapp.OperationContext, message *routesv1.CreateRouteResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Created route %s.", message.Route.Id)},
		Changes: []string{formatRoute(message.Route)},
		NextCommand: []string{
			fmt.Sprintf("`routes get %s` — show this route", message.Route.Id),
			"`routes list` — show all routes",
		},
	}
}

func (h *handlers) updateCall(ctx cliapp.OperationContext) (*routesv1.UpdateRouteResponse, error) {
	req := &routesv1.UpdateRouteRequest{
		Id:            ctx.Positional("id"),
		Subdomain:     ctx.Flag("subdomain"),
		Scenario:      ctx.Flag("scenario"),
		Domain:        ctx.Flag("domain"),
		HealthPath:    ctx.Flag("health-path"),
		ServiceTarget: strings.TrimSpace(ctx.Flag("target")),
	}
	if v := strings.TrimSpace(ctx.Flag("local-port")); v != "" {
		port, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("--local-port must be an integer: %w", err)
		}
		req.LocalPort = int32(port)
	}
	tier, err := tierFlag(ctx.Flag("tier"))
	if err != nil {
		return nil, err
	}
	req.Tier = tier
	exposure, err := publicExposureFlag(ctx.Flag("public-exposure"))
	if err != nil {
		return nil, err
	}
	req.PublicExposure = exposure
	if v := strings.TrimSpace(ctx.Flag("enabled")); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("--enabled must be true or false: %w", err)
		}
		req.Enabled = &enabled
	}
	resp, err := h.client.UpdateRoute(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("update route", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Route == nil {
		return nil, fmt.Errorf("server returned no route")
	}
	return resp.Msg, nil
}

func (h *handlers) updateReport(_ cliapp.OperationContext, message *routesv1.UpdateRouteResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Updated route %s.", message.Route.Id)},
		Changes: []string{formatRoute(message.Route)},
	}
}

func (h *handlers) deleteCall(ctx cliapp.OperationContext) (*routesv1.DeleteRouteResponse, error) {
	id := ctx.Positional("id")
	resp, err := h.client.DeleteRoute(context.Background(), connect.NewRequest(&routesv1.DeleteRouteRequest{Id: id}))
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("delete route %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no delete response")
	}
	return resp.Msg, nil
}

func (h *handlers) deleteReport(ctx cliapp.OperationContext, message *routesv1.DeleteRouteResponse) cliapp.MutationReport {
	id := ctx.Positional("id")
	msg := fmt.Sprintf("Deleted route %s.", id)
	if !message.Deleted {
		msg = fmt.Sprintf("No route with id %s existed.", id)
	}
	return cliapp.MutationReport{
		Result: []string{msg},
	}
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

// publicExposureFlag maps a --public-exposure value (inherit|enabled|disabled|"")
// to the proto enum. Empty returns PUBLIC_EXPOSURE_UNSPECIFIED: on create the
// server treats it as the default (inherit); on update it leaves the route's
// existing override unchanged.
func publicExposureFlag(v string) (routesv1.PublicExposure, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "":
		return routesv1.PublicExposure_PUBLIC_EXPOSURE_UNSPECIFIED, nil
	case "inherit":
		return routesv1.PublicExposure_PUBLIC_EXPOSURE_INHERIT, nil
	case "enabled":
		return routesv1.PublicExposure_PUBLIC_EXPOSURE_ENABLED, nil
	case "disabled":
		return routesv1.PublicExposure_PUBLIC_EXPOSURE_DISABLED, nil
	default:
		return routesv1.PublicExposure_PUBLIC_EXPOSURE_UNSPECIFIED, fmt.Errorf("unknown public-exposure %q (use inherit, enabled, or disabled)", v)
	}
}

// publicExposureLabel renders a route's PublicExposure for display, omitting
// the noise of the unspecified/inherit default.
func publicExposureLabel(e routesv1.PublicExposure) string {
	switch e {
	case routesv1.PublicExposure_PUBLIC_EXPOSURE_ENABLED:
		return "enabled"
	case routesv1.PublicExposure_PUBLIC_EXPOSURE_DISABLED:
		return "disabled"
	default:
		return "inherit"
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
	pub := "public=" + publicExposureLabel(r.PublicExposure)
	if r.Source == routesv1.RouteSource_ROUTE_SOURCE_EXTERNAL {
		return fmt.Sprintf("%s — external → %s [%s, %s, %s, id=%s]",
			r.Subdomain, r.PublicUrl, r.ServiceTarget, state, pub, r.Id)
	}
	return fmt.Sprintf("%s — %s → %s :%d [%s, %s, %s, id=%s]",
		r.Subdomain, r.Scenario, r.PublicUrl, r.LocalPort, strings.ToLower(r.Tier.String()), state, pub, r.Id)
}
