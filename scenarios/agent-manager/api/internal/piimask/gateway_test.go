package piimask

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type fakeRoutingClient struct {
	request  *routingv1.ExecuteRouteRequest
	response *routingv1.ExecuteRouteResponse
	err      error
}

func (f *fakeRoutingClient) ExecuteRoute(_ context.Context, req *connect.Request[routingv1.ExecuteRouteRequest]) (*connect.Response[routingv1.ExecuteRouteResponse], error) {
	f.request = req.Msg
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(f.response), nil
}

func TestGatewayDetectorSendsDetectPIIRequest(t *testing.T) {
	client := &fakeRoutingClient{response: &routingv1.ExecuteRouteResponse{
		Valid:      true,
		OutputText: `{"spans":[{"category":"EMAIL_ADDRESS","start":11,"end":26,"text":"bob@example.com","confidence":0.99,"source":"regex"}],"model_available":false}`,
	}}
	detector := &GatewayDetector{Client: client}

	result, err := detector.Detect(context.Background(), "mail me at bob@example.com")
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(result.Spans) != 1 || result.Spans[0].Category != "EMAIL_ADDRESS" {
		t.Fatalf("unexpected spans: %+v", result.Spans)
	}
	// The governed route must carry the DETECT_PII kind, the local-only profile
	// (PII must not leave the host), and the detector role.
	req := client.request.GetRequest()
	if req.GetKind() != sharedv1.RequestKind_REQUEST_KIND_DETECT_PII {
		t.Fatalf("expected DETECT_PII kind, got %s", req.GetKind())
	}
	if req.GetProfile() != sharedv1.Profile_PROFILE_LOCAL_ONLY {
		t.Fatalf("expected local-only profile, got %s", req.GetProfile())
	}
	if req.GetRole() != detectRole {
		t.Fatalf("expected role %q, got %q", detectRole, req.GetRole())
	}
	if client.request.GetInputText() != "mail me at bob@example.com" {
		t.Fatalf("input text not forwarded: %q", client.request.GetInputText())
	}
}

func TestGatewayDetectorErrorsOnTransportFailure(t *testing.T) {
	detector := &GatewayDetector{Client: &fakeRoutingClient{err: errors.New("gateway down")}}
	if _, err := detector.Detect(context.Background(), "pii here"); err == nil {
		t.Fatalf("a transport failure must surface as an error so masking fails closed")
	}
}

func TestGatewayDetectorErrorsOnRefusedRoute(t *testing.T) {
	detector := &GatewayDetector{Client: &fakeRoutingClient{response: &routingv1.ExecuteRouteResponse{
		Valid:         false,
		PolicyReasons: []string{"no eligible provider route"},
	}}}
	if _, err := detector.Detect(context.Background(), "pii here"); err == nil {
		t.Fatalf("a refused route must error so masking fails closed")
	}
}

func TestGatewayDetectorErrorsOnEmptyOutput(t *testing.T) {
	// A valid response with no output text means nothing served the detector;
	// that is a detection failure, not clean text.
	detector := &GatewayDetector{Client: &fakeRoutingClient{response: &routingv1.ExecuteRouteResponse{Valid: true, OutputText: ""}}}
	if _, err := detector.Detect(context.Background(), "pii here"); err == nil {
		t.Fatalf("an empty detector output must error so masking fails closed")
	}
}
