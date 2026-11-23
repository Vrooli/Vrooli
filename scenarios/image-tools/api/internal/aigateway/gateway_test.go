package aigateway

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"connectrpc.com/connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"

	"image-tools/internal/backends"
	"image-tools/internal/storage"
	"image-tools/internal/technique"
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
	submitted      *routingv1.SubmitMediaRequest
}

func (h *fakeRoutingHandler) SubmitMedia(_ context.Context, req *connect.Request[routingv1.SubmitMediaRequest]) (*connect.Response[routingv1.SubmitMediaResponse], error) {
	h.submitted = req.Msg
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
	provider := NewProvider(media, nil)

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
	if media.request.OutputPath != out || len(media.request.InputFiles) != 0 {
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
	if _, ok := EvidenceVariationFromMeta(0, result.Meta); !ok {
		t.Fatalf("successful gateway execution evidence missing: %#v", result.Meta)
	}
}

func TestAIGatewayProviderRetainsGatewayReceiptWhenGenerateReturnsError(t *testing.T) {
	wantErr := context.Canceled
	media := &fakeMediaGateway{result: mediaGenerationResult{
		ExecutionID: "gw-accepted", GatewayStatus: "MEDIA_EXECUTION_STATUS_FAILED", CostKnown: true, CostUSD: 0, Outcome: "failed",
	}, err: wantErr}
	provider := NewProvider(media, nil)
	result, err := provider.Execute(context.Background(), backends.Request{
		Operation: "text_to_image", Params: map[string]string{"prompt": "x"},
		Output: storage.OutputTarget{LocalPath: filepath.Join(t.TempDir(), "not-written.png")},
	})
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("error=%v", err)
	}
	entry, ok := EvidenceVariationFromMeta(0, result.Meta)
	if !ok || entry.ExecutionID != "gw-accepted" || entry.Status != "MEDIA_EXECUTION_STATUS_FAILED" || entry.Outcome != "failed" || entry.CostState != "known" || entry.CostUSD == nil || *entry.CostUSD != 0 {
		t.Fatalf("partial gateway evidence = %+v, metadata=%#v", entry, result.Meta)
	}
}

func TestAIGatewayProviderMarksLocalOutputFailureButKeepsKnownReceipt(t *testing.T) {
	media := &fakeMediaGateway{result: mediaGenerationResult{
		ExecutionID: "gw-written", GatewayStatus: "SUCCEEDED", CostKnown: true, CostUSD: 0.25, Outcome: "succeeded",
	}}
	provider := NewProvider(media, nil)
	result, err := provider.Execute(context.Background(), backends.Request{
		Operation: "text_to_image", Params: map[string]string{"prompt": "x"},
		Output: storage.OutputTarget{LocalPath: filepath.Join(t.TempDir(), "missing.png")},
	})
	if err == nil {
		t.Fatal("expected absent output error")
	}
	entry, ok := EvidenceVariationFromMeta(0, result.Meta)
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
	provider := NewProvider(media, nil)

	result, err := provider.Execute(context.Background(), backends.Request{
		Operation: "edit_instruct",
		InputKeys: []string{source},
		Params:    map[string]string{"prompt": "make it brighter"},
		Output:    storage.OutputTarget{LocalPath: out},
	})
	if err != nil {
		t.Fatal(err)
	}
	if media.request.Role != "image.edit.default" || len(media.request.InputFiles) != 1 || media.request.InputFiles[0] != source {
		t.Fatalf("gateway edit intent = %#v", media.request)
	}
	if _, ok := result.Meta[MetaCostUSD]; ok {
		t.Fatalf("unknown Gateway cost must remain absent: %#v", result.Meta)
	}
	if _, ok := result.Meta[MetaCostProvenance]; ok {
		t.Fatalf("unknown Gateway provenance must remain absent: %#v", result.Meta)
	}
}

