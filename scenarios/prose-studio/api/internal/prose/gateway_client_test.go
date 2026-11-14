package prose

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type privacyCaptureInference struct {
	inferenceconnect.UnimplementedInferenceServiceHandler
	request *inferencev1.RunRequest
}

func (f *privacyCaptureInference) Run(_ context.Context, request *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	f.request = request.Msg
	return connect.NewResponse(&inferencev1.RunResponse{Error: &inferencev1.InferenceError{
		Code:    inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE,
		Message: "remote providers require an explicit privacy classification",
	}}), nil
}

func TestHTTPGatewayPreservesUnknownPrivacyAndSurfacesRemoteRefusal(t *testing.T) {
	fixture := &privacyCaptureInference{}
	path, handler := inferenceconnect.NewInferenceServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	_, err := (HTTPGateway{BaseURL: server.URL}).Generate(context.Background(), GatewayRequest{
		Role: "write.default", Instruction: "Write a short response.", Query: "unclassified source",
		Strategy: samplerDirect, K: 1, MaxOutputTokens: 128, Locality: "remote_only",
	})
	if err == nil || !strings.Contains(err.Error(), "explicit privacy classification") {
		t.Fatalf("remote refusal error = %v", err)
	}
	if fixture.request == nil || fixture.request.GetProfile() != sharedv1.Profile_PROFILE_REMOTE_ONLY {
		t.Fatalf("profile request = %v, want remote_only", fixture.request)
	}
	if fixture.request.GetPrivacyClass() != sharedv1.PrivacyClass_PRIVACY_CLASS_UNSPECIFIED {
		t.Fatalf("privacy class = %s, want UNSPECIFIED because Prose has no source", fixture.request.GetPrivacyClass())
	}
}
