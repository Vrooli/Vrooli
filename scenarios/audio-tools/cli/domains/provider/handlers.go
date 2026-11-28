package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	"audio-tools/cli/domains/labels"

	plv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/provider_lifecycle"
	plconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/provider_lifecycle/provider_lifecycle_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client plconnect.ProviderLifecycleServiceClient
	now    func() time.Time
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	return newHandlersWithClock(core, func() time.Time { return time.Unix(0, 0) })
}

func newHandlersWithClock(core *cliapp.ScenarioApp, now func() time.Time) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: plconnect.NewProviderLifecycleServiceClient(httpClient, baseURL),
		now:    now,
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	return h.listPrimitive().Run(ctx)
}

func (h *handlers) listPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(_ cliapp.OperationContext) (*plv1.ListLocalProvidersResponse, error) {
		resp, err := h.client.ListLocalProviders(context.Background(), connect.NewRequest(&plv1.ListLocalProvidersRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("provider list", err, nil)
		}
		return resp.Msg, nil
	}, func(ctx cliapp.RunContext, resp *plv1.ListLocalProvidersResponse) error {
		return ctx.RenderMutation(listReport(resp.GetProviders(), h.now))
	}, nil)
}

func (h *handlers) start(ctx cliapp.RunContext) error {
	return h.startPrimitive().Run(ctx)
}

func (h *handlers) startPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(ctx cliapp.OperationContext) (*plv1.StartProviderResponse, error) {
		id := ctx.Positional("provider-id")
		resp, err := h.client.StartProvider(context.Background(), connect.NewRequest(&plv1.StartProviderRequest{ProviderId: id}))
		if err != nil {
			return nil, cliapp.WrapAPIError("provider start", err, nil)
		}
		return resp.Msg, nil
	}, func(ctx cliapp.RunContext, resp *plv1.StartProviderResponse) error {
		return ctx.RenderMutation(actionReport("start", resp.GetProviderId(), resp.GetDryRun(), resp.GetMessage()))
	}, nil)
}

func (h *handlers) stop(ctx cliapp.RunContext) error {
	return h.stopPrimitive().Run(ctx)
}

func (h *handlers) stopPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(ctx cliapp.OperationContext) (*plv1.StopProviderResponse, error) {
		id := ctx.Positional("provider-id")
		resp, err := h.client.StopProvider(context.Background(), connect.NewRequest(&plv1.StopProviderRequest{ProviderId: id}))
		if err != nil {
			return nil, cliapp.WrapAPIError("provider stop", err, nil)
		}
		return resp.Msg, nil
	}, func(ctx cliapp.RunContext, resp *plv1.StopProviderResponse) error {
		return ctx.RenderMutation(actionReport("stop", resp.GetProviderId(), resp.GetDryRun(), resp.GetMessage()))
	}, nil)
}

func (h *handlers) restart(ctx cliapp.RunContext) error {
	return h.restartPrimitive().Run(ctx)
}

func (h *handlers) restartPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(ctx cliapp.OperationContext) (*plv1.RestartProviderResponse, error) {
		id := ctx.Positional("provider-id")
		resp, err := h.client.RestartProvider(context.Background(), connect.NewRequest(&plv1.RestartProviderRequest{ProviderId: id}))
		if err != nil {
			return nil, cliapp.WrapAPIError("provider restart", err, nil)
		}
		return resp.Msg, nil
	}, func(ctx cliapp.RunContext, resp *plv1.RestartProviderResponse) error {
		return ctx.RenderMutation(actionReport("restart", resp.GetProviderId(), resp.GetDryRun(), resp.GetMessage()))
	}, nil)
}

func (h *handlers) pullModel(ctx cliapp.RunContext) error {
	return h.pullModelPrimitive().Run(ctx)
}

func (h *handlers) pullModelPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(ctx cliapp.OperationContext) (*plv1.PullModelResponse, error) {
		model := ctx.Positional("model-name")
		resp, err := h.client.PullModel(context.Background(), connect.NewRequest(&plv1.PullModelRequest{ProviderId: "ollama", ModelName: model}))
		if err != nil {
			return nil, cliapp.WrapAPIError("provider pull-model", err, nil)
		}
		return resp.Msg, nil
	}, func(ctx cliapp.RunContext, resp *plv1.PullModelResponse) error {
		return ctx.RenderMutation(actionReport("pull-model "+resp.GetModelName(), "ollama", resp.GetDryRun(), resp.GetMessage()))
	}, nil)
}

func (h *handlers) logs(ctx cliapp.RunContext) error {
	id := ctx.Positional("provider-id")
	follow := ctx.Flag("follow") == "true"
	tail := 0
	if raw := strings.TrimSpace(ctx.Flag("tail")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return fmt.Errorf("--tail must be a non-negative integer, got %q", raw)
		}
		tail = n
	}
	stream, err := h.client.GetProviderLogs(context.Background(), connect.NewRequest(&plv1.GetProviderLogsRequest{
		ProviderId: id,
		Follow:     follow,
		TailLines:  int32(tail),
	}))
	if err != nil {
		return cliapp.WrapAPIError("provider logs", err, nil)
	}
	defer stream.Close()

	out := ctx.Stdout()
	for stream.Receive() {
		line := stream.Msg()
		fmt.Fprintln(out, line.GetLine())
	}
	if err := stream.Err(); err != nil {
		return cliapp.WrapAPIError("provider logs", err, nil)
	}
	return nil
}

func listReport(providers []*plv1.LocalProvider, now func() time.Time) cliapp.MutationReport {
	rep := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Local providers  (snapshot %s)", now().UTC().Format(time.RFC3339))},
	}
	if len(providers) == 0 {
		rep.Result = append(rep.Result, "(no local providers registered)")
		return rep
	}
	rep.Changes = append(rep.Changes, fmt.Sprintf("%-22s %-22s %-12s %s", "PROVIDER", "RESOURCE", "STATE", "ACTIONS"))
	for _, p := range providers {
		rep.Changes = append(rep.Changes, fmt.Sprintf("%-22s %-22s %-12s %s",
			p.GetProviderId(),
			p.GetResourceSlug(),
			processStateLabel(p.GetProcessState()),
			actionsLabel(p.GetSupportedActions()),
		))
	}
	rep.NextCommand = []string{
		"audio-tools provider start <id>      # start a local provider",
		"audio-tools provider logs <id> -f    # tail logs",
		"audio-tools provider pull-model <m>  # pull a model on ollama",
	}
	return rep
}

func actionReport(verb, target string, dryRun bool, message string) cliapp.MutationReport {
	tag := "OK"
	if dryRun {
		tag = "DRY-RUN"
	}
	headline := fmt.Sprintf("[%s] %s %s", tag, verb, target)
	if message != "" {
		headline = headline + " — " + message
	}
	return cliapp.MutationReport{Result: []string{headline}}
}

func processStateLabel(s plv1.ProcessState) string {
	return labels.Enum(s, "PROCESS_STATE_", "-", strings.ToUpper)
}

func actionsLabel(actions []plv1.Action) string {
	parts := make([]string, 0, len(actions))
	for _, a := range actions {
		if label := labels.Enum(a, "ACTION_", "", labels.LowerWords); label != "" {
			parts = append(parts, label)
		}
	}
	return strings.Join(parts, ",")
}
