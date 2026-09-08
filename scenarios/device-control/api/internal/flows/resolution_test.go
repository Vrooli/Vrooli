package flows

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"strings"
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
	require.Equal(t, []string{"skip", "skip", "attempt_vision", "resolved"}, evidenceNames(result.Evidence))
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
	require.Equal(t, []string{"skip", "skip", "attempt_vision", "fallback", "resolved"}, evidenceNames(result.Evidence))
	require.Equal(t, "anchor_not_found", result.Evidence[1].Reason)
}

func TestResolverReportsTypedUnavailableWithoutProviderFallback(t *testing.T) { // [REQ:DVC-P0-007]
	fake := &fakeGateway{err: &UnavailableError{Reason: "gateway_route_unavailable"}}
	result, err := NewResolver(fake).Resolve(context.Background(), Request{Target: "submit", Frame: testFrame()})
	var unavailable *UnavailableError
	require.ErrorAs(t, err, &unavailable)
	require.Equal(t, "gateway_route_unavailable", unavailable.Reason)
	require.Equal(t, "unavailable", result.Status)
	require.Equal(t, []string{"skip", "skip", "attempt_vision", "unresolved"}, evidenceNames(result.Evidence))
}

func TestResolverSemanticHitSkipsInference(t *testing.T) {
	fake := &fakeGateway{}
	result, err := NewResolver(fake).Resolve(context.Background(), Request{Target: "hello-mobile-input", Frame: testFrame(), Semantic: func(context.Context, string) (SemanticMatch, error) {
		return SemanticMatch{Bounds: []float64{.1, .2, .3, .4}, Confidence: 1}, nil
	}})
	require.NoError(t, err)
	require.Equal(t, "semantic", result.Rung)
	require.Nil(t, fake.request)
}

func TestResolverAnchorAnswersBeforeVision(t *testing.T) {
	fake := &fakeGateway{response: &inferencev1.RunResponse{ValueJson: `{"found":true,"bounds":[.8,.8,.9,.9],"confidence":1}`}}
	anchors := NewAnchorStoreAt("")
	_, err := anchors.Create("submit", "hello-mobile-input", []float64{.1, .2, .3, .4}, .95)
	require.NoError(t, err)
	result, err := NewResolver(fake).Resolve(context.Background(), Request{Target: "hello-mobile-input", Frame: testFrame(), Anchors: anchors})
	require.NoError(t, err)
	require.Equal(t, VisualAnchorRung, result.Rung)
	require.Nil(t, fake.request)
	require.Equal(t, []string{"skip", "resolved"}, evidenceNames(result.Evidence))
}

func TestVisionPromotionReplaysThroughAnchorWithoutGateway(t *testing.T) {
	fake := &fakeGateway{response: &inferencev1.RunResponse{ValueJson: `{"found":true,"bounds":[0.1,0.2,0.3,0.4],"confidence":0.95}`}}
	anchors := NewAnchorStoreAt("")
	first, err := NewResolver(fake).Resolve(context.Background(), Request{Target: "hello-mobile-input", Frame: testFrame(), Anchors: anchors})
	require.NoError(t, err)
	require.Equal(t, VisionRung, first.Rung)
	fake.request = nil
	second, err := NewResolver(fake).Resolve(context.Background(), Request{Target: "hello-mobile-input", Frame: testFrame(), Anchors: anchors})
	require.NoError(t, err)
	require.Equal(t, VisualAnchorRung, second.Rung)
	require.Nil(t, fake.request)
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

func TestValidateAgentPlanRejectsUntrustedOrMalformedShapes(t *testing.T) {
	declared := []string{"media-pause", "key", "property-set"}
	tests := []struct {
		name string
		plan AgentPlan
		want string
	}{
		{name: "goal and step", plan: AgentPlan{GoalMet: true, StepKind: "key"}, want: "goal_met cannot include"},
		{name: "undeclared", plan: AgentPlan{StepKind: "host-shell"}, want: "undeclared"},
		{name: "media action required", plan: AgentPlan{StepKind: "media-pause"}, want: "requires an action"},
		{name: "key value required", plan: AgentPlan{StepKind: "key"}, want: "requires a value"},
		{name: "oversized value", plan: AgentPlan{StepKind: "key", Value: strings.Repeat("x", maxAgentValueBytes)}, want: "exceeds"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.ErrorContains(t, ValidateAgentPlan(tt.plan, declared), tt.want)
		})
	}
	require.NoError(t, ValidateAgentPlan(AgentPlan{StepKind: "media-pause", Action: "pause"}, declared))
	require.NoError(t, ValidateAgentPlan(AgentPlan{GoalMet: true}, declared))
}

func TestGatewayPlannerUsesBoundedTypedContract(t *testing.T) {
	fake := &fakeGateway{response: &inferencev1.RunResponse{ValueJson: `{"goal_met":false,"step_kind":"media-pause","action":"pause"}`}}
	planner := NewGatewayPlanner(fake)
	plan, err := planner.Plan(context.Background(), AgentWorld{Goal: "pause playback", StepKinds: []string{"media-pause"}})
	require.NoError(t, err)
	require.Equal(t, "media-pause", plan.StepKind)
	require.EqualValues(t, defaultPlanTokenCap, fake.request.GetMaxOutputTokens())
	require.NotContains(t, fake.request.GetSchemaJson(), "additionalProperties")
	require.NotContains(t, fake.request.GetSchemaJson(), "maxLength")
}

func TestGatewayPlannerRejectsUnknownAndContradictoryModelFields(t *testing.T) {
	for _, response := range []string{
		`{"goal_met":true,"step_kind":"media-pause"}`,
		`{"goal_met":true,"unexpected":"instruction"}`,
		`{"goal_met":false,"step_kind":"host-shell"}`,
	} {
		fake := &fakeGateway{response: &inferencev1.RunResponse{ValueJson: response}}
		_, err := NewGatewayPlanner(fake).Plan(context.Background(), AgentWorld{Goal: "bounded task", StepKinds: []string{"media-pause"}})
		require.Error(t, err, response)
	}
}

func TestGatewayPlannerRejectsOversizedModelOutput(t *testing.T) {
	fake := &fakeGateway{response: &inferencev1.RunResponse{ValueJson: strings.Repeat("x", maxAgentValueBytes+1)}}
	_, err := NewGatewayPlanner(fake).Plan(context.Background(), AgentWorld{Goal: "bounded task", StepKinds: []string{"media-pause"}})
	require.ErrorContains(t, err, "agent plan exceeds")
}

func TestAgentPolicyRequiresExternalConfirmationAndDoesNotTrustRiskLabels(t *testing.T) {
	for _, test := range []struct {
		stepKind string
		risk     string
	}{
		{stepKind: "credential", risk: "credential"},
		{stepKind: "clipboard-read", risk: "credential"},
		{stepKind: "grant-permission", risk: "permission"},
		{stepKind: "share", risk: "external"},
	} {
		require.Equal(t, test.risk, AgentStepRisk(test.stepKind, ""))
	}

	policy := DefaultAgentPolicy()
	require.ErrorContains(t, AgentPolicyViolation(policy, AgentPlan{StepKind: "share", Action: "safe"}, false), "confirmation required")
	require.NoError(t, AgentPolicyViolation(policy, AgentPlan{StepKind: "share", Action: "safe"}, true))
	policy.AllowPermissionChanges = false
	require.ErrorContains(t, AgentPolicyViolation(policy, AgentPlan{StepKind: "grant-permission", Action: "not-a-risk"}, true), "permission")
}
