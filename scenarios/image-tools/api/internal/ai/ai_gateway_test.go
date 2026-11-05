package ai

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"

	"image-tools/internal/backends"
	"image-tools/internal/storage"
)

type fakeMediaGateway struct {
	request mediaGenerationRequest
	result  mediaGenerationResult
	err     error
}

type fakeRoutingHandler struct {
	routingconnect.RoutingServiceHandler
	submitResponse *routingv1.SubmitMediaResponse
	submitErr      error
	waitResponse   *routingv1.WaitMediaExecutionResponse
	waitErr        error
	waitRequestID  string
}

func (h *fakeRoutingHandler) SubmitMedia(context.Context, *connect.Request[routingv1.SubmitMediaRequest]) (*connect.Response[routingv1.SubmitMediaResponse], error) {
	if h.submitErr != nil {
		return nil, h.submitErr
	}
	return connect.NewResponse(h.submitResponse), nil
}

func (h *fakeRoutingHandler) WaitMediaExecution(_ context.Context, req *connect.Request[routingv1.WaitMediaExecutionRequest]) (*connect.Response[routingv1.WaitMediaExecutionResponse], error) {
	h.waitRequestID = req.Msg.GetExecutionId()
	if h.waitErr != nil {
		return nil, h.waitErr
	}
	return connect.NewResponse(h.waitResponse), nil
}

