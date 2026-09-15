package agentsessions

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"swarm-manager/internal/agentmanager"

	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestHandlerLifecycleEndpointsUseProtoJSONContracts(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	spawner := &fakeSessionSpawner{runState: agentmanager.RunState{
		Status:  "complete",
		Summary: "Session final handoff.",
	}}
	svc := newTestService(t, spawner)
	svc.contextResolver = fakeContextResolver{}
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	createBody := marshalAgentSessionProto(t, &apipb.CreateAgentSessionRequest{
		Kind:  string(KindSwarmOperations),
		Title: "Manage Swarm operations",
	})
	createRec := serveAgentSessionRequest(router, http.MethodPost, "/api/v1/agent-sessions", createBody)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", createRec.Code, createRec.Body.String())
	}
	var createResp apipb.CreateAgentSessionResponse
	unmarshalAgentSessionProto(t, createRec, &createResp)
	sessionID := createResp.GetSession().GetId()
	if sessionID == "" {
		t.Fatalf("created session id is empty: %+v", createResp.GetSession())
	}
	if createResp.GetSession().GetRunId() != "" || createResp.GetSession().GetStatus() != string(StatusDraft) || createResp.GetSession().GetKind() != string(KindSwarmOperations) {
		t.Fatalf("created session should be draft without run: %+v", createResp.GetSession())
	}
	attachBody := marshalAgentSessionProto(t, &apipb.AttachAgentSessionContextRequest{ContextRefs: []*apipb.AgentSessionContextRef{
		{Type: string(ContextBacklogItem), Ref: "chore/first"},
		{Type: string(ContextGoal), Ref: "delivery"},
	}})
	attachRec := serveAgentSessionRequest(router, http.MethodPost, "/api/v1/agent-sessions/"+sessionID+"/context", attachBody)
	if attachRec.Code != http.StatusOK {
		t.Fatalf("attach status = %d, body = %s", attachRec.Code, attachRec.Body.String())
	}
	var attachResp apipb.AttachAgentSessionContextResponse
	unmarshalAgentSessionProto(t, attachRec, &attachResp)
	refs := attachResp.GetSession().GetStagedContextRefs()
	if len(refs) != 2 || refs[0].GetType() != string(ContextBacklogItem) || refs[1].GetRef() != "delivery" {
		t.Fatalf("staged refs = %+v", refs)
	}

	listRec := serveAgentSessionRequest(router, http.MethodGet, "/api/v1/agent-sessions?kind=swarm_operations&active_only=true&limit=10", nil)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}
	var listResp apipb.ListAgentSessionsResponse
	unmarshalAgentSessionProto(t, listRec, &listResp)
	if len(listResp.GetSessions()) != 1 || listResp.GetSessions()[0].GetId() != sessionID {
		t.Fatalf("list response = %+v", listResp.GetSessions())
	}

	startBody := marshalAgentSessionProto(t, &apipb.StartAgentSessionRequest{
		Message: "Draft an operating mode.",
	})
	startRec := serveAgentSessionRequest(router, http.MethodPost, "/api/v1/agent-sessions/"+sessionID+"/start", startBody)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start status = %d, body = %s", startRec.Code, startRec.Body.String())
	}
	var startResp apipb.StartAgentSessionResponse
	unmarshalAgentSessionProto(t, startRec, &startResp)
	if startResp.GetSession().GetRunId() != "run-1" || startResp.GetSession().GetStatus() != string(StatusRunning) {
		t.Fatalf("start response = %+v", startResp.GetSession())
	}

	continueBody := marshalAgentSessionProto(t, &apipb.ContinueAgentSessionRequest{
		Message:       "Refine it.",
		AttachmentIds: []string{"att-1"},
	})
	continueRec := serveAgentSessionRequest(router, http.MethodPost, "/api/v1/agent-sessions/"+sessionID+"/continue", continueBody)
	if continueRec.Code != http.StatusOK {
		t.Fatalf("continue status = %d, body = %s", continueRec.Code, continueRec.Body.String())
	}
	var continueResp apipb.ContinueAgentSessionResponse
	unmarshalAgentSessionProto(t, continueRec, &continueResp)
	if got := len(continueResp.GetSession().GetMessages()); got != 2 {
		t.Fatalf("continue message count = %d, want 2", got)
	}

	refreshRec := serveAgentSessionRequest(router, http.MethodPost, "/api/v1/agent-sessions/"+sessionID+"/refresh", nil)
	if refreshRec.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, body = %s", refreshRec.Code, refreshRec.Body.String())
	}
	var refreshResp apipb.RefreshAgentSessionResponse
	unmarshalAgentSessionProto(t, refreshRec, &refreshResp)
	if refreshResp.GetSession().GetStatus() != string(StatusComplete) {
		t.Fatalf("refresh status = %q, want complete", refreshResp.GetSession().GetStatus())
	}
	if messages := refreshResp.GetSession().GetMessages(); len(messages) != 3 || messages[2].GetContent() != "Session final handoff." {
		t.Fatalf("refresh messages = %+v", messages)
	}

	if _, err := svc.AttachArtifact(context.Background(), Artifact{
		SessionID:    sessionID,
		ArtifactType: ArtifactMilestone,
		Action:       ArtifactActionCreated,
		EntityRef:    "mode-authoring",
		Title:        "Mode Authoring",
	}); err != nil {
		t.Fatalf("AttachArtifact() error = %v", err)
	}
	artifactsRec := serveAgentSessionRequest(router, http.MethodGet, "/api/v1/agent-sessions/"+sessionID+"/artifacts", nil)
	if artifactsRec.Code != http.StatusOK {
		t.Fatalf("artifacts status = %d, body = %s", artifactsRec.Code, artifactsRec.Body.String())
	}
	var artifactsResp apipb.ListAgentSessionArtifactsResponse
	unmarshalAgentSessionProto(t, artifactsRec, &artifactsResp)
	if len(artifactsResp.GetArtifacts()) != 1 || artifactsResp.GetArtifacts()[0].GetEntityRef() != "mode-authoring" {
		t.Fatalf("artifacts response = %+v", artifactsResp.GetArtifacts())
	}

	byEntityRec := serveAgentSessionRequest(router, http.MethodGet, "/api/v1/artifacts/by-entity?artifact_type=milestone&entity_ref=mode-authoring", nil)
	if byEntityRec.Code != http.StatusOK {
		t.Fatalf("by entity status = %d, body = %s", byEntityRec.Code, byEntityRec.Body.String())
	}
	var byEntityResp apipb.GetArtifactsByEntityResponse
	unmarshalAgentSessionProto(t, byEntityRec, &byEntityResp)
	if len(byEntityResp.GetArtifacts()) != 1 || byEntityResp.GetArtifacts()[0].GetSessionId() != sessionID {
		t.Fatalf("by entity response = %+v", byEntityResp.GetArtifacts())
	}

	cancelRec := serveAgentSessionRequest(router, http.MethodPost, "/api/v1/agent-sessions/"+sessionID+"/cancel", nil)
	if cancelRec.Code != http.StatusOK {
		t.Fatalf("cancel status = %d, body = %s", cancelRec.Code, cancelRec.Body.String())
	}
	var cancelResp apipb.CancelAgentSessionResponse
	unmarshalAgentSessionProto(t, cancelRec, &cancelResp)
	if cancelResp.GetSession().GetStatus() != string(StatusCanceled) {
		t.Fatalf("cancel status = %q, want canceled", cancelResp.GetSession().GetStatus())
	}

	applyRec := serveAgentSessionRequest(router, http.MethodPost, "/api/v1/agent-sessions/"+sessionID+"/proposals/prop-1/apply", nil)
	if applyRec.Code != http.StatusNotFound {
		t.Fatalf("apply status = %d, want 404; body = %s", applyRec.Code, applyRec.Body.String())
	}

	deleteRec := serveAgentSessionRequest(router, http.MethodDelete, "/api/v1/agent-sessions/"+sessionID, nil)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", deleteRec.Code, deleteRec.Body.String())
	}
	var deleteResp apipb.DeleteAgentSessionResponse
	unmarshalAgentSessionProto(t, deleteRec, &deleteResp)
	if deleteResp.GetSessionId() != sessionID {
		t.Fatalf("delete session_id = %q, want %q", deleteResp.GetSessionId(), sessionID)
	}

	getDeletedRec := serveAgentSessionRequest(router, http.MethodGet, "/api/v1/agent-sessions/"+sessionID, nil)
	if getDeletedRec.Code != http.StatusNotFound {
		t.Fatalf("get deleted status = %d, want 404; body = %s", getDeletedRec.Code, getDeletedRec.Body.String())
	}
}

