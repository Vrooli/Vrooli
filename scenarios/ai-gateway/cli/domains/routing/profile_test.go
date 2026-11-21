package routing

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
)

type profileCaptureClient struct {
	routingconnect.RoutingServiceClient
	request *routingv1.ListRouteEvidenceRequest
	events  []*routingv1.RouteEvidence
}

func (c *profileCaptureClient) ListRouteEvidence(_ context.Context, req *connect.Request[routingv1.ListRouteEvidenceRequest]) (*connect.Response[routingv1.ListRouteEvidenceResponse], error) {
	c.request = req.Msg
	return connect.NewResponse(&routingv1.ListRouteEvidenceResponse{Events: c.events}), nil
}

func TestProfileEvidenceGroupsByRoleAndModelWithNearestRank(t *testing.T) {
	rows := profileEvidence([]*routingv1.RouteEvidence{
		{TaskRole: "chat.small", SelectedModel: "compact", SelectedProvider: "ollama", Status: "succeeded", LatencyMs: 10, ProviderExecutionDurationNs: 10, CreatedAt: "2026-01-01T00:00:00Z"},
		{TaskRole: "chat.small", SelectedModel: "compact", SelectedProvider: "ollama", Status: "failed", LatencyMs: 20, ProviderExecutionDurationNs: 20, CreatedAt: "2026-01-01T00:00:01Z"},
		{Role: "chat.small", SelectedModel: "compact", SelectedProvider: "ollama", Status: "blocked", LatencyMs: 100, ProviderExecutionDurationNs: 100, CreatedAt: "2026-01-01T00:00:02Z"},
		{TaskRole: "chat.small", SelectedModel: "large", SelectedProvider: "ollama", Status: "succeeded", LatencyMs: 7, CreatedAt: "2026-01-01T00:00:03Z"},
	})
	if len(rows) != 2 {
		t.Fatalf("profile rows = %d, want 2: %v", len(rows), rows)
	}
	if !strings.Contains(rows[0], "role=chat.small model=compact provider=ollama n=3 succeeded=1 failed=1 blocked=1 latency_ms[p50=20 p95=100 min=10 max=100]") {
		t.Fatalf("compact profile row missing expected counts/nearest-rank latency: %s", rows[0])
	}
	if !strings.Contains(rows[0], "provider[p50=20 p95=100 n=3 missing=0]") {
		t.Fatalf("provider stage profile missing: %s", rows[0])
	}
	if !strings.Contains(rows[1], "model=large provider=ollama n=1") {
		t.Fatalf("second model bucket = %s", rows[1])
	}
}

func TestEvidenceProfileReadsMetadataOnlyBoundedWindow(t *testing.T) {
	flags := map[string]string{"scenario": "source-ledger", "limit": "100"}
	flagSchema := []cliapp.Flag{{Name: "scenario"}, {Name: "limit"}}
	stdout := &bytes.Buffer{}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: flagSchema}, Flags: flags, JSON: true, Stdout: stdout,
	})
	client := &profileCaptureClient{events: []*routingv1.RouteEvidence{
		{TaskRole: "embedding.default", SelectedModel: "nomic-embed-text:latest", SelectedProvider: "ollama", Status: "succeeded", LatencyMs: 1865, CreatedAt: "2026-11-20T19:07:00Z"},
	}}
	if err := (&handlers{client: client}).evidenceProfile(ctx); err != nil {
		t.Fatalf("evidence profile: %v", err)
	}
	if client.request == nil || client.request.GetLimit() != 100 || client.request.GetScenario() != "source-ledger" {
		t.Fatalf("route evidence request = %+v", client.request)
	}
	if !strings.Contains(stdout.String(), "role=embedding.default") || strings.Contains(stdout.String(), "prompt") {
		t.Fatalf("profile output should contain metadata role only: %s", stdout.String())
	}
}

func TestEvidenceProfileRejectsLimitAboveAPIWindow(t *testing.T) {
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "limit"}}}, Flags: map[string]string{"limit": "101"},
	})
	if err := (&handlers{client: &profileCaptureClient{}}).evidenceProfile(ctx); err == nil || !strings.Contains(err.Error(), "100-row maximum") {
		t.Fatalf("profile error = %v, want API limit refusal", err)
	}
}

func TestProfileEvidenceSanitizesUnboundedLabels(t *testing.T) {
	rows := profileEvidence([]*routingv1.RouteEvidence{{
		TaskRole: "chat.small\nforged", SelectedModel: "model\tname", SelectedProvider: "ollama",
		Status: "succeeded", LatencyMs: 5,
	}})
	if len(rows) != 1 || strings.ContainsAny(rows[0], "\r\n\t") {
		t.Fatalf("profile row contains control characters: %q", rows)
	}
	if !strings.Contains(rows[0], "role=chat.small forged") || !strings.Contains(rows[0], "model=model name") {
		t.Fatalf("profile row did not normalize labels: %s", rows[0])
	}
}
