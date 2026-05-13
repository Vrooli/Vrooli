package feedback

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"swarm-manager/internal/initiativelock"
	"swarm-manager/internal/proposals"
)

// fakeCanceller records StopRun calls so tests can assert that Cancel
// reaches the agent-manager and tolerates failures.
type fakeCanceller struct {
	calls   []string
	stopErr error
}

func (c *fakeCanceller) StopRun(_ context.Context, runID string) error {
	c.calls = append(c.calls, runID)
	return c.stopErr
}

// withCanceller rebuilds a service with the supplied canceller wired in.
// Mirrors withPoller — newServiceEnv doesn't wire a canceller by default.
func withCanceller(t *testing.T, env *serviceEnv, c RunCanceller) *Service {
	t.Helper()
	stateBuilder := func(name string) (proposals.CurrentState, error) {
		return proposals.CurrentState{InitiativeName: name}, nil
	}
	svc, err := NewService(Config{
		Store:        env.store,
		Lock:         env.lock,
		Spawner:      env.spawner,
		Canceller:    c,
		Apply:        env.applier,
		StateBuilder: stateBuilder,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}

func TestCancel_HappyPath(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	canc := &fakeCanceller{}
	svc := withCanceller(t, env, canc)

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "agent went off the rails",
	})
	if err != nil {
		t.Fatal(err)
	}
	if round.Status != RoundStatusAgentThinking {
		t.Fatalf("precondition: expected agent_thinking, got %s", round.Status)
	}
	if round.RunID == "" {
		t.Fatal("precondition: expected non-empty RunID")
	}
	originalRunID := round.RunID

	// Lock should currently be held.
	if h, _ := env.lock.Inspect("ui-rewrite"); h == nil {
		t.Fatal("precondition: expected initiative lock to be held")
	}

	out, err := svc.Cancel(context.Background(), CancelRequest{
		InitiativeName: "ui-rewrite",
		RoundNumber:    round.Number,
		Rationale:      "user clicked Cancel",
		DecidedBy:      "tester",
	})
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if out.Status != RoundStatusDismissed {
		t.Fatalf("expected dismissed, got %s", out.Status)
	}
	if out.Decision == nil || out.Decision.Kind != DecisionDismiss {
		t.Fatalf("expected dismiss decision, got %#v", out.Decision)
	}
	if out.Decision.Rationale != "user clicked Cancel" {
		t.Fatalf("rationale not preserved: got %q", out.Decision.Rationale)
	}
	if out.RunID != "" {
		t.Fatalf("expected RunID cleared after cancel, got %q", out.RunID)
	}
	if len(canc.calls) != 1 || canc.calls[0] != originalRunID {
		t.Fatalf("expected StopRun(%q), got %v", originalRunID, canc.calls)
	}
	// Lock should be released.
	if h, _ := env.lock.Inspect("ui-rewrite"); h != nil {
		t.Fatalf("expected lock released after cancel, still held by %v", h)
	}
}

