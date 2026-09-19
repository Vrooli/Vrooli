package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/operations"
	"scenario-to-cloud/persistence"
)

// operationsTestServer builds the live router over an in-memory repository
// with a gated runner as the operation owner, so the REST contract is
// exercised end to end without SSH.
func operationsTestServer(t *testing.T, name string, runner operations.Runner) (*Server, *persistence.Repository, *operations.Service) {
	t.Helper()
	srv, repo := newIdentityTestServer(t, name)
	cfg := operations.Config{WorkerID: "test-owner", Workers: 1, LeaseTTL: 200 * time.Millisecond, HeartbeatInterval: 50 * time.Millisecond, ReconcileInterval: time.Hour, ObserverTimeout: 5 * time.Second, Logger: srv.log}
	svc := operations.NewService(cfg, repo, runner, nil, nil)
	svc.Start()
	t.Cleanup(svc.Stop)
	srv.operations = svc
	return srv, repo, svc
}

func seedOperation(t *testing.T, repo *persistence.Repository, id string) *domain.CloudOperation {
	t.Helper()
	seedDeployment(t, repo, "dep-1", "demo-app", "production", "203.0.113.10", "demo.example")
	plan := &execplan.Plan{
		SchemaVersion: execplan.SchemaVersion, DeploymentID: "dep-1", ScenarioID: "demo-app", Environment: "production", Scope: execplan.ScopeFull, Outcome: execplan.OutcomeApply,
		Actions: []execplan.Action{{ID: "release.stage", OwnerOperation: "release.stage", Retry: execplan.RetrySafeReplay, CancelPoint: true}},
	}
	raw, _ := json.Marshal(plan)
	digest, _ := plan.SemanticDigest()
	op, err := repo.AdmitOperation(context.Background(), &domain.CloudOperation{ID: id, DeploymentID: "dep-1", RequestKey: "k-" + id, PlanDigest: digest, Plan: raw})
	if err != nil {
		t.Fatal(err)
	}
	return op
}

// gatedRunner executes the single-action plan once its gate opens.
type gatedRunner struct{ gate chan struct{} }

func (g *gatedRunner) Execute(ctx context.Context, ec *operations.ExecutionContext) error {
	for _, action := range ec.Plan.Actions {
		decision, err := ec.Steps.Begin(ctx, action)
		if err != nil {
			return err
		}
		if decision == operations.DecisionSkip {
			continue
		}
		select {
		case <-g.gate:
		case <-ctx.Done():
			return ctx.Err()
		}
		if err := ec.Steps.Commit(ctx, action, operations.StepSucceeded, "ok", nil); err != nil {
			return err
		}
	}
	return nil
}

func decodeStanding(t *testing.T, rec *httptest.ResponseRecorder) operations.Standing {
	t.Helper()
	var st operations.Standing
	if err := json.Unmarshal(rec.Body.Bytes(), &st); err != nil {
		t.Fatalf("decode standing (%d %s): %v", rec.Code, rec.Body.String(), err)
	}
	return st
}

// TestOperationRESTContract [REQ:STC-P0-020] proves the durable wait/status/
// cancel surface over the live router: status by id, a timed-out wait that
// returns still_pending without mutating, listing by deployment, the SSE
// stream keyed by operation id, and a typed 404 for an unknown operation.
func TestOperationRESTContract(t *testing.T) {
	runner := &gatedRunner{gate: make(chan struct{})}
	srv, repo, svc := operationsTestServer(t, "ops-rest", runner)
	op := seedOperation(t, repo, "op-1")

	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/operations/op-1", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("get: %d %s", rec.Code, rec.Body.String())
	}
	st := decodeStanding(t, rec)
	if st.State != operations.Admitted || st.OperationID != op.ID || st.ReattachCommand == "" || st.NextAction == nil {
		t.Fatalf("standing = %+v", st)
	}

	svc.Submit(context.Background(), op.ID, operations.ExecuteOptions{})
	rec = httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/operations/op-1/wait?timeout=1", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("wait: %d %s", rec.Code, rec.Body.String())
	}
	pending := decodeStanding(t, rec)
	if !pending.StillPending || pending.State != operations.Running || pending.RecommendedNextCheckSeconds < 1 || pending.ActiveStep != "release.stage" {
		t.Fatalf("pending standing = %+v", pending)
	}

	rec = httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/deployments/dep-1/operations", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"operation_id":"op-1"`) {
		t.Fatalf("list: %d %s", rec.Code, rec.Body.String())
	}

	// SSE keyed by operation id: open the stream, release the gate, expect the
	// terminal event and a closed stream.
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments/dep-1/progress?operation_id=op-1", nil)
		srv.Router().ServeHTTP(&streamRecorder{ResponseRecorder: httptest.NewRecorder(), w: pw}, req)
	}()
	time.Sleep(50 * time.Millisecond)
	close(runner.gate)
	scanner := bufio.NewScanner(pr)
	sawCompleted := false
	deadline := time.After(5 * time.Second)
	lines := make(chan string)
	go func() {
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()
loop:
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				break loop
			}
			if strings.HasPrefix(line, "event: completed") {
				sawCompleted = true
			}
		case <-deadline:
			t.Fatal("SSE stream did not terminate")
		}
	}
	if !sawCompleted {
		t.Fatal("SSE stream keyed by operation must replay the terminal completed event")
	}

	rec = httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/operations/op-1/wait", nil))
	final := decodeStanding(t, rec)
	if final.State != operations.Succeeded || !final.Terminal || len(final.CompletedSteps) != 1 {
		t.Fatalf("final standing = %+v", final)
	}
	dep, _ := repo.GetDeployment(context.Background(), "dep-1")
	if dep.Fence == 0 {
		t.Fatalf("acquisition must have bumped the deployment fence: %+v", dep)
	}

	rec = httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/operations/nope", nil))
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "operation_not_found") {
		t.Fatalf("unknown: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/deployments/dep-1/progress?operation_id=nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("progress with unknown operation: %d", rec.Code)
	}
}

