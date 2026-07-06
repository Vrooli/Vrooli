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

	"google.golang.org/protobuf/proto"

	"github.com/vrooli/cli-core/cliapp"
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
		"MeasuresService.CountRouteEvents": func(ctx cliapp.RunContext) error {
			return h.scalar(ctx, "total", func(c context.Context, w *measuresv1.TimeWindow) (proto.Message, string, error) {
				r, err := h.client.CountRouteEvents(c, req(w))
				if err != nil {
					return nil, "", err
				}
				return r.Msg, fmt.Sprintf("%d routes executed", r.Msg.GetCount()), nil
			})
		},
		"MeasuresService.CountBreakerOpenRoutes": func(ctx cliapp.RunContext) error {
			return h.scalar(ctx, "breaker-open", func(c context.Context, w *measuresv1.TimeWindow) (proto.Message, string, error) {
				r, err := h.client.CountBreakerOpenRoutes(c, req(w))
				if err != nil {
					return nil, "", err
				}
				return r.Msg, fmt.Sprintf("%d routes blocked by an open breaker", r.Msg.GetCount()), nil
			})
		},
		"MeasuresService.CountCapacityRejections": func(ctx cliapp.RunContext) error {
			return h.scalar(ctx, "capacity-rejections", func(c context.Context, w *measuresv1.TimeWindow) (proto.Message, string, error) {
				r, err := h.client.CountCapacityRejections(c, req(w))
				if err != nil {
					return nil, "", err
				}
				return r.Msg, fmt.Sprintf("%d local routes rejected for capacity", r.Msg.GetCount()), nil
			})
		},
		"MeasuresService.RouteSuccessRate": func(ctx cliapp.RunContext) error {
			return h.scalar(ctx, "success-rate", func(c context.Context, w *measuresv1.TimeWindow) (proto.Message, string, error) {
				r, err := h.client.RouteSuccessRate(c, req(w))
				if err != nil {
					return nil, "", err
				}
				return r.Msg, "route success rate: " + formatRate(r.Msg.GetRate()), nil
			})
		},
		"MeasuresService.RouteFallbackRate": func(ctx cliapp.RunContext) error {
			return h.scalar(ctx, "fallback-rate", func(c context.Context, w *measuresv1.TimeWindow) (proto.Message, string, error) {
				r, err := h.client.RouteFallbackRate(c, req(w))
				if err != nil {
					return nil, "", err
				}
				return r.Msg, "route fallback rate: " + formatRate(r.Msg.GetRate()), nil
			})
		},
		"MeasuresService.RouteFailureRate": func(ctx cliapp.RunContext) error {
			return h.scalar(ctx, "failure-rate", func(c context.Context, w *measuresv1.TimeWindow) (proto.Message, string, error) {
				r, err := h.client.RouteFailureRate(c, req(w))
				if err != nil {
					return nil, "", err
				}
				return r.Msg, "route failure rate: " + formatRate(r.Msg.GetRate()), nil
			})
		},
		"MeasuresService.RouteLatencyP95": func(ctx cliapp.RunContext) error {
			return h.scalar(ctx, "latency-p95", func(c context.Context, w *measuresv1.TimeWindow) (proto.Message, string, error) {
				r, err := h.client.RouteLatencyP95(c, req(w))
				if err != nil {
					return nil, "", err
				}
				return r.Msg, fmt.Sprintf("p95 route latency: %d ms", r.Msg.GetLatencyMs()), nil
			})
		},
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("measures: load from manifest: %w", err)
	}
	return group, nil
}
