package relay

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	relayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay"
	relayconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay/relayv1connect"

	"github.com/vrooli/cli-core/cliapp"
	"vrooli-bridge/cli/internal/session"
)

type handlers struct {
	client        relayconnect.RelayServiceClient
	clientFactory func(time.Duration) relayconnect.RelayServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := session.NewConnectHTTPClient(core)
	return &handlers{
		client: relayconnect.NewRelayServiceClient(httpClient, baseURL),
		clientFactory: func(timeout time.Duration) relayconnect.RelayServiceClient {
			httpClient, baseURL := session.NewConnectHTTPClientWithTimeout(core, timeout)
			return relayconnect.NewRelayServiceClient(httpClient, baseURL)
		},
	}
}

func (h *handlers) call(ctx cliapp.RunContext) error {
	client := h.client
	if timeout := parseInt64(ctx.Flag("timeout")); timeout > 0 && h.clientFactory != nil {
		// The relay request timeout bounds the server-side node operation. The
		// CLI transport must use the same ceiling or it cancels the HTTP call
		// first, which turns a slow but valid lifecycle operation into a false
		// failure at cli-core's default 120-second deadline.
		client = h.clientFactory(time.Duration(timeout) * time.Second)
	}
	response, err := client.Call(context.Background(), connect.NewRequest(&relayv1.RelayCallRequest{
		NodeId:           ctx.Flag("node-id"),
		Scenario:         ctx.Flag("scenario"),
		Command:          ctx.Flag("command"),
		Args:             splitCSV(ctx.Flag("args")),
		TimeoutSeconds:   parseInt64(ctx.Flag("timeout")),
		MaxResponseBytes: parseUint64(ctx.Flag("max-response-bytes")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("relay call", err, nil)
	}
	if response == nil || response.Msg == nil {
		return fmt.Errorf("server returned no relay response")
	}
	return cliapp.RenderProtoOperational(ctx, response.Msg, cliapp.OperationalReport{
		Status: []string{fmt.Sprintf("Relay %s: %s (exit %d, %d bytes).", response.Msg.CorrelationId, response.Msg.Outcome.String(), response.Msg.ExitCode, response.Msg.TotalBytes)},
		Triage: []cliapp.TriageGroup{{Heading: "Target", Items: []string{fmt.Sprintf("node=%s scenario=%s command=%s", ctx.Flag("node-id"), ctx.Flag("scenario"), ctx.Flag("command"))}}},
	})
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func parseInt64(raw string) int64 {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func parseUint64(raw string) uint64 {
	value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return value
}
