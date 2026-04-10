package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"tunnel-manager/domain"
)

func extractPort(t *testing.T, url string) int {
	t.Helper()
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == ':' {
			port := 0
			for _, c := range url[i+1:] {
				port = port*10 + int(c-'0')
			}
			return port
		}
	}
	t.Fatal("could not extract port from URL")
	return 0
}

// [REQ:PROBE-001] Internal probe per route
func TestProbeInternal(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer ts.Close()

	port := extractPort(t, ts.URL)
	writer := &mockProbeResultWriter{
		persistResultFn: func(pr domain.ProbeResult) error { return nil },
	}
	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{ID: 1, Subdomain: "internal-test", ScenarioName: "some-scenario", LocalPort: port, HealthPath: "/health", Enabled: true, PublicURL: ts.URL},
			}, nil
		},
	}

	probeSvc := NewProbeService(lister, writer)
	results, err := probeSvc.RunAll(context.Background())
	if err != nil {
		t.Fatalf("runAll: %v", err)
	}

	var found bool
	for _, r := range results {
		if r.ProbeType == "internal" && r.Subdomain == "internal-test" {
			found = true
			if r.Status != "up" {
				t.Errorf("internal probe status = %q, want up", r.Status)
			}
			if r.LatencyMs < 0 {
				t.Error("expected non-negative latency")
			}
		}
	}
	if !found {
		t.Error("internal probe result not found")
	}
}

// [REQ:PROBE-004] Concurrent probe execution
func TestProbeConcurrentExecution(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	port := extractPort(t, ts.URL)

	routes := make([]domain.Route, 5)
	for i := 0; i < 5; i++ {
		routes[i] = domain.Route{
			ID: i + 1, Subdomain: "concurrent-" + itoa(i), ScenarioName: "scenario-" + itoa(i),
			LocalPort: port, HealthPath: "/health", Enabled: true, PublicURL: ts.URL,
		}
	}

	writer := &mockProbeResultWriter{persistResultFn: func(pr domain.ProbeResult) error { return nil }}
	lister := &mockRouteLister{listFn: func() ([]domain.Route, error) { return routes, nil }}

	probeSvc := NewProbeService(lister, writer)
	start := time.Now()
	results, err := probeSvc.RunAll(context.Background())
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("runAll: %v", err)
	}

	if len(results) != 10 {
		t.Errorf("expected 10 results, got %d", len(results))
	}

	if elapsed > 400*time.Millisecond {
		t.Errorf("probes took %v, expected concurrent execution to be faster", elapsed)
	}
}

// [REQ:PROBE-006] Custom health endpoint support
func TestProbeCustomHealthPath(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/custom/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	port := extractPort(t, ts.URL)
	writer := &mockProbeResultWriter{persistResultFn: func(pr domain.ProbeResult) error { return nil }}
	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{ID: 1, Subdomain: "custom-health", ScenarioName: "custom-scenario", LocalPort: port, HealthPath: "/custom/health", Enabled: true, PublicURL: ts.URL},
			}, nil
		},
	}

	probeSvc := NewProbeService(lister, writer)
	results, err := probeSvc.RunAll(context.Background())
	if err != nil {
		t.Fatalf("runAll: %v", err)
	}

	var found bool
	for _, r := range results {
		if r.ProbeType == "internal" && r.Subdomain == "custom-health" {
			found = true
			if r.Status != "up" {
				t.Errorf("status = %q, want up", r.Status)
			}
		}
	}
	if !found {
		t.Error("internal probe result for custom-health not found")
	}
}

// [REQ:PROBE-002] External probe per route
func TestProbeExternalUp(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer ts.Close()

	port := extractPort(t, ts.URL)
	writer := &mockProbeResultWriter{persistResultFn: func(pr domain.ProbeResult) error { return nil }}
	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{ID: 1, Subdomain: "ext-test", ScenarioName: "some-scenario", LocalPort: port, HealthPath: "/health", Enabled: true, PublicURL: ts.URL},
			}, nil
		},
	}

	probeSvc := NewProbeService(lister, writer)
	results, err := probeSvc.RunAll(context.Background())
	if err != nil {
		t.Fatalf("runAll: %v", err)
	}

	var found bool
	for _, r := range results {
		if r.ProbeType == "external" && r.Subdomain == "ext-test" {
			found = true
			if r.Status != "up" {
				t.Errorf("external probe status = %q, want up", r.Status)
			}
			if r.StatusCode != http.StatusOK {
				t.Errorf("external probe status_code = %d, want 200", r.StatusCode)
			}
		}
	}
	if !found {
		t.Error("external probe result not found")
	}
}

// [REQ:PROBE-002] External probe returns error when no public_url is set
func TestProbeExternalNoPublicURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	port := extractPort(t, ts.URL)
	writer := &mockProbeResultWriter{persistResultFn: func(pr domain.ProbeResult) error { return nil }}
	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{ID: 1, Subdomain: "no-public", ScenarioName: "some-scenario", LocalPort: port, HealthPath: "/health", Enabled: true},
			}, nil
		},
	}

	probeSvc := NewProbeService(lister, writer)
	results, err := probeSvc.RunAll(context.Background())
	if err != nil {
		t.Fatalf("runAll: %v", err)
	}

	for _, r := range results {
		if r.ProbeType == "external" && r.Subdomain == "no-public" {
			if r.Status != "error" {
				t.Errorf("external probe without public_url: status = %q, want error", r.Status)
			}
			if r.ErrorMsg == "" {
				t.Error("expected error message for missing public_url")
			}
			return
		}
	}
	t.Error("external probe result not found")
}

