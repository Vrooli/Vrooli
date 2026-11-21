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
		Message: "fixture denied provider candidate",
	}}), nil
}

func TestHTTPGatewayCarriesTheSourcedPrivacyContractAndSurfacesRemoteRefusal(t *testing.T) {
	fixture := &privacyCaptureInference{}
	path, handler := inferenceconnect.NewInferenceServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	_, err := (HTTPGateway{BaseURL: server.URL}).Generate(context.Background(), GatewayRequest{
		Role: "write.default", Instruction: "Write a short response.", Query: "unclassified source",
		Strategy: samplerDirect, K: 1, MaxOutputTokens: 128, Locality: "remote_only",
		PrivacyClass: prosePrivacyClass, PrivacySource: prosePrivacySource,
	})
	if err == nil || !strings.Contains(err.Error(), "fixture denied provider candidate") {
		t.Fatalf("gateway error = %v", err)
	}
	if fixture.request == nil || fixture.request.GetProfile() != sharedv1.Profile_PROFILE_REMOTE_ONLY {
		t.Fatalf("profile request = %v, want remote_only", fixture.request)
	}
	if fixture.request.GetPrivacyClass() != sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL {
		t.Fatalf("privacy class = %s, want the Prose private-editorial-content classification", fixture.request.GetPrivacyClass())
	}
	if fixture.request.GetPrivacyProvenance() != prosePrivacySource {
		t.Fatalf("privacy provenance = %q, want Prose Studio data source", fixture.request.GetPrivacyProvenance())
	}
}

func TestHTTPGatewayRefusesMissingPrivacyProvenanceBeforeDispatch(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	_, err := (HTTPGateway{BaseURL: server.URL}).Generate(context.Background(), GatewayRequest{
		Role: "write.default", Instruction: "Write.", Query: "draft", Strategy: samplerDirect, K: 1,
		MaxOutputTokens: 128, Locality: "remote_only",
	})
	if err == nil || !strings.Contains(err.Error(), "requires privacy provenance") {
		t.Fatalf("missing privacy provenance error = %v", err)
	}
	if called {
		t.Fatal("gateway was called without a sourced privacy classification")
	}
}
