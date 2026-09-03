package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	cliv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1/cliv1connect"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	swarmstatsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/stats"
	swarmstatsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/stats/stats_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSwarm_OKResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/stats" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"throughput":42}`))
	}))
	defer srv.Close()

	c := NewSwarm(srv.URL)
	raw, err := c.Fetch(context.Background(), "/api/v1/stats")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if string(raw) != `{"throughput":42}` {
		t.Errorf("unexpected body: %s", raw)
	}
	if c.Name() != "swarm" {
		t.Errorf("name=%q", c.Name())
	}
}

func TestVrooli_EmptyBaseURLIsNotAvailable(t *testing.T) {
	c := NewVrooli("")
	_, err := c.Fetch(context.Background(), "/scenarios")
	if !errors.Is(err, ErrNotAvailable) {
		t.Errorf("expected ErrNotAvailable, got %v", err)
	}
}

type typedVrooliTestService struct {
	cliv1connect.UnimplementedScenarioControlPlaneServiceHandler
}

func (typedVrooliTestService) ListScenarios(context.Context, *connect.Request[cliv1.ListScenariosRequest]) (*connect.Response[cliv1.ScenarioListResponse], error) {
	return connect.NewResponse(&cliv1.ScenarioListResponse{
		ObservedAt: timestamppb.New(time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)),
		Scenarios: []*cliv1.Scenario{
			{Name: "alpha", Status: "running", HealthStatus: "healthy", Ports: []*cliv1.ScenarioPort{{Key: "API_PORT", Port: 1234}}},
			{Name: "beta", Status: "available", HealthStatus: "unhealthy"},
		},
	}), nil
}

func TestVrooliTypedClientUsesGeneratedReadContract(t *testing.T) {
	_, handler := cliv1connect.NewScenarioControlPlaneServiceHandler(typedVrooliTestService{})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewVrooliTypedResolved(func() string { return srv.URL })
	raw, err := client.Fetch(context.Background(), "/scenarios")
	if err != nil {
		t.Fatalf("typed fetch: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("normalized payload: %v", err)
	}
	if payload["observed_at"] != "2026-09-03T12:00:00Z" || payload["contract_version"] != "legacy.v1" {
		t.Fatalf("normalized envelope = %s", raw)
	}
	rows, ok := payload["data"].([]any)
	if !ok || len(rows) != 2 {
		t.Fatalf("data rows = %#v", payload["data"])
	}
	row, ok := rows[0].(map[string]any)
	if !ok || row["health_status"] != "healthy" {
		t.Fatalf("first row = %#v", rows[0])
	}
	ports, ok := row["ports"].(map[string]any)
	if !ok || ports["API_PORT"] != float64(1234) {
		t.Fatalf("first row ports = %#v", row["ports"])
	}
}

func TestLPBS_404FallsThroughToGapMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewLPBS(srv.URL, "sekret")
	_, err := c.Fetch(context.Background(), "/api/v1/admin/dashboard/summary")
	if !errors.Is(err, ErrNotAvailable) {
		t.Errorf("expected ErrNotAvailable on 404, got %v", err)
	}
}

func TestLPBS_SendsBearerToken(t *testing.T) {
	received := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewLPBS(srv.URL, "sekret-token")
	if _, err := c.Fetch(context.Background(), "/any"); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if received != "Bearer sekret-token" {
		t.Errorf("missing bearer, got %q", received)
	}
}

func TestLPBS_NoTokenSkipsHeader(t *testing.T) {
	received := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewLPBS(srv.URL, "")
	if _, err := c.Fetch(context.Background(), "/any"); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if received != "" {
		t.Errorf("expected no Authorization header, got %q", received)
	}
}

func TestResolvedClientRetriesAfterTransportFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	badURL := "http://127.0.0.1:1"
	resolutions := 0
	client := newResolved("retry", func() string {
		resolutions++
		if resolutions == 1 {
			return badURL
		}
		return srv.URL
	})
	body, err := client.Fetch(context.Background(), "/health")
	if err != nil {
		t.Fatalf("fetch after re-resolution: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("body = %s", body)
	}
	if resolutions != 2 {
		t.Fatalf("resolver calls = %d, want one retry with re-resolution", resolutions)
	}
}

