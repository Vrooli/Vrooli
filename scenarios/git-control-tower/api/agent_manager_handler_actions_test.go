package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/discovery"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type actionAgentManagerStub struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	continued *ampb.ContinueRunResponse
	approved  *apipb.ApproveRunResponse
	rejected  *apipb.RejectRunResponse
	stopped   *apipb.StopRunResponse
	err       error
}
func (s *actionAgentManagerStub) ContinueRun(context.Context, *connect.Request[ampb.ContinueRunRequest]) (*connect.Response[ampb.ContinueRunResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.continued), nil
}
func (s *actionAgentManagerStub) ApproveRun(context.Context, *connect.Request[apipb.ApproveRunRequest]) (*connect.Response[apipb.ApproveRunResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.approved), nil
}
func (s *actionAgentManagerStub) RejectRun(context.Context, *connect.Request[apipb.RejectRunRequest]) (*connect.Response[apipb.RejectRunResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.rejected), nil
}
func (s *actionAgentManagerStub) StopRun(context.Context, *connect.Request[apipb.StopRunRequest]) (*connect.Response[apipb.StopRunResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.stopped), nil
}

func actionTestServer(t *testing.T, status CapabilityStatus, stub *actionAgentManagerStub) *Server {
	t.Helper()
	path, handler := apiconnect.NewAgentManagerServiceHandler(stub)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	upstream := httptest.NewServer(mux)
	t.Cleanup(upstream.Close)
	client := &AgentManagerClient{BaseClient: BaseClient{httpClient: upstream.Client(), resolver: discovery.NewStaticResolver(upstream.URL), serviceName: "agent-manager"}}
	registry := NewCapabilityRegistry([]CapabilityDef{{ID: "agent-manager"}}, map[string]StatusChecker{"agent-manager": &FakeStatusChecker{Status: status}}, time.Minute)
	return &Server{git: NewFakeGitRunner(), agentManagerClient: client, capabilities: registry}
}

func actionRequest(t *testing.T, s *Server, method, path, body, id string, call func(http.ResponseWriter, *http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r = mux.SetURLVars(r, map[string]string{"id": id})
	w := httptest.NewRecorder()
	call(w, r)
	return w
}

func TestAgentRunActionsRejectUnavailableAndMissingRunID(t *testing.T) {
	s := actionTestServer(t, StatusUnavailable, &actionAgentManagerStub{})
	w := actionRequest(t, s, http.MethodPost, "/continue", `{"message":"next"}`, "run-a", s.handleAgentRunContinue)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("unavailable status=%d", w.Code)
	}
	s = actionTestServer(t, StatusAvailable, &actionAgentManagerStub{})
	w = actionRequest(t, s, http.MethodPost, "/continue", `{"message":"next"}`, " ", s.handleAgentRunContinue)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing id status=%d", w.Code)
	}
}

func TestAgentRunActionResponseProjections(t *testing.T) {
	t.Run("continue", func(t *testing.T) {
		s := actionTestServer(t, StatusAvailable, &actionAgentManagerStub{continued: &ampb.ContinueRunResponse{Success: true, Run: &ampb.Run{Id: "r1", Status: ampb.RunStatus_RUN_STATUS_RUNNING}}})
		w := actionRequest(t, s, http.MethodPost, "/continue", `{"message":"next"}`, "r1", s.handleAgentRunContinue)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"success":true`) || !strings.Contains(w.Body.String(), `"id":"r1"`) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body)
		}
	})
	t.Run("approve", func(t *testing.T) {
		s := actionTestServer(t, StatusAvailable, &actionAgentManagerStub{approved: &apipb.ApproveRunResponse{Result: &ampb.ApproveResult{Success: true, FilesApplied: 2, CommitHash: "abc"}}})
		w := actionRequest(t, s, http.MethodPost, "/approve", `{}`, "r1", s.handleAgentRunApprove)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"filesApplied":2`) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body)
		}
	})
	t.Run("reject and stop", func(t *testing.T) {
		s := actionTestServer(t, StatusAvailable, &actionAgentManagerStub{rejected: &apipb.RejectRunResponse{Status: "rejected"}, stopped: &apipb.StopRunResponse{Status: "stopped"}})
		w := actionRequest(t, s, http.MethodPost, "/reject", `{"reason":"no"}`, "r1", s.handleAgentRunReject)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"status":"rejected"`) {
			t.Fatalf("reject status=%d body=%s", w.Code, w.Body)
		}
		w = actionRequest(t, s, http.MethodPost, "/stop", `{}`, "r1", s.handleAgentRunStop)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"status":"stopped"`) {
			t.Fatalf("stop status=%d body=%s", w.Code, w.Body)
		}
	})
}
