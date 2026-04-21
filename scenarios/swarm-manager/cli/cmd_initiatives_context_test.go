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
