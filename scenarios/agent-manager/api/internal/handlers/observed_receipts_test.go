package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-manager/internal/runreport"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/eventbus"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventspb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestAttachObservedReceiptsMarksUnavailableWithoutEventsClient(t *testing.T) {
	h := &Handler{receipts: eventbus.Client{}}
	result := &domainpb.RunResult{}
	h.attachObservedReceipts(context.Background(), "run-1", result)
	if result.Observations == nil || result.Observations.State != domainpb.ReceiptObservationState_RECEIPT_OBSERVATION_STATE_UNAVAILABLE {
		t.Fatalf("observations = %+v", result.Observations)
	}
	// Nil results are explicitly safe for callers that have no declared output.
	h.attachObservedReceipts(context.Background(), "run-1", nil)
}

func TestGetObservedReceiptsRejectsInvalidRunIDBeforeServiceAccess(t *testing.T) {
	h := &Handler{receipts: eventbus.Client{}}
	req := httptest.NewRequest(http.MethodGet, "/runs/not-a-uuid/observed-receipts", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "not-a-uuid"})
	rw := httptest.NewRecorder()
	h.GetObservedReceipts(rw, req)
	if rw.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rw.Code, rw.Body.String())
	}
}

func TestWorkflowObservedReceiptsMarksUnavailableWithoutEventsClient(t *testing.T) {
	h := &Handler{receipts: eventbus.Client{}}
	observations := h.workflowObservedReceipts(context.Background(), []string{"run-1", "run-2"})
	if observations.State != domainpb.ReceiptObservationState_RECEIPT_OBSERVATION_STATE_UNAVAILABLE || observations.Reason == "" {
		t.Fatalf("observations = %+v", observations)
	}
}

func TestObservedReceiptsExtractsOnlyVerifiedCorrelatedReceiptEvents(t *testing.T) {
	data, err := anypb.New(&eventspb.ReceiptData{Outcome: "accepted", StatusCode: 201})
	if err != nil {
		t.Fatal(err)
	}
	valid, err := protojson.Marshal(&eventspb.EventEnvelope{
		EventId: "receipt-1", EventType: "vrooli.events.receipt.v1", Data: data,
		Attribution: &eventspb.EventAttribution{Verified: true}, Target: &eventspb.EventTarget{Scenario: "agent-manager", Operation: "create-run"},
		Correlation: &eventspb.EventCorrelation{AgentRunId: "run-1", WorkflowExecutionId: "workflow-1", WorkflowNodeId: "node-1", Attempt: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("agent_run_id") != "run-1" || r.URL.Query().Get("limit") != "100" {
			t.Fatalf("unexpected receipt query: %s", r.URL.String())
		}
		_ = json.NewEncoder(w).Encode([]json.RawMessage{valid, []byte(`{"eventType":"ignored"}`)})
	}))
	t.Cleanup(server.Close)
	h := &Handler{receipts: eventbus.Client{BaseURL: server.URL, HTTPClient: server.Client()}}
	result := &domainpb.RunResult{}
	h.attachObservedReceipts(context.Background(), "run-1", result)
	if result.Observations.GetState() != domainpb.ReceiptObservationState_RECEIPT_OBSERVATION_STATE_AVAILABLE || len(result.Observations.GetReceipts()) != 1 {
		t.Fatalf("observations=%+v", result.Observations)
	}
	receipt := result.Observations.GetReceipts()[0]
	if receipt.GetEventId() != "receipt-1" || receipt.GetAgentRunId() != "run-1" || !receipt.GetAttributionVerified() || receipt.GetStatusCode() != 201 {
		t.Fatalf("receipt=%+v", receipt)
	}
	workflow := h.workflowObservedReceipts(context.Background(), []string{"run-1"})
	if workflow.GetState() != domainpb.ReceiptObservationState_RECEIPT_OBSERVATION_STATE_AVAILABLE || len(workflow.GetReceipts()) != 1 {
		t.Fatalf("workflow observations=%+v", workflow)
	}
}

func TestObservedReceiptsMarksDegradedWhenEventQueryFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "down", http.StatusServiceUnavailable) }))
	t.Cleanup(server.Close)
	h := &Handler{receipts: eventbus.Client{BaseURL: server.URL, HTTPClient: server.Client()}}
	result := &domainpb.RunResult{}
	h.attachObservedReceipts(context.Background(), "run-1", result)
	if result.Observations.GetState() != domainpb.ReceiptObservationState_RECEIPT_OBSERVATION_STATE_DEGRADED {
		t.Fatalf("observations=%+v", result.Observations)
	}
}