// TestOperationCancelAndReconcileRoutes [REQ:STC-P0-020] proves cancel over
// REST (202 cancel_requested for a running operation, 409 once terminal) and
// the reconcile route (202 with the acquired ids).
func TestOperationCancelAndReconcileRoutes(t *testing.T) {
	runner := &gatedRunner{gate: make(chan struct{})}
	srv, repo, svc := operationsTestServer(t, "ops-cancel", runner)
	op := seedOperation(t, repo, "op-1")
	svc.Submit(context.Background(), op.ID, operations.ExecuteOptions{})
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cur, _ := repo.GetOperation(context.Background(), op.ID); cur != nil && cur.State == operations.Running && cur.ActiveStepMarker() != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/operations/op-1/cancel", nil))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("cancel: %d %s", rec.Code, rec.Body.String())
	}
	if st := decodeStanding(t, rec); st.State != operations.CancelRequested || !st.CancelRequested {
		t.Fatalf("cancel standing = %+v", st)
	}
	close(runner.gate)
	// The in-flight action commits, then the worker reaches the end of the
	// plan: cancel_requested → verifying → succeeded (no later cancel point).
	final, pending, err := svc.Wait(context.Background(), op.ID, 5*time.Second)
	if err != nil || pending || !final.State.IsTerminal() {
		t.Fatalf("final = %v %v %v", final, pending, err)
	}
	rec = httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/operations/op-1/cancel", nil))
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "operation_conflict") {
		t.Fatalf("terminal cancel: %d %s", rec.Code, rec.Body.String())
	}

	// Reconcile: a second admitted operation with no owner is acquired.
	plan := &execplan.Plan{SchemaVersion: execplan.SchemaVersion, DeploymentID: "dep-1", Scope: execplan.ScopeFull, Outcome: execplan.OutcomeApply}
	raw, _ := json.Marshal(plan)
	digest, _ := plan.SemanticDigest()
	if _, err := repo.AdmitOperation(context.Background(), &domain.CloudOperation{ID: "op-2", DeploymentID: "dep-1", RequestKey: "k2", PlanDigest: digest, Plan: raw}); err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/operations/reconcile", bytes.NewReader(nil)))
	if rec.Code != http.StatusAccepted || !strings.Contains(rec.Body.String(), `"op-2"`) {
		t.Fatalf("reconcile: %d %s", rec.Code, rec.Body.String())
	}
	if final, _, err := svc.Wait(context.Background(), "op-2", 5*time.Second); err != nil || final.State != operations.Succeeded {
		t.Fatalf("reconciled op-2 = %v %v", final, err)
	}
}

// streamRecorder forwards the SSE body to a pipe while the handler runs so
// the test can observe events before the handler returns.
type streamRecorder struct {
	*httptest.ResponseRecorder
	w interface {
		Write([]byte) (int, error)
	}
}

func (s *streamRecorder) Write(b []byte) (int, error) {
	_, _ = s.ResponseRecorder.Write(b)
	return s.w.Write(b)
}

func (s *streamRecorder) Flush() {}
