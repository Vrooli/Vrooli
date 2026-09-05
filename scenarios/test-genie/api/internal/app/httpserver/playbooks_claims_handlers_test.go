package httpserver

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"test-genie/internal/playbooksclaims"
	playbooksclaimsmocks "test-genie/internal/playbooksclaims/mocks"

	"github.com/gorilla/mux"
)

func newClaimsTestServer(t *testing.T, repo playbooksclaims.Repository) *Server {
	t.Helper()
	svc := playbooksclaims.NewService(playbooksclaims.Config{Repo: repo})
	srv := &Server{
		config:                 Config{Port: "0"},
		router:                 mux.NewRouter(),
		logger:                 log.New(io.Discard, "", 0),
		playbooksClaims:        svc,
		seedSessions:           make(map[string]*seedSession),
		seedSessionsByScenario: make(map[string]string),
	}
	srv.setupRoutes()
	return srv
}

func TestHandleListPlaybooksClaims_Empty(t *testing.T) {
	srv := newClaimsTestServer(t, playbooksclaimsmocks.NewFakeRepository())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/playbooks/claims", nil)
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body struct {
		Claims []claimDTO `json:"claims"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Claims) != 0 {
		t.Fatalf("expected empty claims, got %d", len(body.Claims))
	}
}

func TestHandleListPlaybooksClaims_OneActive(t *testing.T) {
	repo := playbooksclaimsmocks.NewFakeRepository()
	svc := playbooksclaims.NewService(playbooksclaims.Config{Repo: repo})
	if _, err := svc.TryAcquire(t.Context(), playbooksclaims.AcquireInput{
		ScenarioName: "demo", RunID: "run-1", Mode: playbooksclaims.ModeRouted, StartedBy: "tester",
	}); err != nil {
		t.Fatalf("acquire: %v", err)
	}

	srv := &Server{
		config:          Config{Port: "0"},
		router:          mux.NewRouter(),
		logger:          log.New(io.Discard, "", 0),
		playbooksClaims: svc,
	}
	srv.setupRoutes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/playbooks/claims", nil)
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body struct {
		Claims []claimDTO `json:"claims"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Claims) != 1 {
		t.Fatalf("expected 1 claim, got %d", len(body.Claims))
	}
	got := body.Claims[0]
	if got.ScenarioName != "demo" || got.RunID != "run-1" || got.Mode != "routed" || !got.Alive {
		t.Fatalf("unexpected claim DTO: %+v", got)
	}
}

func TestHandleGetPlaybooksClaim_NotFoundReturnsNull(t *testing.T) {
	srv := newClaimsTestServer(t, playbooksclaimsmocks.NewFakeRepository())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/playbooks/claims/missing", nil)
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body struct {
		Claim *claimDTO `json:"claim"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Claim != nil {
		t.Fatalf("expected nil claim, got %+v", body.Claim)
	}
}

func TestHandleGetPlaybooksClaim_Active(t *testing.T) {
	repo := playbooksclaimsmocks.NewFakeRepository()
	svc := playbooksclaims.NewService(playbooksclaims.Config{Repo: repo})
	if _, err := svc.TryAcquire(t.Context(), playbooksclaims.AcquireInput{
		ScenarioName: "demo", RunID: "run-1", Mode: playbooksclaims.ModeFallback, StartedBy: "tester",
	}); err != nil {
		t.Fatalf("acquire: %v", err)
	}

	srv := &Server{
		config:          Config{Port: "0"},
		router:          mux.NewRouter(),
		logger:          log.New(io.Discard, "", 0),
		playbooksClaims: svc,
	}
	srv.setupRoutes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/playbooks/claims/demo", nil)
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body struct {
		Claim *claimDTO `json:"claim"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Claim == nil || body.Claim.RunID != "run-1" || body.Claim.Mode != "fallback" {
		t.Fatalf("unexpected claim: %+v", body.Claim)
	}
}

func TestHandleReleasePlaybooksClaim_BreaksActive(t *testing.T) {
	repo := playbooksclaimsmocks.NewFakeRepository()
	svc := playbooksclaims.NewService(playbooksclaims.Config{Repo: repo})
	if _, err := svc.TryAcquire(t.Context(), playbooksclaims.AcquireInput{
		ScenarioName: "demo", RunID: "run-1", Mode: playbooksclaims.ModeRouted, StartedBy: "tester",
	}); err != nil {
		t.Fatalf("acquire: %v", err)
	}

	srv := &Server{
		config:          Config{Port: "0"},
		router:          mux.NewRouter(),
		logger:          log.New(io.Discard, "", 0),
		playbooksClaims: svc,
	}
	srv.setupRoutes()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/playbooks/claims/demo/release", nil)
	req.Header.Set("X-Vrooli-Actor", "admin@cli")
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Subsequent Get → null.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/playbooks/claims/demo", nil)
	rec = httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)
	var body struct {
		Claim *claimDTO `json:"claim"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Claim != nil {
		t.Fatalf("expected claim cleared, got %+v", body.Claim)
	}
}

func TestHandleReleasePlaybooksClaim_NoneReturns404(t *testing.T) {
	srv := newClaimsTestServer(t, playbooksclaimsmocks.NewFakeRepository())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/playbooks/claims/demo/release", nil)
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}
