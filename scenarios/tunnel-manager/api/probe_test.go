package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// [REQ:PROBE-001] Internal probe per route
func TestProbeInternal(t *testing.T) {
	// Start a local test server to simulate a scenario
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer ts.Close()

	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	// Extract port from test server URL
	port := extractPort(t, ts.URL)
	seedTestRoute(t, db, "internal-test", "some-scenario", port)

	probeSvc := NewProbeService(db, svc)
	results, err := probeSvc.RunAll(context.Background())
	if err != nil {
		t.Fatalf("runAll: %v", err)
	}

	// Find internal probe result
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

	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	port := extractPort(t, ts.URL)

	// Create multiple routes
	for i := 0; i < 5; i++ {
		seedTestRoute(t, db, itoa(i)+"-concurrent", "scenario-"+itoa(i), port)
	}

	probeSvc := NewProbeService(db, svc)
	start := time.Now()
	results, err := probeSvc.RunAll(context.Background())
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("runAll: %v", err)
	}

	// 5 routes × 2 probe types = 10 results
	if len(results) != 10 {
		t.Errorf("expected 10 results, got %d", len(results))
	}

	// With concurrent execution, should be faster than sequential (5 × 50ms × 2 = 500ms)
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

	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	port := extractPort(t, ts.URL)

	// Create route with custom health path
	route, err := svc.Create(RouteInput{
		Subdomain:    "custom-health",
		ScenarioName: "custom-scenario",
		LocalPort:    port,
		HealthPath:   "/custom/health",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	probeSvc := NewProbeService(db, svc)
	result := probeSvc.probeInternal(context.Background(), *route)
	if result.Status != "up" {
		t.Errorf("status = %q, want up", result.Status)
	}
}

func extractPort(t *testing.T, url string) int {
	t.Helper()
	// url is like http://127.0.0.1:12345
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
