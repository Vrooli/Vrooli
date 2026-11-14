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
)

func TestManifestCoversRoutingService(t *testing.T) { // [REQ:AIGW-CLI-OPERATIONS]
	cliapp.RequireProtoServiceCoverage(t, testutil.ManifestBytes(t), routingv1.File_ai_gateway_v1_routing_routing_proto, "RoutingService")
}

type previewCaptureClient struct {
	routingconnect.RoutingServiceClient
	request        *routingv1.PreviewRouteRequest
	executeRequest *routingv1.ExecuteRouteRequest
}

func (c *previewCaptureClient) PreviewRoute(_ context.Context, req *connect.Request[routingv1.PreviewRouteRequest]) (*connect.Response[routingv1.PreviewRouteResponse], error) {
	c.request = req.Msg
	return connect.NewResponse(&routingv1.PreviewRouteResponse{Valid: true, SelectedProvider: "ollama"}), nil
}

func (c *previewCaptureClient) ExecuteRoute(_ context.Context, req *connect.Request[routingv1.ExecuteRouteRequest]) (*connect.Response[routingv1.ExecuteRouteResponse], error) {
	c.executeRequest = req.Msg
	return connect.NewResponse(&routingv1.ExecuteRouteResponse{Valid: true}), nil
}

func TestPreviewCarriesCapacityControlsInGatewayMetadata(t *testing.T) {
	flags := map[string]string{
		"kind": "text", "role": "chat.small", "profile": "local-only", "privacy-class": "internal",
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
}

func TestExecuteCarriesExpectedWorkerDigest(t *testing.T) {
	digest := "2a654d98e6fba55d452b7043684e9b57a947e393bbffa62485a7aac05ee4eefd"
	flags := map[string]string{
		"kind": "text", "role": "chat.small", "profile": "local-only", "privacy-class": "internal",
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
