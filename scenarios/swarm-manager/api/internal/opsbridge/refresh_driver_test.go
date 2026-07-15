package opsbridge

import (
	"context"
	"errors"
	"testing"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/operatingmode"
)

type fakeListerImpl struct {
	ws  []agentops.WorkflowInstance
	err error
}

func (f fakeListerImpl) List() ([]agentops.WorkflowInstance, error) { return f.ws, f.err }

type fakeRefresher struct {
	refreshed []string
	err       error
}

func (f *fakeRefresher) RefreshRunByID(_ context.Context, runID string) (operatingmode.RoundEnvelope, bool, error) {
	f.refreshed = append(f.refreshed, runID)
	if f.err != nil {
		return operatingmode.RoundEnvelope{}, false, f.err
	}
	return operatingmode.RoundEnvelope{}, true, nil
}

func wfWith(ops ...agentops.OperationExecutionRecord) agentops.WorkflowInstance {
	return agentops.WorkflowInstance{Operations: ops}
}

func TestRefreshDriverRefreshesOnlyRunningOpsWithRunIDs(t *testing.T) {
	lister := fakeListerImpl{ws: []agentops.WorkflowInstance{
		wfWith(
			agentops.OperationExecutionRecord{ExecutionID: "e1", RunID: "run-1", State: "running"},
			agentops.OperationExecutionRecord{ExecutionID: "e2", RunID: "run-2", State: "completed"}, // terminal: skip
			agentops.OperationExecutionRecord{ExecutionID: "e3", RunID: "", State: "running"},        // no run id: skip
		),
		wfWith(agentops.OperationExecutionRecord{ExecutionID: "e4", RunID: "run-4", State: "running"}),
	}}
	refresher := &fakeRefresher{}
	if err := NewRefreshDriver(lister, refresher, nil).Tick(context.Background()); err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if len(refresher.refreshed) != 2 {
		t.Fatalf("expected 2 refreshes, got %v", refresher.refreshed)
	}
	want := map[string]bool{"run-1": true, "run-4": true}
	for _, r := range refresher.refreshed {
		if !want[r] {
			t.Fatalf("unexpected refresh of %q", r)
		}
	}
}

func TestRefreshDriverContinuesPastPerRunError(t *testing.T) {
	lister := fakeListerImpl{ws: []agentops.WorkflowInstance{
		wfWith(
			agentops.OperationExecutionRecord{ExecutionID: "e1", RunID: "run-1", State: "running"},
			agentops.OperationExecutionRecord{ExecutionID: "e2", RunID: "run-2", State: "running"},
		),
	}}
	refresher := &fakeRefresher{err: errors.New("refresh boom")}
	// A per-run error is logged and skipped; the sweep still visits every run and
	// Tick returns nil (only a List error aborts the sweep).
	if err := NewRefreshDriver(lister, refresher, nil).Tick(context.Background()); err != nil {
		t.Fatalf("per-run error must not abort the sweep: %v", err)
	}
	if len(refresher.refreshed) != 2 {
		t.Fatalf("expected both runs attempted despite errors, got %v", refresher.refreshed)
	}
}

func TestRefreshDriverPropagatesListError(t *testing.T) {
	refresher := &fakeRefresher{}
	err := NewRefreshDriver(fakeListerImpl{err: errors.New("list boom")}, refresher, nil).Tick(context.Background())
	if err == nil {
		t.Fatalf("a List error must abort the sweep")
	}
	if len(refresher.refreshed) != 0 {
		t.Fatalf("no refreshes should occur when listing fails")
	}
}
