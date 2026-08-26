package relay

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"
	relayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay"
	relayconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay/relayv1connect"
)

type fakeRelayClient struct{}

func (fakeRelayClient) Call(_ context.Context, _ *connect.Request[relayv1.RelayCallRequest]) (*connect.Response[relayv1.RelayCallResponse], error) {
	return connect.NewResponse(&relayv1.RelayCallResponse{
		Outcome: relayv1.RelayCallOutcome_RELAY_CALL_OUTCOME_COMPLETED,
		Data:    []byte("ok"),
	}), nil
}

var _ relayconnect.RelayServiceClient = fakeRelayClient{}

func TestCallUsesRequestedTimeoutForHTTPTransport(t *testing.T) {
	var got time.Duration
	h := &handlers{
		client: fakeRelayClient{},
		clientFactory: func(timeout time.Duration) relayconnect.RelayServiceClient {
			got = timeout
			return fakeRelayClient{}
		},
	}
	core := &cliapp.ScenarioApp{}
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{
			{Name: "node-id"}, {Name: "scenario"}, {Name: "command"},
			{Name: "args"}, {Name: "timeout"}, {Name: "max-response-bytes"},
		},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{
			"node-id": "node-1", "scenario": "system-monitor",
			"command": "scenario start", "timeout": "600",
		},
	})

	require.NoError(t, h.call(ctx))
	require.Equal(t, 600*time.Second, got)
}
