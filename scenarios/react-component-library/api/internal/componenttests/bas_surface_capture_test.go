package componenttests

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
	captureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture/captureconnect"
)

type surfaceCaptureServer struct {
	calls   int
	state   capturev1.GetCaptureByRequestTokenResponse_State
	request *capturev1.CaptureRequest
	result  *capturev1.CaptureResponse
}

func (s *surfaceCaptureServer) Capture(_ context.Context, request *connect.Request[capturev1.CaptureRequest]) (*connect.Response[capturev1.CaptureResponse], error) {
	s.calls++
	s.request = request.Msg
	if s.result == nil {
		if s.calls == 1 {
			s.state = capturev1.GetCaptureByRequestTokenResponse_STATE_IN_PROGRESS
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("lost response"))
		}
		s.result = completedSurfaceCapture()
	}
	s.state = capturev1.GetCaptureByRequestTokenResponse_STATE_COMPLETED
	return connect.NewResponse(s.result), nil
}

func (s *surfaceCaptureServer) GetCaptureByRequestToken(_ context.Context, request *connect.Request[capturev1.GetCaptureByRequestTokenRequest]) (*connect.Response[capturev1.GetCaptureByRequestTokenResponse], error) {
	if request.Msg.GetRequestToken() != s.request.GetRequestToken() {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("not found"))
	}
	response := &capturev1.GetCaptureByRequestTokenResponse{State: s.state, ExecutionId: "exec-123"}
	if s.state == capturev1.GetCaptureByRequestTokenResponse_STATE_COMPLETED {
		response.Capture = s.result
	}
	return connect.NewResponse(response), nil
}

func completedSurfaceCapture() *capturev1.CaptureResponse {
	return &capturev1.CaptureResponse{ExecutionId: "exec-123", Artifacts: []*capturev1.CaptureArtifact{
		{Type: capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT, Reference: "bas://exec-123/screenshot"},
		{Type: capturev1.CaptureType_CAPTURE_TYPE_ACCESSIBILITY, Reference: "bas://exec-123/accessibility"},
	}}
}

func surfaceCaptureTestServer(t *testing.T, service captureconnect.CaptureServiceHandler) *httptest.Server {
	t.Helper()
	path, handler := captureconnect.NewCaptureServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func TestBASCaptureSurfaceRetriesSameTokenAndReturnsDurableReferences(t *testing.T) {
	server := &surfaceCaptureServer{}
	httpServer := surfaceCaptureTestServer(t, server)
	executor := BASCaptureExecutor{BASBaseURL: httpServer.URL, HTTPClient: httpServer.Client()}
	request := &capturev1.CaptureRequest{RequestToken: "rcl-state-abc123", Url: "scenario=react-component-library,path=/design?workspace=ws&page=page&display=phone&state=empty"}
	response, err := executor.CaptureSurface(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, 2, server.calls)
	require.Equal(t, request.GetRequestToken(), server.request.GetRequestToken())
	require.Equal(t, "exec-123", response.GetExecutionId())
	require.Len(t, response.GetArtifacts(), 2)
}

func TestBASCaptureSurfaceRejectsMissingAccessibilityReference(t *testing.T) {
	server := &surfaceCaptureServer{result: &capturev1.CaptureResponse{ExecutionId: "exec-123", Artifacts: []*capturev1.CaptureArtifact{
		{Type: capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT, Reference: "bas://exec-123/screenshot"},
	}}}
	server.state = capturev1.GetCaptureByRequestTokenResponse_STATE_COMPLETED
	httpServer := surfaceCaptureTestServer(t, server)
	executor := BASCaptureExecutor{BASBaseURL: httpServer.URL, HTTPClient: httpServer.Client()}
	_, err := executor.CaptureSurface(context.Background(), &capturev1.CaptureRequest{RequestToken: "rcl-state-abc123", Url: "https://rcl.test/design"})
	require.ErrorContains(t, err, "both screenshot and accessibility")
}
