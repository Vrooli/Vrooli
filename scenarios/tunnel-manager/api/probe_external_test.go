package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:PROBE-002] External probe per route
func TestProbeExternalUp(t *testing.T) {
	// Simulate a public URL endpoint
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer ts.Close()

	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	port := extractPort(t, ts.URL)

	// Create route with public_url pointing to our test server
	_, err := svc.Create(RouteInput{
		Subdomain:    "ext-test",
		ScenarioName: "some-scenario",
		LocalPort:    port,
		PublicURL:    ts.URL,
	})
	if err != nil {
		t.Fatalf("create route: %v", err)
	}

	probeSvc := NewProbeService(db, svc)
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

	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	port := extractPort(t, ts.URL)

	// Create route without public_url
	_, err := svc.Create(RouteInput{
		Subdomain:    "no-public",
		ScenarioName: "some-scenario",
		LocalPort:    port,
	})
	if err != nil {
		t.Fatalf("create route: %v", err)
	}

	probeSvc := NewProbeService(db, svc)
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

	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	port := extractPort(t, ts.URL)

	_, err := svc.Create(RouteInput{
		Subdomain:    "ext-down",
		ScenarioName: "failing-scenario",
		LocalPort:    port,
		PublicURL:    ts.URL,
	})
	if err != nil {
		t.Fatalf("create route: %v", err)
	}

	probeSvc := NewProbeService(db, svc)
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