func TestGatewayMediaClientPreservesIDWhenWaitIsInterrupted(t *testing.T) {
	h := &fakeRoutingHandler{
		submitResponse: &routingv1.SubmitMediaResponse{Execution: &routingv1.MediaExecution{ExecutionId: "gw-wait"}},
		waitErr:        connect.NewError(connect.CodeDeadlineExceeded, context.DeadlineExceeded),
	}
	result, err := runGatewayMediaClient(t, h)
	if err == nil || result.ExecutionID != "gw-wait" || result.GatewayStatus != "" || result.Outcome != "outcome_unknown" || result.CostKnown {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if h.waitRequestID != "gw-wait" {
		t.Fatalf("Wait execution id = %q", h.waitRequestID)
	}
}

func TestGatewayMediaClientAmbiguousSubmitHasNoFabricatedID(t *testing.T) {
	h := &fakeRoutingHandler{submitErr: connect.NewError(connect.CodeUnavailable, errors.New("submit response lost"))}
	result, err := runGatewayMediaClient(t, h)
	if err == nil || result.ExecutionID != "" || result.Outcome != "outcome_unknown" || result.CostKnown {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestGatewayMediaClientPreservesTerminalFailureCost(t *testing.T) {
	cost := float64(0)
	h := &fakeRoutingHandler{
		submitResponse: &routingv1.SubmitMediaResponse{Execution: &routingv1.MediaExecution{ExecutionId: "gw-failed"}},
		waitResponse: &routingv1.WaitMediaExecutionResponse{Execution: &routingv1.MediaExecution{
			ExecutionId: "gw-failed", Status: routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_FAILED,
			ActualCostUsd: &cost, ErrorCode: "executor_failed",
		}},
	}
	result, err := runGatewayMediaClient(t, h)
	if err == nil || result.ExecutionID != "gw-failed" || result.GatewayStatus != "MEDIA_EXECUTION_STATUS_FAILED" || result.Outcome != "failed" || !result.CostKnown || result.CostUSD != 0 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func runGatewayMediaClient(t *testing.T, h *fakeRoutingHandler) (mediaGenerationResult, error) {
	t.Helper()
	path, handler := routingconnect.NewRoutingServiceHandler(h)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := &aiGatewayMediaClient{httpClient: server.Client(), resolve: func(context.Context) (string, error) { return server.URL, nil }}
	return client.Generate(context.Background(), mediaGenerationRequest{
		Operation: "text_to_image", Role: "image.generate.default", Prompt: "fixture", OutputPath: filepath.Join(t.TempDir(), "not-written.png"),
	})
}

func (f *fakeMediaGateway) Generate(_ context.Context, req mediaGenerationRequest) (mediaGenerationResult, error) {
	f.request = req
	return f.result, f.err
}

func TestAIGatewayProvider_DelegatesIntentAndPreservesGatewayOutput(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.svg")
	if err := os.WriteFile(out, []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 8 8"></svg>`), 0o600); err != nil {
		t.Fatal(err)
	}
	media := &fakeMediaGateway{result: mediaGenerationResult{MediaType: "image/svg+xml", Model: "recraft/recraft-v4.1-vector", CostKnown: true, CostUSD: 0, ExecutionID: "gw-success", GatewayStatus: "SUCCEEDED", Outcome: "succeeded"}}
	provider := newAIGatewayProvider(media)

	result, err := provider.Execute(context.Background(), backends.Request{
		Operation: "text_to_image",
		Params:    map[string]string{"prompt": "a precise mark", "openrouter_role": "image.generate.logo"},
		Output:    storage.OutputTarget{LocalPath: out},
	})
	if err != nil {
		t.Fatal(err)
	}
	if media.request.Operation != "text_to_image" || media.request.Role != "image.generate.logo" || media.request.Prompt != "a precise mark" {
		t.Fatalf("gateway intent = %#v", media.request)
	}
	if media.request.OutputPath != out || media.request.InputFile != "" {
		t.Fatalf("gateway paths = %#v", media.request)
	}
	if result.Tier != backends.TierBYOK || result.Meta["gateway"] != "ai-gateway" || result.Meta["media_type"] != "image/svg+xml" {
		t.Fatalf("result = %#v", result)
	}
	if result.Meta["model"] != "recraft/recraft-v4.1-vector" {
		t.Fatalf("resolved model metadata = %#v", result.Meta)
	}
	if result.Meta[MetaCostUSD] != "0" || result.Meta[MetaCostProvenance] != ProviderReportedCostProvenance {
		t.Fatalf("known zero cost metadata = %#v", result.Meta)
	}
	if _, ok := gatewayEvidenceVariationFromMeta(0, result.Meta); !ok {
		t.Fatalf("successful gateway execution evidence missing: %#v", result.Meta)
	}
}

func TestAIGatewayProviderRetainsGatewayReceiptWhenGenerateReturnsError(t *testing.T) {
	wantErr := context.Canceled
	media := &fakeMediaGateway{result: mediaGenerationResult{
		ExecutionID: "gw-accepted", GatewayStatus: "MEDIA_EXECUTION_STATUS_FAILED", CostKnown: true, CostUSD: 0, Outcome: "failed",
	}, err: wantErr}
	provider := newAIGatewayProvider(media)
	result, err := provider.Execute(context.Background(), backends.Request{
		Operation: "text_to_image", Params: map[string]string{"prompt": "x"},
		Output: storage.OutputTarget{LocalPath: filepath.Join(t.TempDir(), "not-written.png")},
	})
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("error=%v", err)
	}
	entry, ok := gatewayEvidenceVariationFromMeta(0, result.Meta)
	if !ok || entry.ExecutionID != "gw-accepted" || entry.Status != "MEDIA_EXECUTION_STATUS_FAILED" || entry.Outcome != "failed" || entry.CostState != "known" || entry.CostUSD == nil || *entry.CostUSD != 0 {
		t.Fatalf("partial gateway evidence = %+v, metadata=%#v", entry, result.Meta)
	}
}

func TestAIGatewayProviderMarksLocalOutputFailureButKeepsKnownReceipt(t *testing.T) {
	media := &fakeMediaGateway{result: mediaGenerationResult{
		ExecutionID: "gw-written", GatewayStatus: "SUCCEEDED", CostKnown: true, CostUSD: 0.25, Outcome: "succeeded",
	}}
	provider := newAIGatewayProvider(media)
	result, err := provider.Execute(context.Background(), backends.Request{
		Operation: "text_to_image", Params: map[string]string{"prompt": "x"},
		Output: storage.OutputTarget{LocalPath: filepath.Join(t.TempDir(), "missing.png")},
	})
	if err == nil {
		t.Fatal("expected absent output error")
	}
	entry, ok := gatewayEvidenceVariationFromMeta(0, result.Meta)
	if !ok || entry.ExecutionID != "gw-written" || entry.Outcome != "failed" || entry.CostState != "known" || entry.CostUSD == nil || *entry.CostUSD != 0.25 {
		t.Fatalf("partial gateway evidence = %+v, metadata=%#v", entry, result.Meta)
	}
}

func TestAIGatewayProvider_ForwardsEditInput(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(source, []byte("\x89PNG\r\n\x1a\nsource"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "out.png")
	if err := os.WriteFile(out, []byte("output"), 0o600); err != nil {
		t.Fatal(err)
	}
	media := &fakeMediaGateway{result: mediaGenerationResult{MediaType: "image/png"}}
	provider := newAIGatewayProvider(media)

	result, err := provider.Execute(context.Background(), backends.Request{
		Operation: "edit_instruct",
		InputKeys: []string{source},
		Params:    map[string]string{"prompt": "make it brighter"},
		Output:    storage.OutputTarget{LocalPath: out},
	})
	if err != nil {
		t.Fatal(err)
	}
	if media.request.Role != "image.edit.default" || media.request.InputFile != source {
		t.Fatalf("gateway edit intent = %#v", media.request)
	}
	if _, ok := result.Meta[MetaCostUSD]; ok {
		t.Fatalf("unknown Gateway cost must remain absent: %#v", result.Meta)
	}
	if _, ok := result.Meta[MetaCostProvenance]; ok {
		t.Fatalf("unknown Gateway provenance must remain absent: %#v", result.Meta)
	}
}

func TestAIGatewayProvider_RejectsMissingOutputAndEmptyGatewayOutput(t *testing.T) {
	provider := newAIGatewayProvider(&fakeMediaGateway{})
	if _, err := provider.Execute(context.Background(), backends.Request{Operation: "text_to_image", Params: map[string]string{"prompt": "x"}}); err == nil {
		t.Fatal("expected missing output path error")
	}

	out := filepath.Join(t.TempDir(), "empty.png")
	if err := os.WriteFile(out, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Execute(context.Background(), backends.Request{
		Operation: "text_to_image",
		Params:    map[string]string{"prompt": "x"},
		Output:    storage.OutputTarget{LocalPath: out},
	}); err == nil {
		t.Fatal("expected empty gateway output error")
	}
}

func TestRoleForRequest(t *testing.T) {
	if got := roleForRequest(backends.Request{Operation: "text_to_image"}); got != "image.generate.default" {
		t.Fatalf("generate role = %q", got)
	}
	if got := roleForRequest(backends.Request{Operation: "edit_instruct"}); got != "image.edit.default" {
		t.Fatalf("edit role = %q", got)
	}
	if got := roleForRequest(backends.Request{Params: map[string]string{"openrouter_role": "image.generate.logo"}}); got != "image.generate.logo" {
		t.Fatalf("explicit role = %q", got)
	}
}
