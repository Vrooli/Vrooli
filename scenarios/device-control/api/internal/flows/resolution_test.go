package flows

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

type fakeGateway struct {
	response *inferencev1.RunResponse
	err      error
	request  *inferencev1.RunRequest
}

func (f *fakeGateway) Run(_ context.Context, req *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	f.request = req.Msg
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(f.response), nil
}

func testFrame() Frame {
	return Frame{Bytes: []byte("png"), MediaType: "image/png", Width: 960, Height: 540, OriginalWidth: 1920, OriginalHeight: 1080}
}

func TestResolverNormalizesVisionResultAndConvertsToOriginalDeviceCoordinates(t *testing.T) {
	fake := &fakeGateway{response: &inferencev1.RunResponse{
		ValueJson: `{"found":true,"bounds":[0.1,0.2,0.5,0.6],"confidence":0.82}`,
		Provider:  "local",
		Model:     "policy-selected",
	}}
	result, err := NewResolver(fake).Resolve(context.Background(), Request{Target: "settings", Frame: testFrame(), ConfidenceThreshold: 0.7})
	require.NoError(t, err)
	require.Equal(t, "resolved", result.Status)
	require.Equal(t, VisionRung, result.Rung)
	require.Equal(t, []float64{0.1, 0.2, 0.5, 0.6}, result.Bounds)
	require.Equal(t, []int{192, 216, 960, 648}, result.DeviceBounds)
	require.Equal(t, uint32(960), fake.request.GetAttachments()[0].GetWidth())
	require.Empty(t, fake.request.GetSource())
	require.Equal(t, []string{"attempt_vision", "resolved"}, evidenceNames(result.Evidence))
}

func TestResolverFallsBackToVisualAnchorBelowCallerThreshold(t *testing.T) { // [REQ:DVC-P0-006]
	fake := &fakeGateway{response: &inferencev1.RunResponse{ValueJson: `{"found":true,"bounds":[0.1,0.1,0.9,0.9],"confidence":0.31}`}}
	result, err := NewResolver(fake).Resolve(context.Background(), Request{
		Target:              "submit",
		Frame:               testFrame(),
		ConfidenceThreshold: 0.8,
		FallbackBounds:      []float64{0.2, 0.25, 0.4, 0.5},
		FallbackConfidence:  0.99,
	})
	require.NoError(t, err)
	require.True(t, result.FallbackUsed)
	require.Equal(t, VisualAnchorRung, result.Rung)
	require.Equal(t, []int{384, 270, 768, 540}, result.DeviceBounds)
	require.Equal(t, []string{"attempt_vision", "fallback", "resolved"}, evidenceNames(result.Evidence))
	require.Equal(t, "confidence_below_threshold", result.Evidence[1].Reason)
}

func TestResolverReportsTypedUnavailableWithoutProviderFallback(t *testing.T) { // [REQ:DVC-P0-007]
	fake := &fakeGateway{err: &UnavailableError{Reason: "gateway_route_unavailable"}}
	result, err := NewResolver(fake).Resolve(context.Background(), Request{Target: "submit", Frame: testFrame()})
	var unavailable *UnavailableError
	require.ErrorAs(t, err, &unavailable)
	require.Equal(t, "gateway_route_unavailable", unavailable.Reason)
	require.Equal(t, "unavailable", result.Status)
	require.Equal(t, []string{"attempt_vision", "unresolved"}, evidenceNames(result.Evidence))
}

func TestPrepareFrameDownscalesBeforeGatewayAndKeepsOriginalDimensions(t *testing.T) {
	input := image.NewRGBA(image.Rect(0, 0, 2048, 1024))
	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, input))

	frame, err := PrepareFrame(encoded.Bytes(), "image/png", 1024)
	require.NoError(t, err)
	require.Equal(t, 2048, frame.OriginalWidth)
	require.Equal(t, 1024, frame.OriginalHeight)
	require.Equal(t, 1024, frame.Width)
	require.Equal(t, 512, frame.Height)
	require.NotEmpty(t, frame.Bytes)
}

func evidenceNames(events []EvidenceEvent) []string {
	names := make([]string, 0, len(events))
	for _, event := range events {
		names = append(names, event.Name)
	}
	return names
}

func TestEvidenceIsJSONSafe(t *testing.T) {
	value := Result{Status: "resolved", Rung: VisionRung, Evidence: []EvidenceEvent{{Name: "resolved", Rung: VisionRung, Confidence: ptr(0.9)}}}
	_, err := json.Marshal(value)
	require.NoError(t, err)
}
