package inference_test

import (
	"log"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"

	handler "ai-gateway/handlers/inference"
	"ai-gateway/internal/clock"
	"ai-gateway/internal/inference"
	"ai-gateway/internal/server"
	"ai-gateway/internal/testutil/httpx"
)

func TestRunIsReachableThroughGeneratedConnectClient(t *testing.T) {
	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: log.Default()},
		handler.Module(handler.Deps{Service: inference.NewService(nil)}),
	)
	live := httpx.NewLiveServer(t, srv)
	client := inferenceconnect.NewInferenceServiceClient(live.Client, live.URL)
	response, err := client.Run(t.Context(), connect.NewRequest(&inferencev1.RunRequest{
		Source:      "a source",
		SchemaJson:  `{"type":"object"}`,
		Instruction: "Return the empty object.",
		Role:        "extract.structured",
	}))
	require.NoError(t, err)
	require.True(t, response.Msg.GetValidated())
	require.Equal(t, `{}`, response.Msg.GetValueJson())
	require.NotNil(t, response.Msg.GetUsage())
}
