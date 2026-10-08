package heartbeat

import (
	"context"
	"encoding/json"
	"github.com/vrooli/api-core/owneridentity"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type auth01QueueExecutor struct{ calls chan context.Context }

func (e *auth01QueueExecutor) Execute(ctx context.Context, _, _, _ string) (*ExecutionResult, error) {
	e.calls <- ctx
	return &ExecutionResult{}, nil
}
func TestAuth01ManualQueueCarriesOriginalProof(t *testing.T) {
	e := &auth01QueueExecutor{make(chan context.Context, 2)}
	client := &AgentManagerClient{createCallerVerifier: auth01ConversationVerifier{}}
	q := newTeamExecutionContext("fixture", e, t.TempDir(), client)
	q.running["occupied"] = runningEntry{}
	parent, cancel := context.WithCancel(withCreateRunCaller(context.Background(), "Bearer fixture-secret", ""))
	if _, err := q.Enqueue(parent, "one", "profile"); err != nil {
		t.Fatal(err)
	}
	cancel()
	encoded, _ := json.Marshal(q.queue)
	if strings.Contains(string(encoded), "fixture-secret") {
		t.Fatal("credential persisted")
	}
	original := q.queue[0].caller
	if _, err := q.Enqueue(withCreateRunCaller(context.Background(), "Bearer replacement", ""), "one", "profile"); !IsMemberAlreadyQueued(err) {
		t.Fatal("duplicate renewed queue")
	}
	if q.queue[0].caller != original {
		t.Fatal("caller replaced")
	}
	q.OnMemberComplete("occupied")
	select {
	case ctx := <-e.calls:
		proof, err := owneridentity.CreateRunAuthorization(ctx, time.Now())
		if err != nil || proof != "Bearer fixture-secret" {
			t.Fatal("proof lost", err)
		}
	case <-time.After(time.Second):
		t.Fatal("valid caller not dispatched")
	}
	q.executionWG.Wait()
}
func TestAuth01QueueMissingExpiredRecoveryHasNoDispatch(t *testing.T) {
	client := &AgentManagerClient{createCallerVerifier: auth01ConversationVerifier{}}
	q := newTeamExecutionContext("fixture", &auth01QueueExecutor{make(chan context.Context, 1)}, t.TempDir(), client)
	if _, err := q.Enqueue(context.Background(), "bad", "profile"); err == nil || len(q.running) != 0 || len(q.queue) != 0 {
		t.Fatal("anonymous admission effects")
	}
	// Proof validated in the past is expired now, without timing sleeps.
	expired, err := owneridentity.AuthorizeCreateRunCaller(context.Background(), http.Header{"Authorization": []string{"Bearer fixture"}}, auth01QueueExpiredVerifier{}, time.Now().Add(-2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	q.queue = []queuedExecution{{AgentID: "expired", caller: expired}, {AgentID: "recovered"}}
	q.queued["expired"] = true
	q.queued["recovered"] = true
	if got := q.dispatchAvailableLocked(); len(got) != 0 || len(q.running) != 0 || len(q.queue) != 2 {
		t.Fatal("held queue dispatched")
	}
	for _, entry := range q.queue {
		if entry.State != "caller_required" {
			t.Fatal("missing hold classification")
		}
	}
}

type auth01QueueExpiredVerifier struct{}

func (auth01QueueExpiredVerifier) Validate(context.Context, string) (owneridentity.Identity, error) {
	return owneridentity.Identity{Subject: "fixture", Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(-time.Hour)}, nil
}
func TestAuth01ManualTriggersRefuseBeforeEffects(t *testing.T) {
	h := &Handlers{runCallerValidator: auth01ConversationVerifier{}}
	for _, handler := range []func(http.ResponseWriter, *http.Request){h.TriggerHeartbeat, h.TriggerTeam} {
		for _, header := range []http.Header{{}, {"Authorization": []string{"Bearer valid", "Bearer duplicate"}}, {"Authorization": []string{"Bearer valid"}, "X-Agent-Identity-Token": []string{"wrong-channel"}}} {
			r := httptest.NewRequest("POST", "/fixture", nil)
			r.Header = header
			w := httptest.NewRecorder()
			handler(w, r)
			if w.Code != 401 {
				t.Fatal(w.Code)
			}
		}
	}
}

func TestAuth01ManualQueueRecoveryHeldWithoutProof(t *testing.T) {
	dir := t.TempDir()
	client := &AgentManagerClient{createCallerVerifier: auth01ConversationVerifier{}}
	q := newTeamExecutionContext("fixture", &captureExecutor{}, dir, client)
	q.running["occupied"] = runningEntry{}
	if _, err := q.Enqueue(withCreateRunCaller(context.Background(), "Bearer fixture-secret", ""), "manual", "profile"); err != nil {
		t.Fatal(err)
	}
	// Retain only the pending intent, avoiding any owner read on recovery.
	q.running = make(map[string]runningEntry)
	q.persistLocked()
	restored := newTeamExecutionContext("fixture", &captureExecutor{}, dir, client)
	restored.Recover(context.Background())
	if len(restored.queue) != 1 || restored.queue[0].State != "caller_required" || restored.queue[0].caller != nil {
		t.Fatal("manual intent lost or proof revived")
	}
	if len(restored.dispatchAvailableLocked()) != 0 {
		t.Fatal("recovered intent dispatched")
	}
}