func TestAIGatewayProvider_FaceSwapSendsBothImagesAndAnIdentityRole(t *testing.T) {
	dir := t.TempDir()
	target, reference, out := filepath.Join(dir, "target.png"), filepath.Join(dir, "reference.png"), filepath.Join(dir, "out.png")
	for _, f := range []string{target, reference, out} {
		if err := os.WriteFile(f, []byte("image"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	media := &fakeMediaGateway{result: mediaGenerationResult{MediaType: "image/png"}}
	provider := NewProvider(media, nil)
	request := backends.Request{
		Operation: "face_swap",
		InputKeys: []string{target, "", reference},
		Params:    map[string]string{},
		Output:    storage.OutputTarget{LocalPath: out},
	}
	if _, err := provider.Execute(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	got := media.request
	if got.Role != "image.edit.identity" {
		t.Errorf("a face swap is an identity edit; role = %q", got.Role)
	}
	if len(got.InputFiles) != 2 || got.InputFiles[0] != target || got.InputFiles[1] != reference {
		t.Errorf("inputs must be the target image then the face reference, got %v", got.InputFiles)
	}
	if got.Prompt != FaceSwapInstruction {
		t.Errorf("an empty prompt must default to the face swap instruction, got %q", got.Prompt)
	}
	request.InputKeys = []string{target}
	if _, err := provider.Execute(context.Background(), request); err == nil {
		t.Fatal("a cloud face swap without the reference image must fail")
	}
}

// Colorize is served by a general editor (measured on a real photograph against
// the colour original: better than the dedicated DDColor models), so it needs the
// identity role and a default instruction, and no second image.
func TestAIGatewayProvider_ColorizeSendsOneImageAnIdentityRoleAndTheDefaultInstruction(t *testing.T) {
	source, out := writeImage(t, 768, 1024), writeImage(t, 896, 1200)
	media := &fakeMediaGateway{result: mediaGenerationResult{MediaType: "image/png"}}
	provider := NewProvider(media, nil)
	if !slices.Contains(provider.Operations(), "colorize") {
		t.Fatalf("the cloud provider must declare colorize, has %v", provider.Operations())
	}
	if _, err := provider.Execute(context.Background(), backends.Request{
		Operation: "colorize",
		InputKeys: []string{source},
		Params:    map[string]string{},
		Output:    storage.OutputTarget{LocalPath: out},
	}); err != nil {
		t.Fatal(err)
	}
	got := media.request
	if got.Role != "image.edit.identity" || got.Prompt != technique.ColorizeInstruction || len(got.InputFiles) != 1 || got.InputFiles[0] != source {
		t.Errorf("colorize request = role %q prompt %q inputs %v", got.Role, got.Prompt, got.InputFiles)
	}
}

// The provider answered a 1152x1536 portrait with a 1408x768 picture of a
// different woman in a different scene. A result that does not have the photo's
// shape is not a colourisation and is refused, while a provider's own rounding
// (768x1024 returned as 896x1200) is accepted.
func TestAIGatewayProvider_ColorizeRefusesAPictureOfAnotherShape(t *testing.T) {
	media := &fakeMediaGateway{result: mediaGenerationResult{MediaType: "image/png"}}
	run := func(inW, inH, outW, outH int) error {
		_, err := NewProvider(media, nil).Execute(context.Background(), backends.Request{
			Operation: "colorize",
			InputKeys: []string{writeImage(t, inW, inH)},
			Params:    map[string]string{},
			Output:    storage.OutputTarget{LocalPath: writeImage(t, outW, outH)},
		})
		return err
	}
	if err := run(1152, 1536, 1408, 768); err == nil || !strings.Contains(err.Error(), "did not edit the photo") {
		t.Errorf("a landscape result for a portrait must be refused, got %v", err)
	}
	if err := run(768, 1024, 896, 1200); err != nil {
		t.Errorf("provider rounding must be accepted, got %v", err)
	}
}

func TestAIGatewayProvider_RejectsMissingOutputAndEmptyGatewayOutput(t *testing.T) {
	provider := NewProvider(&fakeMediaGateway{}, nil)
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

// The job's cloud spend cap must reach AI Gateway as the request's cost ceiling:
// without it the provider reserves its full worst case and refuses affordable work.
func TestGatewayMediaClientSubmitsCostCeiling(t *testing.T) {
	h := &fakeRoutingHandler{submitErr: connect.NewError(connect.CodeUnavailable, errors.New("stop after submit"))}
	path, handler := routingconnect.NewRoutingServiceHandler(h)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := &aiGatewayMediaClient{httpClient: server.Client(), resolve: func(context.Context) (string, error) { return server.URL, nil }}
	_, _ = client.Generate(context.Background(), mediaGenerationRequest{
		Operation: "edit_instruct", Role: "image.edit.identity", Prompt: "fixture", OutputPath: filepath.Join(t.TempDir(), "out.png"), MaxCostUSD: 0.25,
	})
	if h.submitted == nil || h.submitted.GetRequest().GetMaxCostUsd() != 0.25 {
		t.Fatalf("submitted request = %+v", h.submitted)
	}
}

func TestMaxCostForRequestIgnoresAbsentAndInvalidCeilings(t *testing.T) {
	cases := map[string]float64{"0.125": 0.125, "": 0, "abc": 0, "0": 0, "-1": 0, "NaN": 0, "+Inf": 0}
	for raw, want := range cases {
		got := maxCostForRequest(backends.Request{Params: map[string]string{ParamMaxCostUSD: raw}})
		if got != want {
			t.Errorf("max cost %q = %v, want %v", raw, got, want)
		}
	}
	if got := maxCostForRequest(backends.Request{}); got != 0 {
		t.Errorf("nil params = %v", got)
	}
}
