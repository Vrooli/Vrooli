package feedback

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// List used to return cached on-disk state, leaving completed rounds wedged
// at agent_thinking until the user clicked into them. List now calls
// EnsurePolledTurn for any agent_thinking row before returning.
func TestList_PollsAgentThinkingRounds(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	poller := &fakePoller{
		enabled: true,
		state:   RunState{Status: "complete", Summary: "all done"},
	}
	svc := withPoller(t, env, poller)

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "looks awful",
	})
	if err != nil {
		t.Fatal(err)
	}
	if round.Status != RoundStatusAgentThinking {
		t.Fatalf("precondition: expected agent_thinking, got %s", round.Status)
	}

	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/initiatives/ui-rewrite/feedback", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("List returned %d, body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Rounds []Round `json:"rounds"`
		Count  int     `json:"count"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(body.Rounds) != 1 {
		t.Fatalf("expected 1 round, got %d", len(body.Rounds))
	}
	if body.Rounds[0].Status != RoundStatusAwaitingUser {
		t.Fatalf("expected list to advance round to awaiting_user, got %s", body.Rounds[0].Status)
	}
	if poller.calls == 0 {
		t.Fatal("expected poller to be called from List")
	}
}
