package inference_test

import (
	"context"
	"log"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"

	handler "ai-gateway/handlers/inference"
	"ai-gateway/internal/inference"
	"ai-gateway/internal/server"

	httpx "github.com/vrooli/api-core/servertest"

	"github.com/vrooli/api-core/schedule"
)

func TestRunIsReachableThroughGeneratedConnectClient(t *testing.T) {
	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
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

type digestCaptureRepository struct{ requests []inference.ProviderRequest }

func (r *digestCaptureRepository) Run(_ context.Context, request inference.ProviderRequest) (inference.ProviderResult, error) {
	r.requests = append(r.requests, request)
	return inference.ProviderResult{ValueJSON: `{}`}, nil
}

func TestRunBindsCanonicalRequestDigestAndTreatsRootAsIdempotencyOnly(t *testing.T) {
	repository := &digestCaptureRepository{}
	connectHandler := handler.NewConnectHandler(inference.NewService(repository))
	message := &inferencev1.RunRequest{
		RootOperationId: "aabbccddeeff00112233445566778899",
		Source:          "source", SchemaJson: `{"type":"object"}`, Role: "classify.fast",
	}
	_, err := connectHandler.Run(t.Context(), connect.NewRequest(message))
	require.NoError(t, err)
	first := repository.requests[0]
	require.Equal(t, message.GetRootOperationId(), first.RootOperationID)
	require.NotEmpty(t, first.RequestDigest)

	message.RootOperationId = "99887766554433221100ffeeddccbbaa"
	_, err = connectHandler.Run(t.Context(), connect.NewRequest(message))
	require.NoError(t, err)
	require.Equal(t, first.RequestDigest, repository.requests[1].RequestDigest, "root is an idempotency name, not canonical request content")

	message.Source = "changed source"
	_, err = connectHandler.Run(t.Context(), connect.NewRequest(message))
	require.NoError(t, err)
	require.NotEqual(t, first.RequestDigest, repository.requests[2].RequestDigest)
}