func TestGetObservedReceiptsReturnsConfiguredObservationState(t *testing.T) {
	h, router := setupTestHandler(t)
	profileBody := encodeProtoJSON(t, &apipb.CreateProfileRequest{Profile: &domainpb.AgentProfile{Name: "receipt-profile", ProfileKey: "receipt-profile", RoleRef: "code.default"}})
	profileRR := httptest.NewRecorder()
	router.ServeHTTP(profileRR, httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(profileBody)))
	var profile apipb.CreateProfileResponse
	decodeProtoJSON(t, profileRR.Body.Bytes(), &profile)
	taskBody := encodeProtoJSON(t, &apipb.CreateTaskRequest{Task: &domainpb.Task{Title: "receipt task", ScopePath: "src/receipts"}})
	taskRR := httptest.NewRecorder()
	router.ServeHTTP(taskRR, httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(taskBody)))
	var task apipb.CreateTaskResponse
	decodeProtoJSON(t, taskRR.Body.Bytes(), &task)
	profileID := profile.Profile.GetId()
	runBody := encodeProtoJSON(t, &apipb.CreateRunRequest{TaskId: task.Task.GetId(), AgentProfileId: &profileID})
	runRR := httptest.NewRecorder()
	router.ServeHTTP(runRR, httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(runBody)))
	var run apipb.CreateRunResponse
	decodeProtoJSON(t, runRR.Body.Bytes(), &run)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("agent_run_id") != run.Run.GetId() || r.URL.Query().Get("limit") != "2" {
			t.Fatalf("query=%s", r.URL.String())
		}
		_ = json.NewEncoder(w).Encode([]json.RawMessage{})
	}))
	t.Cleanup(server.Close)
	h.receipts = eventbus.Client{BaseURL: server.URL, HTTPClient: server.Client()}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+run.Run.GetId()+"/observed-receipts?limit=2", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil || response["status"] != "unobserved" {
		t.Fatalf("response=%v err=%v", response, err)
	}
	invalid := httptest.NewRecorder()
	router.ServeHTTP(invalid, httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+run.Run.GetId()+"/observed-receipts?limit=101", nil))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid limit status=%d body=%s", invalid.Code, invalid.Body.String())
	}
}

func TestGetObservedReceiptsExplainsEmptyRuntimeState(t *testing.T) {
	h, router := setupTestHandler(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]json.RawMessage{})
	}))
	t.Cleanup(server.Close)
	h.receipts = eventbus.Client{BaseURL: server.URL, HTTPClient: server.Client()}
	h.receiptAvailability = func(context.Context) runreport.Availability {
		return runreport.Availability{State: runreport.AvailabilityPolicyAbsent, Reason: "receipt runtime for test-genie is connected but has no capture policies"}
	}
	// A malformed identifier must still be rejected before either reader runs.
	// Use a directly registered run through the existing handler fixture.
	profileBody := encodeProtoJSON(t, &apipb.CreateProfileRequest{Profile: &domainpb.AgentProfile{Name: "runtime-state-profile", ProfileKey: "runtime-state-profile", RoleRef: "code.default"}})
	profileRR := httptest.NewRecorder()
	router.ServeHTTP(profileRR, httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(profileBody)))
	var profile apipb.CreateProfileResponse
	decodeProtoJSON(t, profileRR.Body.Bytes(), &profile)
	taskBody := encodeProtoJSON(t, &apipb.CreateTaskRequest{Task: &domainpb.Task{Title: "runtime state task", ScopePath: "src/runtime"}})
	taskRR := httptest.NewRecorder()
	router.ServeHTTP(taskRR, httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(taskBody)))
	var task apipb.CreateTaskResponse
	decodeProtoJSON(t, taskRR.Body.Bytes(), &task)
	profileID := profile.Profile.GetId()
	runBody := encodeProtoJSON(t, &apipb.CreateRunRequest{TaskId: task.Task.GetId(), AgentProfileId: &profileID})
	runRR := httptest.NewRecorder()
	router.ServeHTTP(runRR, httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(runBody)))
	var run apipb.CreateRunResponse
	decodeProtoJSON(t, runRR.Body.Bytes(), &run)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+run.Run.GetId()+"/observed-receipts", nil))
	var response map[string]any
	if rr.Code != http.StatusOK || json.Unmarshal(rr.Body.Bytes(), &response) != nil || response["status"] != "policy_absent" || response["reason"] == "" {
		t.Fatalf("status=%d response=%v", rr.Code, response)
	}
}

func TestGetObservedReceiptsExcludesUnverifiedOrUncorrelatedEvidence(t *testing.T) {
	runID := "01010101-0101-0101-0101-010101010101"
	valid, err := protojson.Marshal(&eventspb.EventEnvelope{EventId: "verified", EventType: eventbus.ReceiptEventType, Attribution: &eventspb.EventAttribution{Verified: true}, Correlation: &eventspb.EventCorrelation{AgentRunId: runID}})
	if err != nil {
		t.Fatal(err)
	}
	unverified, err := protojson.Marshal(&eventspb.EventEnvelope{EventId: "unverified", EventType: eventbus.ReceiptEventType, Attribution: &eventspb.EventAttribution{Verified: false}, Correlation: &eventspb.EventCorrelation{AgentRunId: runID}})
	if err != nil {
		t.Fatal(err)
	}
	wrongRun, err := protojson.Marshal(&eventspb.EventEnvelope{EventId: "wrong-run", EventType: eventbus.ReceiptEventType, Attribution: &eventspb.EventAttribution{Verified: true}, Correlation: &eventspb.EventCorrelation{AgentRunId: "02020202-0202-0202-0202-020202020202"}})
	if err != nil {
		t.Fatal(err)
	}
	got := verifiedReceiptObservations([]json.RawMessage{valid, unverified, wrongRun}, runID)
	if len(got) != 1 {
		t.Fatalf("verified receipts = %s", got)
	}
	var envelope eventspb.EventEnvelope
	if err := protojson.Unmarshal(got[0], &envelope); err != nil || envelope.EventId != "verified" {
		t.Fatalf("receipt=%s err=%v", got[0], err)
	}
}
