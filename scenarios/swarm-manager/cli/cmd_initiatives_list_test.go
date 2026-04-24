package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCmdInitiativesList_PassesScenarioFlag(t *testing.T) {
	var seenQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/initiatives" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		seenQuery = r.URL.RawQuery
		resp := ListInitiativesResponse{Items: []InitiativeResponse{}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	t.Setenv("SWARM_MANAGER_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	if err := app.cmdInitiativesList([]string{"--scenario", "web-console"}); err != nil {
		t.Fatalf("cmdInitiativesList: %v", err)
	}
	if seenQuery != "scenario=web-console" {
		t.Errorf("expected query 'scenario=web-console', got %q", seenQuery)
	}
}

func TestCmdInitiativesList_PassesScenarioFlagCSV(t *testing.T) {
	var seenQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenQuery = r.URL.RawQuery
		resp := ListInitiativesResponse{Items: []InitiativeResponse{}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	t.Setenv("SWARM_MANAGER_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	if err := app.cmdInitiativesList([]string{"--scenario", "web-console,command-center"}); err != nil {
		t.Fatalf("cmdInitiativesList: %v", err)
	}
	// url.Values percent-encodes comma as %2C. Accept either raw or encoded
	// depending on how net/url renders the query string.
	want1 := "scenario=web-console%2Ccommand-center"
	want2 := "scenario=web-console,command-center"
	if seenQuery != want1 && seenQuery != want2 {
		t.Errorf("expected scenario CSV query, got %q", seenQuery)
	}
}

func TestCmdInitiativesList_OmitsScenarioWhenAbsent(t *testing.T) {
	var seenQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenQuery = r.URL.RawQuery
		resp := ListInitiativesResponse{Items: []InitiativeResponse{}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	t.Setenv("SWARM_MANAGER_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	if err := app.cmdInitiativesList([]string{}); err != nil {
		t.Fatalf("cmdInitiativesList: %v", err)
	}
	if seenQuery != "" {
		t.Errorf("expected empty query string, got %q", seenQuery)
	}
}

func TestCmdInitiativesList_RendersTargetScenarios(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ListInitiativesResponse{
			Items: []InitiativeResponse{
				{
					Initiative: Initiative{
						Name: "audio-platform", Title: "Audio Platform", Status: "active",
						Items: []string{"execute/web-console-audio"},
					},
					Rollup:          InitiativeRollup{Total: 1, Pending: 1},
					TargetScenarios: []string{"web-console"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	t.Setenv("SWARM_MANAGER_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	if err := app.cmdInitiativesList([]string{}); err != nil {
		t.Fatalf("cmdInitiativesList: %v", err)
	}
}
