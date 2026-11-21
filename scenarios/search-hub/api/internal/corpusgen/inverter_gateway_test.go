package corpusgen

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type corpusgenGatewayStub struct {
	inferenceconnect.UnimplementedInferenceServiceHandler
	request *inferencev1.RunRequest
}

func (g *corpusgenGatewayStub) Run(_ context.Context, request *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	g.request = request.Msg
	value, _ := json.Marshal("Query: restart the API")
	return connect.NewResponse(&inferencev1.RunResponse{ValueJson: string(value), Validated: true}), nil
}

func TestGatewayInverterSendsConfidentialLocalTypedRequest(t *testing.T) {
	stub := &corpusgenGatewayStub{}
	path, handler := inferenceconnect.NewInferenceServiceHandler(stub)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	resolve := func(context.Context) (string, error) { return server.URL, nil }
	inverter := newGatewayInverter(resolve, server.Client())
	query, err := inverter.InvertPositive(context.Background(), Item{
		ID: "run-1", Title: "Private scenario restart workflow", Type: "command",
	})
	require.NoError(t, err)
	require.Equal(t, "restart the API", query)
	require.Equal(t, defaultInverterRole, stub.request.GetRole())
	require.Equal(t, sharedv1.Profile_PROFILE_LOCAL_ONLY, stub.request.GetProfile())
	require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL, stub.request.GetPrivacyClass())
	require.Equal(t, corpusgenPrivacySource, stub.request.GetPrivacyProvenance())
	require.Equal(t, int32(inverterMaxTokens), stub.request.GetMaxOutputTokens())
	require.Contains(t, stub.request.GetSource(), "Private scenario restart workflow")
}
