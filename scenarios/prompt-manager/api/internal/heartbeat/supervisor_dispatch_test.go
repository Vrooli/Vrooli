package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	api "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"prompt-manager/internal/teamconfig"
)

type supervisorDispatchFake struct {
	*mockAgentClient
	requests []*api.CreateSupervisorRunRequest
	tokens   []string
}

func (f *supervisorDispatchFake) CreateSupervisorRun(_ context.Context, req *api.CreateSupervisorRunRequest, token string) (*Run, error) {
	f.requests = append(f.requests, req)
	f.tokens = append(f.tokens, token)
	return &Run{ID: "wake-run-1", Status: "running"}, nil
}

func TestStandingSupervisorDelegatedFutureWakesAndMultipleEfforts(t *testing.T) {
	f := newSupervisionFixture(t)
	agent := &supervisorDispatchFake{mockAgentClient: f.agent}
	f.s.Agent = agent
	f.s.DispatchCredential = func(context.Context) (string, error) { return "secret-dispatch-fixture", nil }
	f.cfg.Supervision.DispatchAuthorization = &teamconfig.SupervisorDispatchBinding{EffortRef: "service:standing", AuthorizationID: "owner-granted-id"}
	f.cfg.Supervision.MaxEffortsPerWake = 2
	f.owner.rows = []EffortObservation{effort("effort:first"), effort("effort:second")}
	f.tick(t)
	f.dispatch(t)
	f.receipt(t, "quiet")
	f.agent.getRuns["wake-run-1"].Status = "complete"
	f.now = f.now.Add(time.Minute)
	f.tick(t)
	f.owner.rows[0].EvidenceRevision = "future-proof"
	f.tick(t)
	f.dispatch(t)
	if len(agent.requests) != 2 || len(f.agent.createRunCalls) != 0 {
		t.Fatal("delegated wake missing or ordinary create fallback occurred")
	}
	first, next := agent.requests[0], agent.requests[1]
	if first.IdempotencyKey == next.IdempotencyKey || first.AuthorizationId != next.AuthorizationId || first.TeamId != "supervisors" || first.MemberId != "leader" || len(first.WorkReferences) != 2 {
		t.Fatal("future wakes lost stable purpose binding or multi-effort attribution")
	}
	state, err := f.s.State.Load("supervisors", "leader")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(state)
	if strings.Contains(string(body), agent.tokens[0]) {
		t.Fatal("wake state persisted dispatcher credential")
	}
}

func TestStandingSupervisorMissingCredentialFailsBeforeTaskWithoutFallback(t *testing.T) {
	f := newSupervisionFixture(t)
	f.s.Agent = &supervisorDispatchFake{mockAgentClient: f.agent}
	f.cfg.Supervision.DispatchAuthorization = &teamconfig.SupervisorDispatchBinding{EffortRef: "service:standing", AuthorizationID: "grant"}
	f.s.DispatchCredential = func(context.Context) (string, error) { return "", errors.New("private-authority-detail") }
	f.owner.rows = []EffortObservation{effort("work-reference-is-not-a-grant")}
	f.tick(t)
	_, err := f.s.Dispatch(context.Background(), "supervisors", "leader")
	if err == nil || strings.Contains(err.Error(), "private-authority-detail") || !strings.Contains(err.Error(), "issue-dispatch") {
		t.Fatal("missing actionable secret-safe authority error")
	}
	state, _ := f.s.State.Load("supervisors", "leader")
	if state.Pending.DispatchStarted || len(f.agent.createRunCalls) != 0 || len(f.agent.createTaskCalls) != 0 {
		t.Fatal("missing credential created work or widened default authority")
	}
}

func TestCreateSupervisorRunCredentialStaysInHeaderAndErrorsAreRedacted(t *testing.T) {
	const token = "secret-dispatch-credential"
	for _, refused := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "refused"}[refused], func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				body, _ := io.ReadAll(r.Body)
				if r.Header.Get("Authorization") != "Bearer "+token || r.Header.Get("X-Agent-Identity-Token") != "" || strings.Contains(string(body), token) || !strings.HasSuffix(r.URL.Path, "/CreateSupervisorRun") {
					t.Error("credential transport escaped dedicated bearer-only RPC")
				}
				w.Header().Set("Content-Type", "application/json")
				if refused {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(`{"code":"permission_denied","message":"` + token + `"}`))
					return
				}
				_, _ = w.Write([]byte(`{"run":{"id":"wake-run","status":"RUN_STATUS_RUNNING"}}`))
			}))
			defer server.Close()
			run, err := newTestClient(t, server).CreateSupervisorRun(context.Background(), &api.CreateSupervisorRunRequest{AuthorizationId: "grant", IdempotencyKey: "wake"}, token)
			if refused {
				if err == nil || strings.Contains(err.Error(), token) {
					t.Fatal("refusal leaked token")
				}
			} else if err != nil || run.ID != "wake-run" {
				t.Fatalf("typed response failed: %v", err)
			}
			if calls != 1 {
				t.Fatal("client retried delegated dispatch")
			}
		})
	}
}

func TestCreateSupervisorRunNeverForwardsCredentialOnRedirect(t *testing.T) {
	forwarded := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { forwarded = true }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	_, err := newTestClient(t, server).CreateSupervisorRun(context.Background(), &api.CreateSupervisorRunRequest{AuthorizationId: "grant", IdempotencyKey: "wake"}, "secret")
	if err == nil || forwarded {
		t.Fatal("credential followed redirect")
	}
}
