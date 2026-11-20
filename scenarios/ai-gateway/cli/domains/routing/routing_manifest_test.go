package routing

import (
	"bytes"
	"context"
	"testing"

	testutil "ai-gateway/cli/internal/testutil"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

func TestManifestCoversRoutingService(t *testing.T) { // [REQ:AIGW-CLI-OPERATIONS]
	cliapp.RequireProtoServiceCoverage(t, testutil.ManifestBytes(t), routingv1.File_ai_gateway_v1_routing_routing_proto, "RoutingService")
}

type previewCaptureClient struct {
	routingconnect.RoutingServiceClient
	request        *routingv1.PreviewRouteRequest
	executeRequest *routingv1.ExecuteRouteRequest
	mediaRequest   *routingv1.SubmitMediaRequest
}

func (c *previewCaptureClient) PreviewRoute(_ context.Context, req *connect.Request[routingv1.PreviewRouteRequest]) (*connect.Response[routingv1.PreviewRouteResponse], error) {
	c.request = req.Msg
	return connect.NewResponse(&routingv1.PreviewRouteResponse{Valid: true, SelectedProvider: "ollama"}), nil
}

func (c *previewCaptureClient) ExecuteRoute(_ context.Context, req *connect.Request[routingv1.ExecuteRouteRequest]) (*connect.Response[routingv1.ExecuteRouteResponse], error) {
	c.executeRequest = req.Msg
	return connect.NewResponse(&routingv1.ExecuteRouteResponse{Valid: true}), nil
}

func (c *previewCaptureClient) SubmitMedia(_ context.Context, req *connect.Request[routingv1.SubmitMediaRequest]) (*connect.Response[routingv1.SubmitMediaResponse], error) {
	c.mediaRequest = req.Msg
	return connect.NewResponse(&routingv1.SubmitMediaResponse{Execution: &routingv1.MediaExecution{ExecutionId: "media-id", Status: routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_QUEUED}}), nil
}

func TestPreviewCarriesCapacityControlsInGatewayMetadata(t *testing.T) {
	flags := map[string]string{
		"kind": "text", "role": "chat.small", "profile": "local-only", "privacy-class": "internal", "privacy-provenance": "cli_test.user_text",
		"operation": "test", "scenario": "test", "timeout-ms": "90000", "max-output-tokens": "180",
		"max-cost-usd": "0", "required-vram-bytes": "6442450944", "request-id": "test-request",
	}
	flagSchema := make([]cliapp.Flag, 0, len(flags))
	for name := range flags {
		flagSchema = append(flagSchema, cliapp.Flag{Name: name})
	}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: flagSchema}, Flags: flags, JSON: true, Stdout: &bytes.Buffer{},
	})
	client := &previewCaptureClient{}
	h := &handlers{client: client}
	if err := h.preview(ctx); err != nil {
		t.Fatalf("preview: %v", err)
	}
	if client.request == nil || client.request.GetRequest() == nil {
		t.Fatal("preview did not send a GatewayRequest")
	}
	metadata := client.request.GetRequest().GetMetadata()
	if metadata["required_vram_bytes"] != "6442450944" || metadata["capacity_allow_reclaim"] != "false" {
		t.Fatalf("capacity metadata = %#v, want explicit no-reclaim 6 GiB", metadata)
	}
	if client.request.GetRequest().GetPrivacyProvenance() != flags["privacy-provenance"] {
		t.Fatalf("privacy provenance = %q, want explicit caller source %q", client.request.GetRequest().GetPrivacyProvenance(), flags["privacy-provenance"])
	}
}

func TestExecuteCarriesExpectedWorkerDigest(t *testing.T) {
	digest := "2a654d98e6fba55d452b7043684e9b57a947e393bbffa62485a7aac05ee4eefd"
	flags := map[string]string{
		"kind": "text", "role": "chat.small", "profile": "local-only", "privacy-class": "internal", "privacy-provenance": "cli_test.user_text",
		"operation": "test", "scenario": "test", "timeout-ms": "90000", "max-output-tokens": "180",
		"max-cost-usd": "0", "required-vram-bytes": "6442450944", "expected-worker-digest": digest,
		"request-id": "test-request", "input-text": "frozen trial input",
	}
	flagSchema := make([]cliapp.Flag, 0, len(flags))
	for name := range flags {
		flagSchema = append(flagSchema, cliapp.Flag{Name: name})
	}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: flagSchema}, Flags: flags, JSON: true, Stdout: &bytes.Buffer{},
	})
	client := &previewCaptureClient{}
	if err := (&handlers{client: client}).execute(ctx); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if client.executeRequest == nil || client.executeRequest.GetRequest() == nil {
		t.Fatal("execute did not send a GatewayRequest")
	}
	if got := client.executeRequest.GetRequest().GetMetadata()["expected_worker_digest"]; got != digest {
		t.Fatalf("expected worker digest metadata = %q, want %q", got, digest)
	}
	if got := client.executeRequest.GetInputText(); got != flags["input-text"] {
		t.Fatalf("input text = %q, want %q", got, flags["input-text"])
	}
}

func TestMediaSubmitCarriesGatewayRequestPromptCountAndOutputReference(t *testing.T) {
	flags := map[string]string{
		"kind": "image", "role": "image.generate.default", "prompt": "a lighthouse",
		"output-count": "2", "idempotency-key": "media-key-1", "output-reference": "/tmp/ai-gateway-media.png",
		"profile": "local-first", "privacy-class": "internal", "privacy-provenance": "cli_test.user_text",
		"operation": "test.media", "scenario": "test", "timeout-ms": "90000", "max-cost-usd": "0",
		"request-id": "media-request",
	}
	flagSchema := make([]cliapp.Flag, 0, len(flags))
	for name := range flags {
		flagSchema = append(flagSchema, cliapp.Flag{Name: name})
	}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: flagSchema}, Flags: flags, JSON: true, Stdout: &bytes.Buffer{},
	})
	client := &previewCaptureClient{}
	if err := (&handlers{client: client}).mediaSubmit(ctx); err != nil {
		t.Fatalf("media submit: %v", err)
	}
	if client.mediaRequest == nil || client.mediaRequest.GetRequest() == nil {
		t.Fatal("media submit did not send a GatewayRequest")
	}
	if got := client.mediaRequest.GetRequest().GetKind(); got != sharedv1.RequestKind_REQUEST_KIND_IMAGE_GENERATION {
		t.Fatalf("kind = %v", got)
	}
	if got := client.mediaRequest.GetRequest().GetRole(); got != flags["role"] {
		t.Fatalf("role = %q", got)
	}
	if got := client.mediaRequest.GetPrompt(); got != flags["prompt"] {
		t.Fatalf("prompt = %q", got)
	}
	if got := client.mediaRequest.GetOutputCount(); got != 2 {
		t.Fatalf("output count = %d", got)
	}
	if got := client.mediaRequest.GetOutputReference(); got != flags["output-reference"] {
		t.Fatalf("output reference = %q", got)
	}
	if got := client.mediaRequest.GetIdempotencyKey(); got != flags["idempotency-key"] {
		t.Fatalf("idempotency key = %q", got)
	}
}