func TestHandlerDeleteStopsActiveRunAndRejectsInvalidIDs(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	spawner := &fakeSessionSpawner{}
	svc := newTestService(t, spawner)
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)

	createBody := marshalAgentSessionProto(t, &apipb.CreateAgentSessionRequest{
		Kind:  string(KindMetaOrchestration),
		Title: "Plan work",
	})
	createRec := serveAgentSessionRequest(router, http.MethodPost, "/api/v1/agent-sessions", createBody)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", createRec.Code, createRec.Body.String())
	}
	var createResp apipb.CreateAgentSessionResponse
	unmarshalAgentSessionProto(t, createRec, &createResp)
	sessionID := createResp.GetSession().GetId()
	startBody := marshalAgentSessionProto(t, &apipb.StartAgentSessionRequest{Message: "Plan the next milestone."})
	startRec := serveAgentSessionRequest(router, http.MethodPost, "/api/v1/agent-sessions/"+sessionID+"/start", startBody)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start status = %d, body = %s", startRec.Code, startRec.Body.String())
	}
	var startResp apipb.StartAgentSessionResponse
	unmarshalAgentSessionProto(t, startRec, &startResp)
	runID := startResp.GetSession().GetRunId()

	deleteRec := serveAgentSessionRequest(router, http.MethodDelete, "/api/v1/agent-sessions/"+sessionID, nil)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", deleteRec.Code, deleteRec.Body.String())
	}
	if spawner.stoppedRunID != runID {
		t.Fatalf("stopped run = %q, want %q", spawner.stoppedRunID, runID)
	}

	missingRec := serveAgentSessionRequest(router, http.MethodDelete, "/api/v1/agent-sessions/sess_missing", nil)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("missing delete status = %d, want 404; body = %s", missingRec.Code, missingRec.Body.String())
	}

	invalidRec := serveAgentSessionRequest(router, http.MethodDelete, "/api/v1/agent-sessions/bad_id", nil)
	if invalidRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid delete status = %d, want 400; body = %s", invalidRec.Code, invalidRec.Body.String())
	}
}

func TestHandlerRejectsInvalidCreateRequest(t *testing.T) {
	router := mux.NewRouter()
	NewHandler(newTestService(t, &fakeSessionSpawner{})).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-sessions", strings.NewReader(`{"kind":"custom","title":"Bad"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func serveAgentSessionRequest(router *mux.Router, method, target string, body []byte) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func marshalAgentSessionProto(t *testing.T, msg proto.Message) []byte {
	t.Helper()
	payload, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal proto: %v", err)
	}
	return payload
}

func unmarshalAgentSessionProto(t *testing.T, rec *httptest.ResponseRecorder, msg proto.Message) {
	t.Helper()
	if err := protojson.Unmarshal(rec.Body.Bytes(), msg); err != nil {
		t.Fatalf("unmarshal response %q: %v", rec.Body.String(), err)
	}
}
