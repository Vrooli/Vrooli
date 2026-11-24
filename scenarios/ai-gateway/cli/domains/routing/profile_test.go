package routing

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
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

func TestProfileEvidenceGroupsByRoleCallerAndModelWithNearestRank(t *testing.T) {
	rows := profileEvidence([]*routingv1.RouteEvidence{
		{TaskRole: "chat.small", SelectedResourceRole: "chat.small", Scenario: "agent-manager", Caller: "scenario:agent-manager/api", SelectedModel: "compact", SelectedProvider: "ollama", Status: "succeeded", LatencyMs: 10, ProviderExecutionDurationNs: 10, CreatedAt: "2026-01-01T00:00:00Z"},
		{TaskRole: "chat.small", SelectedResourceRole: "chat.small", Scenario: "agent-manager", Caller: "scenario:agent-manager/api", SelectedModel: "compact", SelectedProvider: "ollama", Status: "failed", LatencyMs: 20, ProviderExecutionDurationNs: 20, CreatedAt: "2026-01-01T00:00:01Z"},
		{Role: "chat.small", SelectedResourceRole: "chat.small", Scenario: "agent-manager", Caller: "scenario:agent-manager/api", SelectedModel: "compact", SelectedProvider: "ollama", Status: "blocked", LatencyMs: 100, ProviderExecutionDurationNs: 100, CreatedAt: "2026-01-01T00:00:02Z"},
		{TaskRole: "chat.small", SelectedResourceRole: "chat.small", Scenario: "web-console", Caller: "scenario:web-console/api", SelectedModel: "compact", SelectedProvider: "ollama", Status: "succeeded", LatencyMs: 30, CreatedAt: "2026-01-01T00:00:02Z"},
		{TaskRole: "chat.small", SelectedResourceRole: "chat.small", Scenario: "agent-manager", Caller: "scenario:agent-manager/api", SelectedModel: "large", SelectedProvider: "ollama", Status: "succeeded", LatencyMs: 7, CreatedAt: "2026-01-01T00:00:03Z"},
		{TaskRole: "chat.small", SelectedResourceRole: "chat.small.v2", Scenario: "agent-manager", Caller: "scenario:agent-manager/api", SelectedModel: "compact", SelectedProvider: "ollama", Status: "succeeded", LatencyMs: 50, CreatedAt: "2026-01-01T00:00:04Z"},
	})
	if len(rows) != 4 {
		t.Fatalf("profile rows = %d, want 4: %v", len(rows), rows)
	}
	if !strings.Contains(rows[0], "role=chat.small resource_role=chat.small scenario=agent-manager caller=scenario:agent-manager/api model=compact provider=ollama n=3 succeeded=1 failed=1 blocked=1 latency_ms[p50=20 p95=100 min=10 max=100]") {
		t.Fatalf("compact profile row missing expected counts/nearest-rank latency: %s", rows[0])
	}
	if !strings.Contains(rows[0], "provider[p50=20 p95=100 n=3 missing=0]") {
		t.Fatalf("provider stage profile missing: %s", rows[0])
	}
	if !strings.Contains(rows[1], "resource_role=chat.small scenario=agent-manager caller=scenario:agent-manager/api model=large provider=ollama n=1") {
		t.Fatalf("second model bucket = %s", rows[1])
	}
	if !strings.Contains(rows[2], "resource_role=chat.small scenario=web-console caller=scenario:web-console/api model=compact provider=ollama n=1") {
		t.Fatalf("third caller bucket = %s", rows[2])
	}
	if !strings.Contains(rows[3], "resource_role=chat.small.v2 scenario=agent-manager caller=scenario:agent-manager/api model=compact provider=ollama n=1") {
		t.Fatalf("fourth resource-role bucket = %s", rows[3])
	}
}