type typedLPBSTestService struct {
	lpbsconnect.UnimplementedMetricsServiceHandler
	receivedAuth string
}

func (s *typedLPBSTestService) GetAnalyticsSummary(_ context.Context, req *connect.Request[lpbsv1.GetAnalyticsSummaryRequest]) (*connect.Response[lpbsv1.AnalyticsSummary], error) {
	s.receivedAuth = req.Header().Get("Authorization")
	return connect.NewResponse(&lpbsv1.AnalyticsSummary{
		TotalVisitors: 12,
		ObservedAt:    timestamppb.New(time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)),
		VariantStats:  []*lpbsv1.VariantStats{{CtaClicks: 4, Conversions: 2}},
	}), nil
}

func TestLPBSTypedClientUsesGeneratedReadContract(t *testing.T) {
	service := &typedLPBSTestService{}
	_, handler := lpbsconnect.NewMetricsServiceHandler(service)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewLPBSTypedResolved(func() string { return srv.URL }, "typed-secret")
	raw, err := client.Fetch(context.Background(), "/api/v1/admin/dashboard/summary")
	if err != nil {
		t.Fatalf("typed fetch: %v", err)
	}
	if service.receivedAuth != "Bearer typed-secret" {
		t.Fatalf("authorization = %q", service.receivedAuth)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("normalized payload: %v", err)
	}
	if payload["observed_at"] != "2026-09-03T12:00:00Z" || payload["cta_clicks"] != float64(4) || payload["conversions"] != float64(2) || payload["variant_ab"] != float64(1) || payload["visitors"] != float64(12) {
		t.Fatalf("normalized payload = %s", raw)
	}
	if payload["contract_version"] != "legacy.v1" {
		t.Fatalf("contract_version = %v", payload["contract_version"])
	}
}

type typedSwarmTestService struct {
	swarmstatsconnect.UnimplementedStatsServiceHandler
}

func (typedSwarmTestService) GetPortfolioStats(context.Context, *connect.Request[swarmstatsv1.GetPortfolioStatsRequest]) (*connect.Response[swarmstatsv1.PortfolioStats], error) {
	return connect.NewResponse(&swarmstatsv1.PortfolioStats{
		ObservedAt:          timestamppb.New(time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)),
		SwarmThroughput:     11,
		ThroughputStats:     12,
		SwarmActiveAgents:   13,
		AgentStats:          0.75,
		TimingStats:         14.5,
		BlockingStats:       15,
		DashboardStats:      16,
		CompositeThroughput: 17,
		ReviewStats:         18,
		ScopeStats:          2,
	}), nil
}