func TestCancel_DefaultRationale(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	svc := withCanceller(t, env, &fakeCanceller{})

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "stuck",
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := svc.Cancel(context.Background(), CancelRequest{
		InitiativeName: "ui-rewrite",
		RoundNumber:    round.Number,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Decision == nil || out.Decision.Rationale != "cancelled by user" {
		t.Fatalf("expected default rationale, got %#v", out.Decision)
	}
}

func TestCancel_NilCanceller_StillSucceeds(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	svc := withCanceller(t, env, nil) // no canceller wired

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "stuck",
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := svc.Cancel(context.Background(), CancelRequest{
		InitiativeName: "ui-rewrite",
		RoundNumber:    round.Number,
	})
	if err != nil {
		t.Fatalf("Cancel without canceller should succeed, got %v", err)
	}
	if out.Status != RoundStatusDismissed {
		t.Fatalf("expected dismissed, got %s", out.Status)
	}
	if h, _ := env.lock.Inspect("ui-rewrite"); h != nil {
		t.Fatal("expected lock released even without canceller")
	}
}

func TestCancel_StopRunError_DoesNotBlockCancellation(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	canc := &fakeCanceller{stopErr: errors.New("agent-manager unreachable")}
	svc := withCanceller(t, env, canc)

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "stuck",
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := svc.Cancel(context.Background(), CancelRequest{
		InitiativeName: "ui-rewrite",
		RoundNumber:    round.Number,
	})
	if err != nil {
		t.Fatalf("StopRun error must not block local cancel, got %v", err)
	}
	if out.Status != RoundStatusDismissed {
		t.Fatalf("expected dismissed, got %s", out.Status)
	}
	if h, _ := env.lock.Inspect("ui-rewrite"); h != nil {
		t.Fatal("expected lock released even after StopRun error")
	}
}

func TestCancel_AlreadyTerminal_ReturnsErr(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	svc := withCanceller(t, env, &fakeCanceller{})

	// A note round lands directly in dismissed.
	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeNote,
		Text:           "just a note",
	})
	if err != nil {
		t.Fatal(err)
	}
	if round.Status != RoundStatusDismissed {
		t.Fatalf("precondition: expected dismissed note, got %s", round.Status)
	}

	if _, err := svc.Cancel(context.Background(), CancelRequest{
		InitiativeName: "ui-rewrite",
		RoundNumber:    round.Number,
	}); !errors.Is(err, ErrRoundAlreadyTerminal) {
		t.Fatalf("expected ErrRoundAlreadyTerminal, got %v", err)
	}
}

func TestCancel_NoRunID_SkipsStopRun(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	canc := &fakeCanceller{}
	svc := withCanceller(t, env, canc)

	// Hand-craft an agent_thinking round with no RunID — the kind that
	// could exist if the spawn path ever lost the RunID write.
	r := Round{
		InitiativeName: "ui-rewrite",
		Number:         77,
		Slug:           "manual",
		Type:           RoundTypeFeedback,
		Status:         RoundStatusAgentThinking,
		Submission:     Submission{Text: "x", CreatedAt: "2024-01-01T00:00:00Z"},
		CreatedAt:      "2024-01-01T00:00:00Z",
		UpdatedAt:      "2024-01-01T00:00:00Z",
	}
	if err := env.store.SaveRound(r); err != nil {
		t.Fatal(err)
	}

	out, err := svc.Cancel(context.Background(), CancelRequest{
		InitiativeName: "ui-rewrite",
		RoundNumber:    77,
	})
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if out.Status != RoundStatusDismissed {
		t.Fatalf("expected dismissed, got %s", out.Status)
	}
	if len(canc.calls) != 0 {
		t.Fatalf("StopRun should not be called when RunID is empty, got %v", canc.calls)
	}
}

func TestCancel_RoundNotFound_ReturnsErr(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	svc := withCanceller(t, env, &fakeCanceller{})

	_, err := svc.Cancel(context.Background(), CancelRequest{
		InitiativeName: "ui-rewrite",
		RoundNumber:    999,
	})
	if !errors.Is(err, ErrRoundNotFound) {
		t.Fatalf("expected ErrRoundNotFound, got %v", err)
	}
}

func TestDecide_DismissActiveRound_RoutesThroughCancel(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	canc := &fakeCanceller{}
	svc := withCanceller(t, env, canc)

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "stuck",
	})
	if err != nil {
		t.Fatal(err)
	}

	out, _, err := svc.Decide(context.Background(), DecideRequest{
		InitiativeName: "ui-rewrite",
		RoundNumber:    round.Number,
		Kind:           DecisionDismiss,
		Rationale:      "give up",
	})
	if err != nil {
		t.Fatalf("Decide(dismiss) on agent_thinking should route through Cancel: %v", err)
	}
	if out.Status != RoundStatusDismissed {
		t.Fatalf("expected dismissed, got %s", out.Status)
	}
	if len(canc.calls) != 1 {
		t.Fatalf("expected StopRun via Cancel, got %v", canc.calls)
	}
	if h, _ := env.lock.Inspect("ui-rewrite"); h != nil {
		t.Fatal("expected lock released")
	}
}

