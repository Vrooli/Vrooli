package dispatch

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	dispatchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch"
	dispatchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch/dispatch_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"vrooli-bridge/cli/internal/session"
)

// handlers closes over *cliapp.ScenarioApp so each RunContext-func has typed
// access to the generated DispatchService client. The owner JWT rides the
// configured token source (set it via `configure token` or $VROOLI_BRIDGE_API_TOKEN).
type handlers struct {
	core   *cliapp.ScenarioApp
	client dispatchconnect.DispatchServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := session.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: dispatchconnect.NewDispatchServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) job(ctx cliapp.RunContext) error {
	nodeID := ctx.Positional("node-id")
	resp, err := h.client.DispatchJob(context.Background(), connect.NewRequest(&dispatchv1.DispatchJobRequest{
		NodeId:         nodeID,
		Verb:           ctx.Flag("verb"),
		Scenario:       ctx.Flag("scenario"),
		Args:           splitCSV(ctx.Flag("args")),
		TimeoutSeconds: parseInt64(ctx.Flag("timeout")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("dispatch job (set a token via `configure token` or $VROOLI_BRIDGE_API_TOKEN if unauthenticated)", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no dispatch response")
	}
	msg := resp.Msg

	cmd := strings.TrimSpace(msg.Verb + " " + msg.Scenario)
	if msg.DryRun {
		return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
			Result:  []string{fmt.Sprintf("[dry-run] %q would be dispatched to node %s (validated, nothing run).", cmd, msg.NodeId)},
			Changes: []string{"No run created, no audit written, nothing pushed."},
			NextCommand: []string{
				"Re-run without --dry-run to dispatch for real.",
			},
		})
	}
	return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Dispatched run %s — %q on node %s.", msg.RunId, cmd, msg.NodeId)},
		Changes: []string{fmt.Sprintf("run %s queued (verb=%q scenario=%q args=%v)", msg.RunId, msg.Verb, msg.Scenario, msg.Args)},
		NextCommand: []string{
			fmt.Sprintf("`runs wait %s` — block until it finishes", msg.RunId),
			fmt.Sprintf("`runs follow %s` — stream live output", msg.RunId),
		},
	})
}

// splitCSV parses a comma-separated flag value into a trimmed, empty-free slice.
func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseInt64 parses a flag as int64, defaulting to 0 on empty/invalid input
// (the server applies its default timeout for 0).
func parseInt64(raw string) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return v
}
