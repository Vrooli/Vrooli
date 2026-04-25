package feedback

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestDelete_TerminalRoundIsRemoved(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	svc := env.svc

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeNote,
		Text:           "just a note", // note rounds are terminal on creation
	})
	if err != nil {
		t.Fatal(err)
	}
	if !round.Status.IsTerminal() {
		t.Fatalf("precondition: expected terminal status, got %s", round.Status)
	}

	if err := svc.Delete("ui-rewrite", round.Number); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := env.store.LoadRound("ui-rewrite", round.Number); !errors.Is(err, ErrRoundNotFound) {
		t.Fatalf("expected ErrRoundNotFound after delete, got %v", err)
	}
}

func TestDelete_RefusesInFlightRound(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	svc := env.svc

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "wip",
	})
	if err != nil {
		t.Fatal(err)
	}
	if round.Status.IsTerminal() {
		t.Fatalf("precondition: expected non-terminal, got %s", round.Status)
	}

	err = svc.Delete("ui-rewrite", round.Number)
	if !errors.Is(err, ErrRoundNotTerminal) {
		t.Fatalf("expected ErrRoundNotTerminal, got %v", err)
	}
	if _, loadErr := env.store.LoadRound("ui-rewrite", round.Number); loadErr != nil {
		t.Fatalf("round must still exist: %v", loadErr)
	}
}

func TestDelete_HandlerReturns204(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)

	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeNote,
		Text:           "note",
	})
	if err != nil {
		t.Fatal(err)
	}

	router := mux.NewRouter()
	NewHandler(env.svc).RegisterRoutes(router)
	target := "/api/v1/initiatives/ui-rewrite/feedback/" + itoa(round.Number)
	req := httptest.NewRequest(http.MethodDelete, target, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestDelete_HandlerReturns409ForActiveRound(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)

	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "wip",
	})
	if err != nil {
		t.Fatal(err)
	}

	router := mux.NewRouter()
	NewHandler(env.svc).RegisterRoutes(router)
	target := "/api/v1/initiatives/ui-rewrite/feedback/" + itoa(round.Number)
	req := httptest.NewRequest(http.MethodDelete, target, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
