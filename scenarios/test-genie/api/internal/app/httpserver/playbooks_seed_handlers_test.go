package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/workspace"
)

func TestServer_handlePlaybooksSeedApply_Success(t *testing.T) {
	scenario := "demo"
	scenariosRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(scenariosRoot, scenario), 0o755); err != nil {
		t.Fatalf("create scenario dir: %v", err)
	}

	var gotScenario string
	apply := func(ctx context.Context, env workspace.Environment, logWriter io.Writer, retain bool) (*phases.PlaybooksSeedSession, error) {
		gotScenario = env.ScenarioName
		return &phases.PlaybooksSeedSession{
			RunID:     "run-123",
			SeedState: map[string]any{"seeded": true},
		}, nil
	}

	previousApply := applyPlaybooksSeed
	applyPlaybooksSeed = apply
	t.Cleanup(func() { applyPlaybooksSeed = previousApply })

	server := &Server{
		config:                 Config{Port: "0"},
		router:                 mux.NewRouter(),
		scenarios:              &stubScenarioDirectory{scenarioRoot: scenariosRoot},
		logger:                 log.New(io.Discard, "", 0),
		seedSessions:           make(map[string]*seedSession),
		seedSessionsByScenario: make(map[string]string),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/"+scenario+"/playbooks/seed/apply", strings.NewReader(`{}`))
	req = mux.SetURLVars(req, map[string]string{"name": scenario})
	rec := httptest.NewRecorder()

	server.handlePlaybooksSeedApply(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if gotScenario != scenario {
		t.Fatalf("expected apply to receive scenario %q, got %q", scenario, gotScenario)
	}

	var payload map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "seeded" {
		t.Fatalf("expected status seeded, got %#v", payload["status"])
	}
	if payload["cleanup_token"] == "" {
		t.Fatalf("expected cleanup_token in response")
	}
}

func TestServer_handlePlaybooksSeedApply_Conflict(t *testing.T) {
	server := &Server{
		config:                 Config{Port: "0"},
		router:                 mux.NewRouter(),
		scenarios:              &stubScenarioDirectory{scenarioRoot: "/tmp"},
		logger:                 log.New(io.Discard, "", 0),
		seedSessions:           make(map[string]*seedSession),
		seedSessionsByScenario: map[string]string{"demo": "existing-token"},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/demo/playbooks/seed/apply", strings.NewReader(`{}`))
	req = mux.SetURLVars(req, map[string]string{"name": "demo"})
	rec := httptest.NewRecorder()

	server.handlePlaybooksSeedApply(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestServer_handlePlaybooksSeedCleanup_Success(t *testing.T) {
	token := "cleanup-token"
	server := &Server{
		config:    Config{Port: "0"},
		router:    mux.NewRouter(),
		scenarios: &stubScenarioDirectory{scenarioRoot: "/tmp"},
		logger:    log.New(io.Discard, "", 0),
		seedSessions: map[string]*seedSession{
			token: {
				Scenario: "demo",
				Session: &phases.PlaybooksSeedSession{
					RunID: "run-123",
				},
			},
		},
		seedSessionsByScenario: map[string]string{"demo": token},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/demo/playbooks/seed/cleanup", strings.NewReader(`{"cleanup_token":"cleanup-token"}`))
	req = mux.SetURLVars(req, map[string]string{"name": "demo"})
	rec := httptest.NewRecorder()

	server.handlePlaybooksSeedCleanup(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if len(server.seedSessions) != 0 || len(server.seedSessionsByScenario) != 0 {
		t.Fatalf("expected cleanup session to be removed")
	}
}

func TestServer_handlePlaybooksSeedCleanup_Conflict(t *testing.T) {
	token := "cleanup-token"
	server := &Server{
		config:    Config{Port: "0"},
		router:    mux.NewRouter(),
		scenarios: &stubScenarioDirectory{scenarioRoot: "/tmp"},
		logger:    log.New(io.Discard, "", 0),
		seedSessions: map[string]*seedSession{
			token: {
				Scenario: "other",
				Session:  &phases.PlaybooksSeedSession{RunID: "run-123"},
			},
		},
		seedSessionsByScenario: map[string]string{"other": token},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/demo/playbooks/seed/cleanup", strings.NewReader(`{"cleanup_token":"cleanup-token"}`))
	req = mux.SetURLVars(req, map[string]string{"name": "demo"})
	rec := httptest.NewRecorder()

	server.handlePlaybooksSeedCleanup(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}
