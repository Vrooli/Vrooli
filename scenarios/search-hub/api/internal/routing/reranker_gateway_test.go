package routing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type rerankGatewayStub struct {
	inferenceconnect.UnimplementedInferenceServiceHandler
	request *inferencev1.RunRequest
}

func (g *rerankGatewayStub) Run(_ context.Context, request *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	g.request = request.Msg
	value, _ := json.Marshal(`{"scores":[{"index":0,"score":2},{"index":1,"score":9}]}`)
	return connect.NewResponse(&inferencev1.RunResponse{ValueJson: string(value), Validated: true}), nil
}

func TestGatewayLLMRerankerCarriesLocalConfidentialContract(t *testing.T) {
	stub := &rerankGatewayStub{}
	path, handler := inferenceconnect.NewInferenceServiceHandler(stub)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	resolve := func(context.Context) (string, error) { return server.URL, nil }
	reranker := newGatewayLLMReranker(resolve, server.Client())
	scores, err := reranker.(interface {
		Rerank(context.Context, string, []aisearch.RerankCandidate) ([]aisearch.RerankScore, error)
	}).Rerank(context.Background(), "restart a scenario", []aisearch.RerankCandidate{
		{ID: "first", Text: "Restart help"},
		{ID: "second", Text: "scenario restart"},
	})
	require.NoError(t, err)
	require.Len(t, scores, 2)
	require.Equal(t, "first", scores[0].ID)
	require.Greater(t, scores[1].Score, scores[0].Score)
	require.Equal(t, "rerank.llm_fallback", stub.request.GetRole())
	require.Equal(t, sharedv1.Profile_PROFILE_LOCAL_ONLY, stub.request.GetProfile())
	require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL, stub.request.GetPrivacyClass())
	require.Equal(t, searchHubRerankPrivacySource, stub.request.GetPrivacyProvenance())
	require.Equal(t, `{"type":"string"}`, stub.request.GetSchemaJson())
	require.Equal(t, int32(0), stub.request.GetMaxOutputTokens(), "Gateway role policy owns the output cap")
	require.Contains(t, stub.request.GetSource(), "restart a scenario")
	require.Contains(t, stub.request.GetSource(), "scenario restart")
}
