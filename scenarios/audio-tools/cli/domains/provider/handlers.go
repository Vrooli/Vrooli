package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	plv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/provider_lifecycle"
	plconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/provider_lifecycle/provider_lifecycle_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client plconnect.ProviderLifecycleServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: plconnect.NewProviderLifecycleServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListLocalProviders(context.Background(), connect.NewRequest(&plv1.ListLocalProvidersRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("provider list", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, listReport(resp.Msg.GetProviders()))
}

func (h *handlers) start(ctx cliapp.RunContext) error {
	id := ctx.Positional("provider-id")
	resp, err := h.client.StartProvider(context.Background(), connect.NewRequest(&plv1.StartProviderRequest{ProviderId: id}))
	if err != nil {
		return cliapp.WrapAPIError("provider start", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, actionReport("start", id, resp.Msg.GetDryRun(), resp.Msg.GetMessage()))
}

func (h *handlers) stop(ctx cliapp.RunContext) error {
	id := ctx.Positional("provider-id")
	resp, err := h.client.StopProvider(context.Background(), connect.NewRequest(&plv1.StopProviderRequest{ProviderId: id}))
	if err != nil {
		return cliapp.WrapAPIError("provider stop", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, actionReport("stop", id, resp.Msg.GetDryRun(), resp.Msg.GetMessage()))
}

func (h *handlers) restart(ctx cliapp.RunContext) error {
	id := ctx.Positional("provider-id")
	resp, err := h.client.RestartProvider(context.Background(), connect.NewRequest(&plv1.RestartProviderRequest{ProviderId: id}))
	if err != nil {
		return cliapp.WrapAPIError("provider restart", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, actionReport("restart", id, resp.Msg.GetDryRun(), resp.Msg.GetMessage()))
}

func (h *handlers) pullModel(ctx cliapp.RunContext) error {
	model := ctx.Positional("model-name")
	resp, err := h.client.PullModel(context.Background(), connect.NewRequest(&plv1.PullModelRequest{ProviderId: "ollama", ModelName: model}))
	if err != nil {
		return cliapp.WrapAPIError("provider pull-model", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, actionReport("pull-model "+model, "ollama", resp.Msg.GetDryRun(), resp.Msg.GetMessage()))
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

func listReport(providers []*plv1.LocalProvider) cliapp.MutationReport {
	rep := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Local providers  (snapshot %s)", time.Now().UTC().Format(time.RFC3339))},
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
	switch s {
	case plv1.ProcessState_PROCESS_STATE_RUNNING:
		return "RUNNING"
	case plv1.ProcessState_PROCESS_STATE_STOPPED:
		return "STOPPED"
	case plv1.ProcessState_PROCESS_STATE_UNKNOWN:
		return "UNKNOWN"
	}
	return "-"
}

func actionsLabel(actions []plv1.Action) string {
	parts := make([]string, 0, len(actions))
	for _, a := range actions {
		switch a {
		case plv1.Action_ACTION_START:
			parts = append(parts, "start")
		case plv1.Action_ACTION_STOP:
			parts = append(parts, "stop")
		case plv1.Action_ACTION_RESTART:
			parts = append(parts, "restart")
		case plv1.Action_ACTION_PULL_MODEL:
			parts = append(parts, "pull-model")
		case plv1.Action_ACTION_VIEW_LOGS:
			parts = append(parts, "logs")
		}
	}
	return strings.Join(parts, ",")
}
