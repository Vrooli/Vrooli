package agentmanager

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestWorkflowServiceProgressReadsLiveTrace(t *testing.T) {
	now := timestamppb.New(time.Now().UTC())
	execution := &domainpb.WorkflowExecution{Id: "workflow-1", CurrentNodeId: "slice_review", UpdatedAt: now, BudgetUsage: &domainpb.WorkflowBudgetUsage{Turns: 4, Tokens: 900, CostUsd: 0.12}, EdgeTraversals: map[string]int32{"review": 2}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/workflow-executions/workflow-1/trace" {
			http.NotFound(w, r)
			return
		}
		writeWorkflowProto(t, w, &apipb.GetWorkflowExecutionTraceResponse{Execution: execution, Journal: []*domainpb.WorkflowJournalEntry{{NodeId: "slice"}, {NodeId: "slice"}, {NodeId: "review"}}}, http.StatusOK)
	}))
	defer server.Close()
	service := NewWorkflowServiceWithClient(NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client()))
	progress, err := service.GetWorkflowProgress(context.Background(), "workflow-1")
	if err != nil {
		t.Fatal(err)
	}
	if progress.CurrentNode != "slice_review" || progress.SliceCount != 2 || progress.Turns != 4 || progress.Tokens != 900 || progress.CostUSD != 0.12 || progress.EdgeTraversals["review"] != 2 || progress.UpdatedAt == "" {
		t.Fatalf("progress = %#v", progress)
	}
}

func TestWorkflowServiceCommandResultHandshake(t *testing.T) {
	input, _ := structpb.NewValue(map[string]any{"entity": map[string]any{"kind": "idea", "name": "search", "version": "sha256:v"}, "snapshot": map[string]any{}, "operatorNote": "focus"})
	output, _ := structpb.NewValue(map[string]any{"result": map[string]any{"outcome": "no_questions", "note": "ready", "readiness": map[string]any{"problem_clarity": 3}}})
	execution := &domainpb.WorkflowExecution{Id: "11111111-1111-1111-1111-111111111111", DefinitionDigest: "sha256:def", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, Input: input, Output: output}
	attempt := &domainpb.WorkflowNodeAttempt{NodeId: "review", RunId: "22222222-2222-2222-2222-222222222222", ProfileIdentity: "profile:swarm-manager/deep-work"}
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.RequestURI())
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workflow-executions":
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"workflowKey":"swarm-manager/plan-workshop-review"`) || !strings.Contains(string(body), `"idempotencyKey":"stable-key"`) {
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
	start, err := service.StartWorkflow(context.Background(), Invocation{Owner: "swarm-manager", WorkflowKey: "swarm-manager/plan-workshop-review", Input: input, IdempotencyKey: "stable-key", FirstRunNodeID: "review"})
	if err != nil {
		t.Fatal(err)
	}
	if start.ExecutionID != execution.Id || start.RunID != attempt.RunId {
		t.Fatalf("start = %#v", start)
	}
	completion, err := service.CollectWorkflow(context.Background(), execution.Id)
	if err != nil {
		t.Fatal(err)
	}
	if completion.Input == nil || completion.Output == nil || len(completion.Attempts) != 1 || !strings.Contains(completion.Output.String(), "no_questions") {
		t.Fatalf("completion = %#v", completion)
	}
	// No per-start reconcile: registration is agent-manager-startup-owned. The
	// handshake is start + trace, then wait + result + trace on collect.
	if len(calls) != 5 {
		t.Fatalf("calls = %v", calls)
	}
}

func TestWorkflowServiceCollectsCanceledCompletion(t *testing.T) {
	execution := &domainpb.WorkflowExecution{Id: "11111111-1111-1111-1111-111111111111", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/wait"):
			writeWorkflowProto(t, w, &apipb.WaitWorkflowExecutionResponse{Execution: execution}, http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/result"):
			writeWorkflowProto(t, w, &apipb.WorkflowExecutionResponse{Execution: execution}, http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/trace"):
			writeWorkflowProto(t, w, &apipb.GetWorkflowExecutionTraceResponse{Execution: execution}, http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	completion, err := NewWorkflowServiceWithClient(client).CollectWorkflow(context.Background(), execution.Id)
	if err != nil || completion.Status != domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED {
		t.Fatalf("canceled completion = %#v, err=%v", completion, err)
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

func TestGenericWorkflowCommandResultAndControlSurface(t *testing.T) {
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
	start, err := service.StartWorkflow(context.Background(), Invocation{
		Owner: "swarm-manager", WorkflowKey: "swarm-manager/phased-plan-drain", Input: input,
		IdempotencyKey: "stable-drain", FirstRunNodeID: "slice",
	})
	if err != nil {
		t.Fatal(err)
	}
	if start.ExecutionID != execution.Id || start.RunID != "run-slice-1" {
		t.Fatalf("start = %#v", start)
	}
	completion, err := service.CollectWorkflow(context.Background(), execution.Id)
	if err != nil {
		t.Fatal(err)
	}
	if completion.ExecutionID != execution.Id || completion.DefinitionDigest != "sha256:def" || len(completion.Attempts) != 2 || completion.Output == nil {
		t.Fatalf("completion = %#v", completion)
	}
	payload, err := structpb.NewValue(map[string]any{"executionId": "consumer-1", "actor": "alice"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.SignalWorkflow(context.Background(), execution.Id, "slice_approved", payload, "approve-1"); err != nil {
		t.Fatal(err)
	}
	if err := service.CancelWorkflow(context.Background(), execution.Id, "cancel-"+execution.Id, "operator canceled"); err != nil {
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