func TestSwarmTypedClientUsesGeneratedReadContract(t *testing.T) {
	_, handler := swarmstatsconnect.NewStatsServiceHandler(typedSwarmTestService{})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	raw, err := NewSwarmTypedResolved(func() string { return srv.URL }).Fetch(context.Background(), "/api/v1/stats")
	if err != nil {
		t.Fatalf("typed fetch: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("normalized payload: %v", err)
	}
	if payload["observed_at"] != "2026-09-03T12:00:00Z" {
		t.Fatalf("normalized payload = %s", raw)
	}
	if payload["contract_version"] != "legacy.v1" {
		t.Fatalf("contract_version = %v", payload["contract_version"])
	}
	units, ok := payload["units"].(map[string]any)
	if !ok || units["swarm_throughput"] != "count" || units["agent_stats"] != "percent" {
		t.Fatalf("units = %#v", payload["units"])
	}
	throughput, ok := payload["throughput"].(map[string]any)
	if !ok || throughput["completed_last_7_days"] != float64(11) || throughput["created_last_7_days"] != float64(12) {
		t.Fatalf("throughput envelope = %#v", payload["throughput"])
	}
	agent, ok := payload["agent"].(map[string]any)
	if !ok || agent["total_executions"] != float64(13) || agent["success_rate"] != 0.75 || agent["avg_execution_minutes"] != 14.5 {
		t.Fatalf("agent envelope = %#v", payload["agent"])
	}
	if payload["scope_stats"] != float64(2) {
		t.Fatalf("scope_stats = %#v", payload["scope_stats"])
	}
}

func TestSwarmTypedClientProvesCanonicalFeatureCompatibility(t *testing.T) {
	_, handler := swarmstatsconnect.NewStatsServiceHandler(typedSwarmTestService{})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	probe, ok := NewSwarmTypedResolved(func() string { return srv.URL }).(FeatureProbe)
	if !ok {
		t.Fatal("typed swarm client does not expose FeatureProbe")
	}
	features, reasons := probe.ProbeFeatures(context.Background())
	if features["swarm_throughput"] != "compatible" || reasons["swarm_throughput"] == "" {
		t.Fatalf("features=%v reasons=%v", features, reasons)
	}
}

func TestLPBSTypedClientProvesOnlyReturnedFeatureContracts(t *testing.T) {
	_, handler := lpbsconnect.NewMetricsServiceHandler(&typedLPBSTestService{})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	probe, ok := NewLPBSTypedResolved(func() string { return srv.URL }, "").(FeatureProbe)
	if !ok {
		t.Fatal("typed LPBS client does not expose FeatureProbe")
	}
	features, _ := probe.ProbeFeatures(context.Background())
	if features["visitors"] != "compatible" || features["conversions"] != "compatible" {
		t.Fatalf("features=%v", features)
	}
	if _, exists := features["revenue_mrr"]; exists {
		t.Fatalf("unsupported feature was promoted: %v", features)
	}
}

func TestTypedClientsRejectUndeclaredPaths(t *testing.T) {
	clients := []Client{
		NewSwarmTypedResolved(func() string { return "http://127.0.0.1:1" }),
		NewVrooliTypedResolved(func() string { return "http://127.0.0.1:1" }),
		NewLPBSTypedResolved(func() string { return "http://127.0.0.1:1" }, ""),
	}
	for _, client := range clients {
		t.Run(client.Name(), func(t *testing.T) {
			body, err := client.Fetch(context.Background(), "/private")
			if err == nil || body != nil {
				t.Fatalf("undeclared path returned body=%s err=%v", body, err)
			}
		})
	}
}

func TestSwarmTypedClientRejectsLegacyOnlyCanonicalRead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"generated_at":"2026-09-03T12:00:00Z","throughput":{"completed_last_7_days":4}}`))
	}))
	defer srv.Close()

	raw, err := NewSwarmTypedResolved(func() string { return srv.URL }).Fetch(context.Background(), "/api/v1/stats")
	if err == nil || raw != nil {
		t.Fatalf("legacy-only producer unexpectedly satisfied typed read: raw=%s err=%v", raw, err)
	}
}

func TestLPBSTypedClientRejectsLegacyOnlyCanonicalRead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"generated_at":"2026-09-03T12:00:00Z","visitors":7}`))
	}))
	defer srv.Close()

	raw, err := NewLPBSTypedResolved(func() string { return srv.URL }, "legacy-token").Fetch(context.Background(), "/api/v1/admin/dashboard/summary")
	if err == nil || raw != nil {
		t.Fatalf("legacy-only producer unexpectedly satisfied typed read: raw=%s err=%v", raw, err)
	}
}

func TestBaseClient_Returns5xxAsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`boom`))
	}))
	defer srv.Close()

	c := NewSwarm(srv.URL)
	_, err := c.Fetch(context.Background(), "/x")
	if err == nil {
		t.Fatal("expected error on 500")
	}
	if errors.Is(err, ErrNotAvailable) {
		t.Error("500 should be a normal error, not ErrNotAvailable")
	}
}
