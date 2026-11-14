package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
)

type unknownMediaClient struct {
	routingconnect.RoutingServiceClient
	receipt *routingv1.MediaExecution
}

func (c unknownMediaClient) CancelMediaExecution(context.Context, *connect.Request[routingv1.CancelMediaExecutionRequest]) (*connect.Response[routingv1.CancelMediaExecutionResponse], error) {
	return connect.NewResponse(&routingv1.CancelMediaExecutionResponse{Execution: c.receipt}), nil
}

func TestMediaCancelPreservesUnknownOutcomeInHumanAndJSONOutput(t *testing.T) {
	receipt := &routingv1.MediaExecution{ExecutionId: "unknown-id", Status: routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_FAILED, ErrorCode: "outcome_unknown", ErrorMessage: "provider outcome unknown; not replayed; cost unknown"}
	for _, structured := range []bool{false, true} {
		t.Run(map[bool]string{false: "human", true: "json"}[structured], func(t *testing.T) {
			var out bytes.Buffer
			ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "execution-id"}}}, Positionals: map[string]string{"execution-id": receipt.GetExecutionId()}, JSON: structured, Stdout: &out})
			require.NoError(t, (&handlers{client: unknownMediaClient{receipt: receipt}}).mediaCancel(ctx))
			if structured {
				var body map[string]any
				require.NoError(t, json.Unmarshal(out.Bytes(), &body))
				execution := body["execution"].(map[string]any)
				require.Equal(t, "outcome_unknown", execution["error_code"])
				require.Equal(t, "MEDIA_EXECUTION_STATUS_FAILED", execution["status"])
				require.NotContains(t, execution, "actual_cost_usd", "unknown cost must not become a string in typed proto JSON")
			} else {
				require.Contains(t, out.String(), "Unchanged media execution unknown-id")
				require.NotContains(t, out.String(), "Cancelled media execution")
				require.Contains(t, out.String(), "actual_cost_usd=unknown")
				require.Contains(t, out.String(), "outcome_unknown")
				require.NotContains(t, out.String(), "actual_cost_usd=0.0000")
			}
		})
	}
}

func TestMediaUnknownSubmitDescribesRetainedReceipt(t *testing.T) {
	var out bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Stdout: &out})
	require.NoError(t, renderMediaSubmit(ctx, &routingv1.SubmitMediaResponse{Execution: &routingv1.MediaExecution{ExecutionId: "same-id", Status: routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_FAILED, ErrorCode: "outcome_unknown"}}))
	require.Contains(t, out.String(), "Retained media execution same-id")
	require.False(t, strings.Contains(out.String(), "Submitted media execution"))
}
