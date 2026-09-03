package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestObservationTimeRequiresProducerMetadata(t *testing.T) {
	if got := observationTime([]byte(`{"value":1}`)); got != nil {
		t.Fatalf("observationTime without metadata = %v, want nil", got)
	}
	got := observationTime([]byte(`{"observed_at":"2026-09-02T12:00:00Z"}`))
	if got == nil || got.UTC().Format(time.RFC3339) != "2026-09-02T12:00:00Z" {
		t.Fatalf("observationTime = %v", got)
	}
	if got := observationTime([]byte(`{"generated_at":"2026-09-02T12:00:00Z"}`)); got == nil {
		t.Fatal("generated_at is a producer observation timestamp and must be accepted")
	}
}

func testRegistry() *Registry {
	return &Registry{
		Version: "1.0.0",
		Dashboards: map[string][]MetricEntry{
			"mission-control": {
				{ID: "live_swarm", Label: "swarm throughput", DataSource: StatusLive, UpstreamSource: SourceSwarm, Source: SourceBinding{Binding: "scenario:swarm-manager", Read: "/api/v1/stats", IntegrationID: "swarm-manager", FeatureID: "swarm_throughput", Selector: "swarm_throughput", ContractVersion: "portfolio.v1", ExpectedUnit: "count", SourceTimePolicy: "producer_required", TTLSeconds: 60}},
				{ID: "gap_revenue", Label: "revenue", DataSource: StatusGap, UpstreamSource: SourceLPBS},
			},
			"hive": {
				{ID: "live_scenarios", Label: "active scenarios", DataSource: StatusLive, UpstreamSource: SourceVrooli},
			},
		},
	}
}

type staticUpstreamClient struct {
	name string
	body json.RawMessage
}

func (c staticUpstreamClient) Name() string { return c.name }

func (c staticUpstreamClient) Fetch(context.Context, string) (json.RawMessage, error) {
	return append(json.RawMessage(nil), c.body...), nil
}

func TestHandleDashboard_QualifiesTypedSwarmEnvelope(t *testing.T) {
	s := NewServer(testRegistry())
	s.swarm = staticUpstreamClient{name: "swarm", body: json.RawMessage(fmt.Sprintf(`{"observed_at":%q,"throughput":{"completed_last_7_days":11}}`, time.Now().UTC().Add(-5*time.Second).Format(time.RFC3339)))}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboards/mission-control", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body dashboardResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Metrics) != 2 {
		t.Fatalf("metrics=%d, want 2", len(body.Metrics))
	}
	metric := body.Metrics[0]
	if metric.Trust != TrustValid || metric.Value != float64(11) || metric.ObservedAt == nil {
		t.Fatalf("typed swarm reading=%+v", metric)
	}
}

func TestHandleDashboardRejectsProducerContractAndUnitMismatch(t *testing.T) {
	tests := []struct {
		name string
		meta string
		want string
	}{
		{name: "contract version", meta: `"contractVersion":"wrong.v1","unit":"count"`, want: "producer contract version wrong.v1"},
		{name: "unit", meta: `"contractVersion":"portfolio.v1","unit":"usd"`, want: "producer unit usd"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := NewServer(testRegistry())
			s.swarm = staticUpstreamClient{name: "swarm", body: json.RawMessage(fmt.Sprintf(`{"observed_at":%q,%s,"throughput":{"completed_last_7_days":11}}`, time.Now().UTC().Add(-5*time.Second).Format(time.RFC3339), tc.meta))}
			req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboards/mission-control", nil)
			rr := httptest.NewRecorder()
			s.router.ServeHTTP(rr, req)
			var body dashboardResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body.Metrics[0].Trust != TrustUntrusted || !strings.Contains(body.Metrics[0].TrustReason, tc.want) {
				t.Fatalf("metric=%+v, want UNTRUSTED reason containing %q", body.Metrics[0], tc.want)
			}
		})
	}
}

func TestHandleDashboardRejectsMalformedAndImplausibleObservationsAsUntrusted(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{name: "malformed json", payload: `{"`, want: "not JSON"},
		{name: "missing selector", payload: `{"observed_at":"2026-09-03T03:00:00Z","other":1}`, want: "found no number"},
		{name: "negative count", payload: `{"observed_at":"2026-09-03T03:00:00Z","throughput":{"completed_last_7_days":-1}}`, want: "implausible"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := NewServer(testRegistry())
			payload := strings.ReplaceAll(tc.payload, "2026-09-03T03:00:00Z", time.Now().UTC().Add(-5*time.Second).Format(time.RFC3339))
			s.swarm = staticUpstreamClient{name: "swarm", body: json.RawMessage(payload)}
			req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboards/mission-control", nil)
			rr := httptest.NewRecorder()
			s.router.ServeHTTP(rr, req)
			var body dashboardResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body.Metrics[0].Trust != TrustUntrusted || !strings.Contains(body.Metrics[0].TrustReason, tc.want) {
				t.Fatalf("metric=%+v, want UNTRUSTED reason containing %q", body.Metrics[0], tc.want)
			}
		})
	}
}

func TestHandleGaps_ReturnsOnlyGapAndPartial(t *testing.T) {
	s := NewServer(testRegistry())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gaps", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body gapsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := body.Dashboards["mission-control"]; !ok {
		t.Error("expected mission-control in gaps")
	}
	if _, ok := body.Dashboards["hive"]; ok {
		t.Error("hive has no gap/partial entries — should be omitted")
	}
}

func TestHandleDashboard_NotFound(t *testing.T) {
	s := NewServer(testRegistry())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboards/unknown", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "dashboard_not_found") {
		t.Errorf("error code missing: %s", rr.Body.String())
	}
}

func TestHandleDashboard_OK(t *testing.T) {
	s := NewServer(testRegistry())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboards/hive", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body dashboardResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Dashboard != "hive" {
		t.Errorf("dashboard=%q", body.Dashboard)
	}
	if len(body.Metrics) != 1 {
		t.Errorf("expected 1 metric, got %d", len(body.Metrics))
	}
}

func TestHandleR3FStats_PostThenGet(t *testing.T) {
	s := NewServer(testRegistry())

	post := httptest.NewRequest(http.MethodPost, "/api/v1/debug/r3f-stats",
		strings.NewReader(`{"tier":2,"fps":58.3,"route":"/mission-control"}`))
	post.Header.Set("Content-Type", "application/json")
	pw := httptest.NewRecorder()
	s.router.ServeHTTP(pw, post)
	if pw.Code != http.StatusAccepted {
		t.Fatalf("post status=%d body=%s", pw.Code, pw.Body.String())
	}

	get := httptest.NewRequest(http.MethodGet, "/api/v1/debug/r3f-stats", nil)
	gw := httptest.NewRecorder()
	s.router.ServeHTTP(gw, get)
	if gw.Code != http.StatusOK {
		t.Fatalf("get status=%d", gw.Code)
	}
	var out struct {
		Events []R3FEvent `json:"events"`
	}
	if err := json.Unmarshal(gw.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out.Events) != 1 || out.Events[0].Tier != 2 {
		t.Errorf("unexpected events: %+v", out.Events)
	}
}

func TestHandleR3FStats_InvalidBody(t *testing.T) {
	s := NewServer(testRegistry())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/debug/r3f-stats",
		strings.NewReader(`not json`))
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHealth_Endpoint(t *testing.T) {
	s := NewServer(testRegistry())
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