// [REQ:PROBE-002] External probe reports down when endpoint returns 500
func TestProbeExternalDown(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	port := extractPort(t, ts.URL)
	writer := &mockProbeResultWriter{persistResultFn: func(pr domain.ProbeResult) error { return nil }}
	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{ID: 1, Subdomain: "ext-down", ScenarioName: "failing-scenario", LocalPort: port, HealthPath: "/health", Enabled: true, PublicURL: ts.URL},
			}, nil
		},
	}

	probeSvc := NewProbeService(lister, writer)
	results, err := probeSvc.RunAll(context.Background())
	if err != nil {
		t.Fatalf("runAll: %v", err)
	}

	for _, r := range results {
		if r.ProbeType == "external" && r.Subdomain == "ext-down" {
			if r.Status != "down" {
				t.Errorf("external probe status = %q, want down", r.Status)
			}
			return
		}
	}
	t.Error("external probe result not found")
}

// [REQ:PROBE-003] Probe result classification - up
func TestClassifyRouteUp(t *testing.T) {
	results := []domain.ProbeResult{
		{RouteID: 1, Subdomain: "app", ProbeType: "internal", Status: "up"},
		{RouteID: 1, Subdomain: "app", ProbeType: "external", Status: "up"},
	}
	classifications := ClassifyProbeResults(results)
	if len(classifications) != 1 {
		t.Fatalf("expected 1 classification, got %d", len(classifications))
	}
	if classifications[0].Status != "up" {
		t.Errorf("status = %q, want up", classifications[0].Status)
	}
	if classifications[0].Assessment == "" {
		t.Error("expected non-empty assessment")
	}
}

// [REQ:PROBE-003] Probe result classification - tunnel-issue
func TestClassifyTunnelIssue(t *testing.T) {
	results := []domain.ProbeResult{
		{RouteID: 1, Subdomain: "app", ProbeType: "internal", Status: "up"},
		{RouteID: 1, Subdomain: "app", ProbeType: "external", Status: "down"},
	}
	classifications := ClassifyProbeResults(results)
	if len(classifications) != 1 {
		t.Fatalf("expected 1 classification, got %d", len(classifications))
	}
	if classifications[0].Status != "tunnel-issue" {
		t.Errorf("status = %q, want tunnel-issue", classifications[0].Status)
	}
}

// [REQ:PROBE-003] Probe result classification - scenario-down
func TestClassifyScenarioDown(t *testing.T) {
	results := []domain.ProbeResult{
		{RouteID: 1, Subdomain: "app", ProbeType: "internal", Status: "down"},
		{RouteID: 1, Subdomain: "app", ProbeType: "external", Status: "up"},
	}
	classifications := ClassifyProbeResults(results)
	if len(classifications) != 1 {
		t.Fatalf("expected 1 classification, got %d", len(classifications))
	}
	if classifications[0].Status != "scenario-down" {
		t.Errorf("status = %q, want scenario-down", classifications[0].Status)
	}
}

// [REQ:PROBE-003] Probe result classification - unknown
func TestClassifyUnknown(t *testing.T) {
	results := []domain.ProbeResult{
		{RouteID: 1, Subdomain: "app", ProbeType: "internal", Status: "down"},
		{RouteID: 1, Subdomain: "app", ProbeType: "external", Status: "down"},
	}
	classifications := ClassifyProbeResults(results)
	if len(classifications) != 1 {
		t.Fatalf("expected 1 classification, got %d", len(classifications))
	}
	if classifications[0].Status != "unknown" {
		t.Errorf("status = %q, want unknown", classifications[0].Status)
	}
}

// [REQ:PROBE-003] Classification handles multiple routes
func TestClassifyMultipleRoutes(t *testing.T) {
	results := []domain.ProbeResult{
		{RouteID: 1, Subdomain: "app1", ProbeType: "internal", Status: "up"},
		{RouteID: 1, Subdomain: "app1", ProbeType: "external", Status: "up"},
		{RouteID: 2, Subdomain: "app2", ProbeType: "internal", Status: "up"},
		{RouteID: 2, Subdomain: "app2", ProbeType: "external", Status: "down"},
	}
	classifications := ClassifyProbeResults(results)
	if len(classifications) != 2 {
		t.Fatalf("expected 2 classifications, got %d", len(classifications))
	}

	statusByRoute := make(map[string]string)
	for _, c := range classifications {
		statusByRoute[c.Subdomain] = c.Status
	}
	if statusByRoute["app1"] != "up" {
		t.Errorf("app1 status = %q, want up", statusByRoute["app1"])
	}
	if statusByRoute["app2"] != "tunnel-issue" {
		t.Errorf("app2 status = %q, want tunnel-issue", statusByRoute["app2"])
	}
}

// [REQ:PROBE-001] Probe writer is called for persisted results
func TestProbePersistsResults(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	port := extractPort(t, ts.URL)

	var mu sync.Mutex
	var persisted []domain.ProbeResult
	writer := &mockProbeResultWriter{
		persistResultFn: func(pr domain.ProbeResult) error {
			mu.Lock()
			persisted = append(persisted, pr)
			mu.Unlock()
			return nil
		},
	}
	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{ID: 1, Subdomain: "persist-test", ScenarioName: "test", LocalPort: port, HealthPath: "/health", Enabled: true, PublicURL: ts.URL},
			}, nil
		},
	}

	probeSvc := NewProbeService(lister, writer)
	_, err := probeSvc.RunAll(context.Background())
	if err != nil {
		t.Fatalf("runAll: %v", err)
	}

	mu.Lock()
	count := len(persisted)
	mu.Unlock()
	// internal + external = 2
	if count != 2 {
		t.Errorf("expected 2 persisted results, got %d", count)
	}
}

func itoa(i int) string {
	return string(rune('0' + i))
}