func TestDecide_NonDismissOnActiveRound_StillRejected(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	svc := withCanceller(t, env, &fakeCanceller{})

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "stuck",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := svc.Decide(context.Background(), DecideRequest{
		InitiativeName: "ui-rewrite",
		RoundNumber:    round.Number,
		Kind:           DecisionAccept,
	}); err == nil {
		t.Fatal("expected error: accept on agent_thinking should still fail")
	}
}

func TestCancel_LockAlreadyReleased_StillSucceeds(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	canc := &fakeCanceller{}
	svc := withCanceller(t, env, canc)

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "stuck",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Manually release the lock to simulate a prior cleanup or sweeper.
	if err := env.lock.Release("ui-rewrite", round.RunID); err != nil {
		t.Fatal(err)
	}

	out, err := svc.Cancel(context.Background(), CancelRequest{
		InitiativeName: "ui-rewrite",
		RoundNumber:    round.Number,
	})
	if err != nil {
		t.Fatalf("Cancel after lock release: %v", err)
	}
	if out.Status != RoundStatusDismissed {
		t.Fatalf("expected dismissed, got %s", out.Status)
	}
}

// --- Handler-level tests -----------------------------------------------

// newCancelHandlerEnv builds a handler stack with a fakeCanceller wired in
// so HTTP tests can exercise the cancel route end-to-end.
type cancelHandlerEnv struct {
	*serviceEnv
	router *mux.Router
	canc   *fakeCanceller
}

func newCancelHandlerEnv(t *testing.T) *cancelHandlerEnv {
	t.Helper()
	env := newServiceEnv(t)
	canc := &fakeCanceller{}
	svc, err := NewService(Config{
		Store:     env.store,
		Lock:      env.lock,
		Spawner:   env.spawner,
		Canceller: canc,
		Apply:     env.applier,
		StateBuilder: func(name string) (proposals.CurrentState, error) {
			return proposals.CurrentState{InitiativeName: name}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	env.svc = svc
	handler := NewHandler(svc)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	return &cancelHandlerEnv{serviceEnv: env, router: router, canc: canc}
}

func (h *cancelHandlerEnv) do(method, path, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.router.ServeHTTP(rec, req)
	return rec
}

func TestHandler_Cancel_200(t *testing.T) {
	env := newCancelHandlerEnv(t)

	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "stuck",
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := env.do("POST",
		fmt.Sprintf("/api/v1/initiatives/ui-rewrite/feedback/%d/cancel", round.Number),
		`{"rationale":"timed out"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got Round
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Status != RoundStatusDismissed {
		t.Fatalf("expected dismissed, got %s", got.Status)
	}
	if got.Decision == nil || got.Decision.Rationale != "timed out" {
		t.Fatalf("rationale lost: %#v", got.Decision)
	}
	if len(env.canc.calls) != 1 {
		t.Fatalf("expected one StopRun call, got %v", env.canc.calls)
	}
}

func TestHandler_Cancel_404(t *testing.T) {
	env := newCancelHandlerEnv(t)
	rec := env.do("POST", "/api/v1/initiatives/ui-rewrite/feedback/999/cancel", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Cancel_409Terminal(t *testing.T) {
	env := newCancelHandlerEnv(t)
	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeNote,
		Text:           "note",
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := env.do("POST",
		fmt.Sprintf("/api/v1/initiatives/ui-rewrite/feedback/%d/cancel", round.Number), "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// Smoke check that initiativelock.Holder is reachable from this test file
// (sanity check after editing imports).
var _ initiativelock.Holder
