package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCmdInitiativesContext_HumanOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/initiatives/main/context" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		resp := InitiativeContextResponse{
			Initiative: Initiative{
				Name: "main", Title: "Main", Status: "active", Priority: 3,
				DependsOn: []string{"upstream-a"},
				Items:     []string{"idea/one"},
			},
			Rollup: InitiativeRollup{Total: 1, Pending: 1},
			Items: []InitiativeContextItem{
				{Kind: "idea", Name: "one", Title: "One", Status: "backlog", Priority: 3},
			},
			UpstreamInitiatives:   []Initiative{{Name: "upstream-a", Title: "Upstream A", Status: "active"}},
			DownstreamInitiatives: []Initiative{{Name: "downstream-b", Title: "Downstream B", Status: "active"}},
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

	if err := app.cmdInitiativesContext([]string{"--name", "main"}); err != nil {
		t.Fatalf("cmdInitiativesContext: %v", err)
	}
}

func TestCmdInitiativesContext_JSONOutput(t *testing.T) {
	payload := `{"initiative":{"name":"main","title":"Main","status":"active","items":[]},"rollup":{"total":0},"items":[],"upstream_initiatives":[],"downstream_initiatives":[]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer server.Close()

	t.Setenv("SWARM_MANAGER_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	if err := app.cmdInitiativesContext([]string{"--name", "main", "--json"}); err != nil {
		t.Fatalf("cmdInitiativesContext --json: %v", err)
	}
}

func TestCmdInitiativesContext_RequiresName(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	err = app.cmdInitiativesContext([]string{})
	if err == nil || !strings.Contains(err.Error(), "--name") {
		t.Fatalf("expected --name required error, got %v", err)
	}
}

func TestCmdInitiativesContext_ScenarioMode_HitsScenarioEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/scenarios/web-console/context" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		resp := ScenarioContextResponse{
			ScenarioName: "web-console",
			Initiatives: []InitiativeResponse{
				{
					Initiative:      Initiative{Name: "audio-platform", Title: "Audio", Status: "active"},
					Rollup:          InitiativeRollup{Total: 3, Completed: 1, Pending: 2},
					TargetScenarios: []string{"web-console"},
				},
			},
			OrphanItems: []ScenarioContextOrphanItem{
				{Kind: "execute", Name: "orphan-one", Title: "Orphan 1", Status: "backlog", Priority: 3},
			},
			Rollup: ScenarioContextRollup{Total: 4, Completed: 1, Pending: 3},
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
	if err := app.cmdInitiativesContext([]string{"--scenario", "web-console"}); err != nil {
		t.Fatalf("cmdInitiativesContext --scenario: %v", err)
	}
}

func TestCmdInitiativesContext_NameAndScenarioMutuallyExclusive(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	err = app.cmdInitiativesContext([]string{"--name", "foo", "--scenario", "bar"})
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutually-exclusive error, got %v", err)
	}
}

func TestCmdInitiativesContext_ScenarioMode_JSON(t *testing.T) {
	payload := `{"scenario_name":"x","initiatives":[],"orphan_items":[],"rollup":{"total":0}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/scenarios/x/context" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer server.Close()

	t.Setenv("SWARM_MANAGER_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	if err := app.cmdInitiativesContext([]string{"--scenario", "x", "--json"}); err != nil {
		t.Fatalf("cmdInitiativesContext --scenario --json: %v", err)
	}
}
