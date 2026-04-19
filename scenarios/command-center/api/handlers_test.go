package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testRegistry() *Registry {
	return &Registry{
		Version: "1.0.0",
		Dashboards: map[string][]MetricEntry{
			"mission-control": {
				{ID: "live_swarm", Label: "swarm throughput", DataSource: StatusLive, UpstreamSource: SourceSwarm},
				{ID: "gap_revenue", Label: "revenue", DataSource: StatusGap, UpstreamSource: SourceLPBS},
			},
			"hive": {
				{ID: "live_scenarios", Label: "active scenarios", DataSource: StatusLive, UpstreamSource: SourceVrooli},
			},
		},
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