func TestEvidenceProfileReadsMetadataOnlyBoundedWindow(t *testing.T) {
	flags := map[string]string{"scenario": "source-ledger", "limit": "100"}
	flagSchema := []cliapp.Flag{{Name: "scenario"}, {Name: "limit"}, {Name: "input"}}
	stdout := &bytes.Buffer{}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: flagSchema}, Flags: flags, JSON: true, Stdout: stdout,
	})
	client := &profileCaptureClient{events: []*routingv1.RouteEvidence{
		{TaskRole: "embedding.default", SelectedResourceRole: "embedding.default", Scenario: "source-ledger", Caller: "scenario:source-ledger/api", SelectedModel: "nomic-embed-text:latest", SelectedProvider: "ollama", Status: "succeeded", LatencyMs: 1865, CreatedAt: "2026-11-20T19:07:00Z"},
	}}
	if err := (&handlers{client: client}).evidenceProfile(ctx); err != nil {
		t.Fatalf("evidence profile: %v", err)
	}
	if client.request == nil || client.request.GetLimit() != 100 || client.request.GetScenario() != "source-ledger" {
		t.Fatalf("route evidence request = %+v", client.request)
	}
	if !strings.Contains(stdout.String(), "role=embedding.default resource_role=embedding.default scenario=source-ledger caller=scenario:source-ledger/api") || strings.Contains(stdout.String(), "prompt") {
		t.Fatalf("profile output should contain metadata role only: %s", stdout.String())
	}
}

func TestEvidenceProfileRejectsLimitAboveAPIWindow(t *testing.T) {
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "limit"}, {Name: "input"}}}, Flags: map[string]string{"limit": "101"},
	})
	if err := (&handlers{client: &profileCaptureClient{}}).evidenceProfile(ctx); err == nil || !strings.Contains(err.Error(), "100-row maximum") {
		t.Fatalf("profile error = %v, want API limit refusal", err)
	}
}

func TestEvidenceProfileReplaysOneFrozenReceiptWithoutAPICall(t *testing.T) {
	receipt := &routingv1.ListRouteEvidenceResponse{Events: []*routingv1.RouteEvidence{
		{TaskRole: "chat.default", Scenario: "source-ledger", Caller: "scenario:source-ledger/api", SelectedModel: "qwen3.5:9b", SelectedProvider: "ollama", Status: "succeeded", LatencyMs: 25, ProviderRuntimeDurationNs: 17, ProviderAdmissionWaitNs: 3, CreatedAt: "2026-11-24T12:00:00Z"},
		{TaskRole: "chat.default", Scenario: "agent-manager", Caller: "scenario:agent-manager/api", SelectedModel: "qwen3.5:9b", SelectedProvider: "ollama", Status: "succeeded", LatencyMs: 40, CreatedAt: "2026-11-24T12:00:01Z"},
	}}
	raw, err := protojson.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "route-evidence.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	flags := map[string]string{"input": path, "scenario": "source-ledger", "limit": "100"}
	stdout := &bytes.Buffer{}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "input"}, {Name: "scenario"}, {Name: "limit"}}},
		Flags:  flags, JSON: true, Stdout: stdout,
	})
	if err := (&handlers{}).evidenceProfile(ctx); err != nil {
		t.Fatalf("offline route evidence profile: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "1 retained route events") || !strings.Contains(output, "caller=scenario:source-ledger/api") {
		t.Fatalf("profile did not use the filtered retained receipt: %s", output)
	}
	if !strings.Contains(output, "provider_runtime[p50=17 p95=17 n=1 missing=0]") || !strings.Contains(output, "admission_wait[p50=3 p95=3 n=1 missing=0]") {
		t.Fatalf("profile dropped observed provider timings: %s", output)
	}
}

func TestRouteEvidenceProfileReceiptBoundsAndPreservesMissingTiming(t *testing.T) {
	path := filepath.Join(t.TempDir(), "route-evidence.json")
	oversized := bytes.Repeat([]byte(" "), maxRouteEvidenceReceiptBytes+1)
	if err := os.WriteFile(path, oversized, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := routeEvidenceProfileEvents(nil, path, "", 100); err == nil || !strings.Contains(err.Error(), "byte limit") {
		t.Fatalf("oversized receipt error = %v", err)
	}

	receipt := &routingv1.ListRouteEvidenceResponse{Events: make([]*routingv1.RouteEvidence, 101)}
	for i := range receipt.Events {
		receipt.Events[i] = &routingv1.RouteEvidence{TaskRole: "chat.default"}
	}
	raw, err := protojson.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := routeEvidenceProfileEvents(nil, path, "", 100); err == nil || !strings.Contains(err.Error(), "maximum is 100") {
		t.Fatalf("over-limit receipt error = %v", err)
	}

	receipt.Events = receipt.Events[:1]
	raw, err = protojson.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	events, _, err := routeEvidenceProfileEvents(nil, path, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	rows := profileEvidence(events)
	if len(rows) != 1 || !strings.Contains(rows[0], "provider_runtime=unavailable(1/1)") || !strings.Contains(rows[0], "admission_wait=unavailable(1/1)") {
		t.Fatalf("unobserved provider timings should stay unavailable: %v", rows)
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
