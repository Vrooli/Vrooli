package agentmanager

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestWorkflowServiceCommandResultHandshake(t *testing.T) {
	input, _ := structpb.NewValue(map[string]any{"entity": map[string]any{"kind": "idea", "name": "search", "version": "sha256:v"}, "snapshot": map[string]any{}, "operatorNote": "focus"})
	output, _ := structpb.NewValue(map[string]any{"result": map[string]any{"outcome": "no_questions", "note": "ready", "readiness": map[string]any{"problem_clarity": 3}}})
	execution := &domainpb.WorkflowExecution{Id: "11111111-1111-1111-1111-111111111111", DefinitionDigest: "sha256:def", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, Input: input, Output: output}
	attempt := &domainpb.WorkflowNodeAttempt{NodeId: "workshop", RunId: "22222222-2222-2222-2222-222222222222", ProfileIdentity: "profile:swarm-manager/deep-work"}
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.RequestURI())
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workflow-executions":
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"workflowKey":"swarm-manager/backlog-workshop-round"`) || !strings.Contains(string(body), `"idempotencyKey":"stable-key"`) {
				t.Fatalf("unexpected start body: %s", body)
			}
			// Agent Manager starts workflow execution asynchronously and therefore
			// returns 202 Accepted. This real-transport fixture pins that contract so
			// a successfully-created workflow cannot be reported as a failed start.
			writeWorkflowProto(t, w, &apipb.WorkflowExecutionResponse{Execution: execution}, http.StatusAccepted)
		case strings.HasSuffix(r.URL.Path, "/wait"):
			writeWorkflowProto(t, w, &apipb.WaitWorkflowExecutionResponse{Execution: execution}, http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/result"):
			if r.URL.Query().Get("explicitly_authorized") != "true" {
				t.Fatal("result request was not explicitly authorized")
			}
			writeWorkflowProto(t, w, &apipb.WorkflowExecutionResponse{Execution: execution}, http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/trace"):
			writeWorkflowProto(t, w, &apipb.GetWorkflowExecutionTraceResponse{Execution: execution, Attempts: []*domainpb.WorkflowNodeAttempt{attempt}}, http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	service := NewWorkflowServiceWithClient(client)
	start, err := service.StartWorkshopRound(context.Background(), BacklogWorkshopSnapshot{Kind: "idea", Name: "search", Version: "sha256:v", Title: "Search", OperatorNote: "focus"}, "stable-key")
	if err != nil {
		t.Fatal(err)
	}
	if start.ExecutionID != execution.Id || start.RunID != attempt.RunId {
		t.Fatalf("start = %#v", start)
	}
	completion, err := service.CollectWorkshopRound(context.Background(), execution.Id)
	if err != nil {
		t.Fatal(err)
	}
	if completion.EntityName != "search" || completion.EntityVersion != "sha256:v" || completion.RunID != attempt.RunId || !strings.Contains(string(completion.Result), `"no_questions"`) {
		t.Fatalf("completion = %#v", completion)
	}
	// No per-start reconcile: registration is agent-manager-startup-owned. The
	// handshake is start + trace, then wait + result + trace on collect.
	if len(calls) != 5 {
		t.Fatalf("calls = %v", calls)
	}
}

func TestWorkflowServiceRejectsCanceledCompletion(t *testing.T) {
	execution := &domainpb.WorkflowExecution{Id: "11111111-1111-1111-1111-111111111111", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/wait"):
			writeWorkflowProto(t, w, &apipb.WaitWorkflowExecutionResponse{Execution: execution}, http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/result"):
			writeWorkflowProto(t, w, &apipb.WorkflowExecutionResponse{Execution: execution}, http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	_, err := NewWorkflowServiceWithClient(client).CollectWorkshopRound(context.Background(), execution.Id)
	if err == nil || !strings.Contains(err.Error(), "not successfully terminal") {
		t.Fatalf("canceled completion error = %v", err)
	}
}

func TestPhasedPlanTerminalStatusesIncludeBlockedAndAbstained(t *testing.T) {
	for _, status := range []domainpb.WorkflowExecutionStatus{
		domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BLOCKED,
		domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_ABSTAINED,
	} {
		if !terminalWorkflowStatus(status) {
			t.Fatalf("status %s was not terminal", status)
		}
	}
}

func TestPhasedPlanWorkflowCommandResultAndControlSurface(t *testing.T) {
	input, _ := structpb.NewValue(map[string]any{
		"plan":        map[string]any{"reference": "plan-1", "frontierDigest": "sha256:frontier"},
		"consumer":    map[string]any{"executionId": "consumer-1", "entityKind": "execute", "entityName": "bounded-plan", "entityVersion": "sha256:entity"},
		"constraints": map[string]any{"maxSlices": 6.0, "writeScope": []any{"scenarios/swarm-manager/**"}},
	})
	output, _ := structpb.NewValue(map[string]any{"result": map[string]any{"outcome": "complete", "summary": "done"}})
	execution := &domainpb.WorkflowExecution{
		Id: "33333333-3333-3333-3333-333333333333", DefinitionDigest: "sha256:def", Input: input, Output: output,
		Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED,
	}
	attempts := []*domainpb.WorkflowNodeAttempt{
		{NodeId: "slice", Ordinal: 1, RunId: "run-slice-1", ProfileIdentity: "swarm-manager/deep-work"},
		{NodeId: "review_handoff", Ordinal: 1, RunId: "run-review-1", ProfileIdentity: "swarm-manager/analysis"},
	}
	var signalBody, cancelBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workflow-executions":
			body, _ := io.ReadAll(r.Body)
			text := string(body)
			for _, want := range []string{`"workflowKey":"swarm-manager/phased-plan-drain"`, `"idempotencyKey":"stable-drain"`, `"frontierDigest":"sha256:frontier"`, `"executionId":"consumer-1"`} {
				if !strings.Contains(text, want) {
					t.Fatalf("start body missing %s: %s", want, text)
				}
			}
			writeWorkflowProto(t, w, &apipb.WorkflowExecutionResponse{Execution: execution}, http.StatusAccepted)
		case strings.HasSuffix(r.URL.Path, "/wait"):
			writeWorkflowProto(t, w, &apipb.WaitWorkflowExecutionResponse{Execution: execution}, http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/result"):
			if r.URL.Query().Get("explicitly_authorized") != "true" {
				t.Fatal("result request was not explicitly authorized")
			}
			writeWorkflowProto(t, w, &apipb.WorkflowExecutionResponse{Execution: execution}, http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/trace"):
			writeWorkflowProto(t, w, &apipb.GetWorkflowExecutionTraceResponse{Execution: execution, Attempts: attempts}, http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/signals"):
			body, _ := io.ReadAll(r.Body)
			signalBody = string(body)
			writeWorkflowProto(t, w, &apipb.WorkflowExecutionOperationResponse{Execution: execution}, http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/cancel"):
			body, _ := io.ReadAll(r.Body)
			cancelBody = string(body)
			writeWorkflowProto(t, w, &apipb.WorkflowExecutionOperationResponse{Execution: execution}, http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	service := NewWorkflowServiceWithClient(NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client()))
	start, err := service.StartPhasedPlan(context.Background(), PhasedPlanSnapshot{
		PlanReference: "plan-1", FrontierDigest: "sha256:frontier", ExecutionID: "consumer-1",
		EntityKind: "execute", EntityName: "bounded-plan", EntityVersion: "sha256:entity", MaxSlices: 6,
		WriteScope: []string{"scenarios/swarm-manager/**"},
	}, "stable-drain")
	if err != nil {
		t.Fatal(err)
	}
	if start.ExecutionID != execution.Id || start.RunID != "run-slice-1" {
		t.Fatalf("start = %#v", start)
	}
	completion, err := service.CollectPhasedPlan(context.Background(), execution.Id)
	if err != nil {
		t.Fatal(err)
	}
	if completion.ConsumerID != "consumer-1" || completion.FrontierDigest != "sha256:frontier" || len(completion.Attempts) != 2 || !strings.Contains(string(completion.Result), `"complete"`) {
		t.Fatalf("completion = %#v", completion)
	}
	if err := service.SignalPhasedPlanApproval(context.Background(), execution.Id, "consumer-1", "alice", "approve-1"); err != nil {
		t.Fatal(err)
	}
	if err := service.CancelPhasedPlan(context.Background(), execution.Id, "operator canceled"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"signal":"slice_approved"`, `"idempotencyKey":"approve-1"`, `"executionId":"consumer-1"`} {
		if !strings.Contains(signalBody, want) {
			t.Fatalf("signal body missing %s: %s", want, signalBody)
		}
	}
	for _, want := range []string{`"idempotencyKey":"cancel-` + execution.Id + `"`, `"reason":"operator canceled"`} {
		if !strings.Contains(cancelBody, want) {
			t.Fatalf("cancel body missing %s: %s", want, cancelBody)
		}
	}
}

func writeWorkflowProto(t *testing.T, w http.ResponseWriter, message proto.Message, status int) {
	t.Helper()
	data, err := protoJSONMarshal.Marshal(message)
	if err != nil {
		t.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}
