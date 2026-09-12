package flows_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	flowsH "flow-verifier/handlers/flows"

	flowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/flows"
	flowsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/flows/flows_v1connect"
)

func newClient(t *testing.T) flowsconnect.FlowsServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, handler := flowsconnect.NewFlowsServiceHandler(flowsH.NewConnectHandler(flowsH.Deps{Scenarios: nil, Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return flowsconnect.NewFlowsServiceClient(server.Client(), server.URL)
}

func TestListFlowsRequiresRootOrScenarios(t *testing.T) {
	client := newClient(t)
	_, err := client.ListFlows(context.Background(), connect.NewRequest(&flowsv1.ListFlowsRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestGetFlowMissingRoot(t *testing.T) {
	client := newClient(t)
	_, err := client.GetFlow(context.Background(), connect.NewRequest(&flowsv1.GetFlowRequest{FlowId: "x"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestCreateFlowInvalidLanguage(t *testing.T) {
	client := newClient(t)
	_, err := client.CreateFlow(context.Background(), connect.NewRequest(&flowsv1.CreateFlowRequest{
		FlowId:   "x",
		Language: "rust",
		Root:     t.TempDir(),
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestExplainFlowRequiresFlowID(t *testing.T) {
	client := newClient(t)
	_, err := client.ExplainFlow(context.Background(), connect.NewRequest(&flowsv1.ExplainFlowRequest{Root: t.TempDir()}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
