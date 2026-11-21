// Package measures is the CLI surface for AI Gateway's declared route-analytics
// measures over route_events. Each command calls the Connect-RPC MeasuresService
// — the same compute path the /measures serve registry uses, so both resolve
// identical numbers. The manifest (cli/manifest.json, group "measures") is the
// single source of truth for the command shape (flags, governance, RPC binding,
// and the measure block); handlers live in handlers.go.
package measures

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/cli-core/cliapp"
	aigwmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/measures"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

// GroupName is the manifest group name this package owns.
const GroupName = "measures"

// Register builds the measures subcommand group from the embedded manifest and
// wires each MeasuresService RPC to the shared scalar runner. Each closure
// supplies only its typed RPC call and the value formatting; the window/render
// scaffold lives in handlers.scalar.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"MeasuresService.CountRouteEvents": scalarRPC(h, "total", h.client.CountRouteEvents, func(r *connect.Response[aigwmeasuresv1.RouteCountResponse]) (proto.Message, string) {
			return r.Msg, fmt.Sprintf("%d routes executed", r.Msg.GetCount())
		}),
		"MeasuresService.CountBreakerOpenRoutes": scalarRPC(h, "breaker-open", h.client.CountBreakerOpenRoutes, func(r *connect.Response[aigwmeasuresv1.RouteCountResponse]) (proto.Message, string) {
			return r.Msg, fmt.Sprintf("%d routes blocked by an open breaker", r.Msg.GetCount())
		}),
		"MeasuresService.CountCapacityRejections": scalarRPC(h, "capacity-rejections", h.client.CountCapacityRejections, func(r *connect.Response[aigwmeasuresv1.RouteCountResponse]) (proto.Message, string) {
			return r.Msg, fmt.Sprintf("%d local routes rejected for capacity", r.Msg.GetCount())
		}),
		"MeasuresService.RouteSuccessRate": scalarRPC(h, "success-rate", h.client.RouteSuccessRate, func(r *connect.Response[aigwmeasuresv1.RouteRateResponse]) (proto.Message, string) {
			return r.Msg, "route success rate: " + formatRate(r.Msg.GetRate())
		}),
		"MeasuresService.RouteFallbackRate": scalarRPC(h, "fallback-rate", h.client.RouteFallbackRate, func(r *connect.Response[aigwmeasuresv1.RouteRateResponse]) (proto.Message, string) {
			return r.Msg, "route fallback rate: " + formatRate(r.Msg.GetRate())
		}),
		"MeasuresService.RouteFailureRate": scalarRPC(h, "failure-rate", h.client.RouteFailureRate, func(r *connect.Response[aigwmeasuresv1.RouteRateResponse]) (proto.Message, string) {
			return r.Msg, "route failure rate: " + formatRate(r.Msg.GetRate())
		}),
		"MeasuresService.RouteLatencyP95": scalarRPC(h, "latency-p95", h.client.RouteLatencyP95, func(r *connect.Response[aigwmeasuresv1.RouteLatencyResponse]) (proto.Message, string) {
			return r.Msg, fmt.Sprintf("p95 route latency: %d ms", r.Msg.GetLatencyMs())
		}),
		"MeasuresService.RouteCost": scalarRPC(h, "cost", h.client.RouteCost, func(r *connect.Response[aigwmeasuresv1.RouteCostResponse]) (proto.Message, string) {
			return r.Msg, fmt.Sprintf("route cost: %.6f USD", r.Msg.GetCostUsd())
		}),
		"MeasuresService.RouteTokens": scalarRPC(h, "tokens", h.client.RouteTokens, func(r *connect.Response[aigwmeasuresv1.RouteTokenResponse]) (proto.Message, string) {
			return r.Msg, fmt.Sprintf("route tokens: %d", r.Msg.GetTotalTokens())
		}),
		"MeasuresService.RouteLocalShare": scalarRPC(h, "local-share", h.client.RouteLocalShare, func(r *connect.Response[aigwmeasuresv1.RouteShareResponse]) (proto.Message, string) {
			return r.Msg, "route local share: " + formatRate(r.Msg.GetShare())
		}),
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("measures: load from manifest: %w", err)
	}
	return group, nil
}

func scalarRPC[T any](h *handlers, label string, invoke func(context.Context, *connect.Request[aigwmeasuresv1.RouteMeasureRequest]) (*connect.Response[T], error), render func(*connect.Response[T]) (proto.Message, string)) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		return h.scalar(ctx, label, func(callCtx context.Context, window *measuresv1.TimeWindow) (proto.Message, string, error) {
			response, err := invoke(callCtx, req(window))
			if err != nil {
				return nil, "", err
			}
			message, description := render(response)
			return message, description, nil
		})
	}
}
