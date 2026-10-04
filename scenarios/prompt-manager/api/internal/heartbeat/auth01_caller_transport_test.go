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

func TestAuth01DirectCreateRunPreservesOfferedProofWithoutDefaultMutation(t *testing.T) {
	for _, kind := range []string{"human", "run", "both", "refused", "redirect"} {
		t.Run(kind, func(t *testing.T) {
			auth, runProof := "Bearer fixture-owner", ""
			if kind == "run" {
				auth = ""
			}
			if kind == "run" || kind == "both" {
				runProof = "fixture-run"
			}
			calls := 0
			parent := "fixture-parent"
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Path != "/api/v1/runs" {
					t.Error("credential redirected")
				}
				if r.Header.Get("Authorization") != auth || r.Header.Get("X-Agent-Identity-Token") != runProof {
					t.Error("offered channel dropped/substituted")
				}
				var req CreateRunRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
				}
				if req.ParentRunID == nil || *req.ParentRunID != parent {
					t.Error("explicit parent lost")
				}
				data, _ := json.Marshal(req)
				if strings.Contains(string(data), "fixture-owner") || strings.Contains(string(data), "fixture-run") {
					t.Error("proof persisted in body")
				}
				if kind == "refused" {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(auth + " " + runProof))
					return
				}
				if kind == "redirect" {
					w.Header().Set("Location", "/other")
					w.WriteHeader(http.StatusTemporaryRedirect)
					return
				}
				_, _ = w.Write([]byte(`{"run":{"id":"fixture-accepted"}}`))
			}))
			defer srv.Close()
			client := newTestClient(t, srv)
			ctx := withCreateRunCaller(context.Background(), auth, runProof)
			got, err := client.CreateRun(ctx, &CreateRunRequest{TaskID: "fixture-task", ParentRunID: &parent, IdempotencyKey: "fixture-key"})
			if kind == "refused" || kind == "redirect" {
				if got != nil || err == nil || strings.Contains(err.Error(), "fixture-owner") {
					t.Fatalf("refusal ignored or proof leaked: %v", err)
				}
			} else if err != nil || got == nil {
				t.Fatalf("proof forwarding failed: %v", err)
			}
			if calls != 1 {
				t.Fatalf("creation retried/redirected %d", calls)
			}
			if createRunCaller(context.Background()) != (createRunCallerProof{}) {
				t.Fatal("default proof changed")
			}
		})
	}
}
func TestAuth01DirectCreateRunIngressRejectsMissingOrMalformedProofBeforeOwnerCall(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer srv.Close()
	h := &Handlers{agentClient: newTestClient(t, srv)}
	for _, headers := range [][2]string{{"", ""}, {"Bearer", ""}, {"fixture-run", ""}, {"Bearer fixture-owner", " "}} {
		req := httptest.NewRequest(http.MethodPost, "/runs", strings.NewReader(`{"task_id":"fixture-task"}`))
		req.Header.Set("Authorization", headers[0])
		req.Header.Set("X-Agent-Identity-Token", headers[1])
		w := httptest.NewRecorder()
		h.CreateRun(w, req)
		if w.Code != http.StatusUnauthorized || calls != 0 {
			t.Fatalf("invalid proof reached owner: status%d calls%d", w.Code, calls)
		}
	}
}
func TestAuth01CreateRunProofDoesNotAuthorizeSiblingRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("X-Agent-Identity-Token") != "" {
			t.Error("CreateRun proof widened to sibling")
		}
		_, _ = w.Write([]byte(`{"task":{"id":"fixture-task"}}`))
	}))
	defer srv.Close()
	client := newTestClient(t, srv)
	_, err := client.CreateTask(withCreateRunCaller(context.Background(), "Bearer fixture-owner", "fixture-run"), &Task{Title: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAuth01RESTIngressPreservesCallerAndExactParent(t *testing.T) {
	for _, headers := range [][2]string{{"Bearer fixture-owner", ""}, {"", "fixture-run"}, {"Bearer fixture-owner", "fixture-run"}} {
		calls := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.Header.Get("Authorization") != headers[0] || r.Header.Get("X-Agent-Identity-Token") != headers[1] {
				t.Error("ingress lost offered proof")
			}
			var payload CreateRunRequest
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.ParentRunID == nil || *payload.ParentRunID != "fixture-parent" {
				t.Error("ingress lost exact parent")
			}
			_, _ = w.Write([]byte(`{"run":{"id":"fixture-run"}}`))
		}))
		h := &Handlers{agentClient: newTestClient(t, srv)}
		req := httptest.NewRequest(http.MethodPost, "/runs", strings.NewReader(`{"task_id":"fixture-task","parent_run_id":"fixture-parent"}`))
		if headers[0] != "" {
			req.Header.Set("Authorization", headers[0])
		}
		if headers[1] != "" {
			req.Header.Set("X-Agent-Identity-Token", headers[1])
		}
		w := httptest.NewRecorder()
		h.CreateRun(w, req)
		srv.Close()
		if w.Code != http.StatusOK || calls != 1 {
			t.Fatalf("ingress: status%d calls%d body%s", w.Code, calls, w.Body.String())
		}
	}
}

type auth01ConversationVerifier struct{ invalid bool }

func (v auth01ConversationVerifier) Validate(_ context.Context, token string) (owneridentity.Identity, error) {
	if v.invalid {
		return owneridentity.Identity{}, owneridentity.ErrUnauthenticated
	}
	return owneridentity.Identity{Subject: "fixture", Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func TestAuth01ConversationInvalidCallerRejectsBeforeTaskOrReplay(t *testing.T) {
	h, client, _ := conversationFixture(t)
	h.runCallerValidator = auth01ConversationVerifier{invalid: true}
	req := httptest.NewRequest(http.MethodPost, "/runs", strings.NewReader(conversationBody))
	req.Header.Set("Authorization", "Bearer invalid-fixture")
	w := httptest.NewRecorder()
	h.CreateRun(w, req)
	if w.Code != 401 || len(client.createTaskCalls) != 0 || len(client.createRunCalls) != 0 {
		t.Fatal("invalid conversation caller reached writes or replay")
	}
}

func TestAuth01AmbiguousOrBlankOfferedChannelsRejectBeforeProxy(t *testing.T) {
	h := &Handlers{agentClient: newMockAgentClient()}
	for _, headers := range []http.Header{{"Authorization": []string{"Bearer fixture", ""}}, {"Authorization": []string{"Bearer fixture"}, "X-Agent-Identity-Token": []string{""}}, {"Authorization": []string{""}, "X-Agent-Identity-Token": []string{"fixture"}}} {
		r := httptest.NewRequest(http.MethodPost, "/runs", strings.NewReader(`{bad`))
		r.Header = headers
		w := httptest.NewRecorder()
		h.CreateRun(w, r)
		if w.Code != 401 {
			t.Fatal("ambiguous/blank proof masked")
		}
	}
}

func TestAuth01ClientMissingProofRejectsWithoutAnonymousRetry(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer srv.Close()
	_, err := newTestClient(t, srv).CreateRun(context.Background(), &CreateRunRequest{TaskID: "fixture"})
	if err == nil || calls != 0 {
		t.Fatal("anonymous caller reached network")
	}
}

func TestAuth01RealExecutorMissingCallerRejectsBeforeLocalOrTaskEffects(t *testing.T) {
	e := &Executor{agentClient: &AgentManagerClient{}}
	_, err := e.Execute(context.Background(), "fixture", "fixture", "fixture")
	if err == nil {
		t.Fatal("unattended executor accepted missing caller")
	}
}
