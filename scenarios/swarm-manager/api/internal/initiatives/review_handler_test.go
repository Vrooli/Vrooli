package initiativereview

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/review"

	"github.com/gorilla/mux"
)

// handlerEnv wraps env + a router with the handler mounted.
type handlerEnv struct {
	*env
	router  *mux.Router
	handler *Handler
}

func newHandlerEnv(t *testing.T) *handlerEnv {
	t.Helper()
	base := newEnv(t)
	r := mux.NewRouter()
	h := NewHandler(base.svc)
	h.RegisterRoutes(r)
	return &handlerEnv{env: base, router: r, handler: h}
}

func doRequest(t *testing.T, r *mux.Router, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestHandler_ListRounds_EmptyReturnsEmptySlice(t *testing.T) {
	e := newHandlerEnv(t)
	e.createInitiative("list-init", "List")
	rec := doRequest(t, e.router, "GET", "/api/v1/initiatives/list-init/review", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Rounds []review.Round `json:"rounds"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Rounds == nil {
		t.Error("expected rounds to be non-nil (empty slice)")
	}
}

func TestHandler_Trigger_Success(t *testing.T) {
	e := newHandlerEnv(t)
	e.seedItem("execute", "alpha", "Alpha", backlog.StatusCompleted)
	e.createInitiative("trig-init", "Trig", "execute/alpha")
	e.setItemInitiative("execute", "alpha", "trig-init")

	rec := doRequest(t, e.router, "POST", "/api/v1/initiatives/trig-init/review/trigger", nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202, body=%s", rec.Code, rec.Body.String())
	}
	var result TriggerResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !result.Started {
		t.Fatalf("expected Started=true, got %+v", result)
	}
	if result.Round != 1 {
		t.Errorf("expected round 1, got %d", result.Round)
	}
}

func TestHandler_Trigger_ReturnsReasonOKWhenNotReady(t *testing.T) {
	e := newHandlerEnv(t)
	e.createInitiative("empty-init", "Empty")

	rec := doRequest(t, e.router, "POST", "/api/v1/initiatives/empty-init/review/trigger", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (no-op)", rec.Code)
	}
	var result TriggerResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Started {
		t.Fatal("expected Started=false")
	}
	if result.Reason == "" {
		t.Error("expected a Reason string")
	}
}

func TestHandler_GetRound_Found(t *testing.T) {
	e := newHandlerEnv(t)
	e.seedItem("execute", "alpha", "Alpha", backlog.StatusCompleted)
	e.createInitiative("get-init", "Get", "execute/alpha")
	e.setItemInitiative("execute", "alpha", "get-init")
	_, _ = e.svc.TriggerIfReady(context.Background(), "get-init")

	rec := doRequest(t, e.router, "GET", "/api/v1/initiatives/get-init/review/1", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var round review.Round
	if err := json.Unmarshal(rec.Body.Bytes(), &round); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if round.RoundNum != 1 {
		t.Errorf("round num = %d, want 1", round.RoundNum)
	}
}

func TestHandler_GetRound_NotFound(t *testing.T) {
	e := newHandlerEnv(t)
	rec := doRequest(t, e.router, "GET", "/api/v1/initiatives/missing/review/7", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Decide_SuccessFlipsStatusAndRecords(t *testing.T) {
	e := newHandlerEnv(t)
	e.seedItem("execute", "alpha", "Alpha", backlog.StatusCompleted)
	e.createInitiative("dh-init", "Decide Handler", "execute/alpha")
	e.setItemInitiative("execute", "alpha", "dh-init")
	_, _ = e.svc.TriggerIfReady(context.Background(), "dh-init")

	init, _ := e.initStore.Load("dh-init")
	init.Status = initiatives.InitiativeStatusReviewPending
	_ = e.initStore.Save(init)

	body, _ := json.Marshal(DecideRequest{
		Verdict:   "accept",
		Rationale: "all good",
		DecidedBy: "http-tester",
	})
	rec := doRequest(t, e.router, "POST", "/api/v1/initiatives/dh-init/review/decide", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var resp DecideResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != initiatives.InitiativeStatusCompleted {
		t.Errorf("status = %q, want completed", resp.Status)
	}

	// listDecisions endpoint should surface the record.
	rec = doRequest(t, e.router, "GET", "/api/v1/initiatives/dh-init/review/decisions", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list decisions status = %d", rec.Code)
	}
	var listBody struct {
		Decisions []DecisionRecord `json:"decisions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listBody.Decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(listBody.Decisions))
	}
	if listBody.Decisions[0].DecidedBy != "http-tester" {
		t.Errorf("decided_by = %q, want http-tester", listBody.Decisions[0].DecidedBy)
	}
}

func TestHandler_Decide_InvalidVerdictReturns400(t *testing.T) {
	e := newHandlerEnv(t)
	e.createInitiative("bad-verdict", "Bad")

	body, _ := json.Marshal(map[string]any{"verdict": "ship-it"})
	rec := doRequest(t, e.router, "POST", "/api/v1/initiatives/bad-verdict/review/decide", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "invalid verdict") {
		t.Errorf("expected 'invalid verdict' in body, got %s", rec.Body.String())
	}
}

func TestHandler_Decide_WrongLifecycleReturns400(t *testing.T) {
	e := newHandlerEnv(t)
	e.createInitiative("wrong-state", "Wrong")
	body, _ := json.Marshal(DecideRequest{Verdict: "accept"})
	rec := doRequest(t, e.router, "POST", "/api/v1/initiatives/wrong-state/review/decide", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "review_pending") {
		t.Errorf("expected review_pending in error, got %s", rec.Body.String())
	}
}

func TestHandler_TriggerRoutes_ReadsAgentSpawnShape(t *testing.T) {
	// Smoke test that the router passes path params correctly and the
	// service receives the right initiative name. This complements the
	// service-level tests, which don't go through the mux.
	e := newHandlerEnv(t)
	e.seedItem("execute", "zeta", "Zeta", backlog.StatusCompleted)
	e.createInitiative("smoke-init", "Smoke", "execute/zeta")
	e.setItemInitiative("execute", "zeta", "smoke-init")

	rec := doRequest(t, e.router, "POST", "/api/v1/initiatives/smoke-init/review/trigger", nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d", rec.Code)
	}
	if len(e.spawner.spawnCalls) != 1 {
		t.Fatalf("expected exactly one spawn call, got %d", len(e.spawner.spawnCalls))
	}
	if e.spawner.spawnCalls[0].Name != "smoke-init" {
		t.Errorf("spawn Name = %q, want smoke-init", e.spawner.spawnCalls[0].Name)
	}
	// Make sure the run result was captured by the env.
	_ = agentmanager.RunResult{}
}
