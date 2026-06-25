package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/initiativereview"
)

// seedBacklogItem writes a minimal backlog item to disk through the real
// FileStore so adapter reads exercise the actual on-disk spec.json path.
func seedBacklogItem(t *testing.T, store *backlog.FileStore, kind backlog.BacklogKind, name, title string) {
	t.Helper()
	if err := os.MkdirAll(store.ItemDir(kind, name), 0o755); err != nil {
		t.Fatalf("mkdir item %s/%s: %v", kind, name, err)
	}
	if err := store.SaveItem(backlog.BacklogItem{
		Kind:     kind,
		Name:     name,
		Title:    title,
		Status:   backlog.StatusReady,
		Priority: 5,
	}); err != nil {
		t.Fatalf("save item %s/%s: %v", kind, name, err)
	}
}

// TestInitiativeReviewBacklogAdapter_NilStore guards the constructor's
// nil-store contract: passing a nil store yields a nil adapter (not a
// non-nil adapter wrapping nil that would panic on first read).
func TestInitiativeReviewBacklogAdapter_NilStore(t *testing.T) {
	if got := newInitiativeReviewBacklogAdapter(nil); got != nil {
		t.Fatalf("newInitiativeReviewBacklogAdapter(nil) = %v, want nil", got)
	}
}

// TestInitiativeReviewBacklogAdapter_ReadsThroughToStore proves the adapter
// narrows backlog.Store to LoadItem + ItemDir and forwards both faithfully
// to the real FileStore: a seeded item round-trips its title, ItemDir
// matches the store's own path, and a missing item surfaces ErrNotFound
// (absence is an error here, distinct from the execution adapter's
// absence-is-not-an-error contract).
func TestInitiativeReviewBacklogAdapter_ReadsThroughToStore(t *testing.T) {
	root := t.TempDir()
	store := backlog.NewFileStore(root)
	seedBacklogItem(t, store, backlog.KindExecute, "do-thing", "Do thing")

	adapter := newInitiativeReviewBacklogAdapter(store)
	if adapter == nil {
		t.Fatal("expected non-nil adapter for non-nil store")
	}

	item, err := adapter.LoadItem(backlog.KindExecute, "do-thing")
	if err != nil {
		t.Fatalf("LoadItem: %v", err)
	}
	if item.Title != "Do thing" || item.Name != "do-thing" || item.Kind != backlog.KindExecute {
		t.Fatalf("LoadItem returned %+v", item)
	}

	if got, want := adapter.ItemDir(backlog.KindExecute, "do-thing"), store.ItemDir(backlog.KindExecute, "do-thing"); got != want {
		t.Fatalf("ItemDir = %q, want %q", got, want)
	}

	if _, err := adapter.LoadItem(backlog.KindExecute, "missing"); !errors.Is(err, backlog.ErrNotFound) {
		t.Fatalf("LoadItem(missing) err = %v, want ErrNotFound", err)
	}
}

// newExecServiceWithRecords builds a real execution.Service whose store is
// pre-seeded with the given records (written via the real FileStore at the
// service's StorePath). Terminal records carry finalization but no live
// run, so List's ProcessActiveExecutions pass is a no-op and never reaches
// out to agent-manager.
func newExecServiceWithRecords(t *testing.T, records []execution.Record) *execution.Service {
	t.Helper()
	root := t.TempDir()
	storePath := filepath.Join(root, ".vrooli", "execution-runs.json")
	if err := execution.NewStore(storePath).Save(records); err != nil {
		t.Fatalf("seed execution store: %v", err)
	}
	return execution.NewService(execution.ServiceConfig{
		DataRoot:  root,
		StorePath: storePath,
	})
}

// TestInitiativeReviewExecutionAdapter_NilDeps guards both nil branches of
// the constructor: a nil execution service or a nil backlog store must
// yield a nil adapter so initiativereview.NewService can detect the
// degraded wiring rather than dereference a half-built adapter.
func TestInitiativeReviewExecutionAdapter_NilDeps(t *testing.T) {
	store := backlog.NewFileStore(t.TempDir())
	svc := newExecServiceWithRecords(t, nil)

	if got := newInitiativeReviewExecutionAdapter(nil, store); got != nil {
		t.Fatalf("nil exec → %v, want nil adapter", got)
	}
	if got := newInitiativeReviewExecutionAdapter(svc, nil); got != nil {
		t.Fatalf("nil store → %v, want nil adapter", got)
	}
	if got := newInitiativeReviewExecutionAdapter(svc, store); got == nil {
		t.Fatal("both deps present → nil adapter, want non-nil")
	}
}

// TestInitiativeReviewExecutionAdapter_LatestFinalizationFor exercises the
// adapter's core projection: among records for an item (List returns them
// created_at descending), the first one carrying a Finalization wins, and
// only its AffectedScenarios are surfaced — defensively copied so callers
// can't mutate the record's slice. Records without finalization are
// skipped, not treated as the answer.
func TestInitiativeReviewExecutionAdapter_LatestFinalizationFor(t *testing.T) {
	store := backlog.NewFileStore(t.TempDir())
	// Newest record (highest created_at) has NO finalization → must be
	// skipped in favor of the older one that does, proving "first with a
	// populated Finalization" rather than "first record".
	records := []execution.Record{
		{
			ExecutionID: "exec-old",
			BacklogKind: string(backlog.KindExecute),
			BacklogName: "do-thing",
			Status:      execution.StatusCompleted,
			CreatedAt:   "2026-01-01T00:00:00Z",
			Finalization: &execution.Finalization{
				Eligible:          true,
				AffectedScenarios: []string{"alpha", "beta"},
			},
		},
		{
			ExecutionID:  "exec-new",
			BacklogKind:  string(backlog.KindExecute),
			BacklogName:  "do-thing",
			Status:       execution.StatusCompleted,
			CreatedAt:    "2026-02-01T00:00:00Z",
			Finalization: nil,
		},
	}
	svc := newExecServiceWithRecords(t, records)

	adapter := newInitiativeReviewExecutionAdapter(svc, store)
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}

	fin, err := adapter.LatestFinalizationFor(backlog.KindExecute, "do-thing")
	if err != nil {
		t.Fatalf("LatestFinalizationFor: %v", err)
	}
	if fin == nil {
		t.Fatal("expected finalization, got nil")
	}
	if len(fin.AffectedScenarios) != 2 || fin.AffectedScenarios[0] != "alpha" || fin.AffectedScenarios[1] != "beta" {
		t.Fatalf("affected scenarios = %v", fin.AffectedScenarios)
	}
	// Defensive-copy contract: mutating the returned slice must not bleed
	// back into a subsequent read.
	fin.AffectedScenarios[0] = "MUTATED"
	again, err := adapter.LatestFinalizationFor(backlog.KindExecute, "do-thing")
	if err != nil {
		t.Fatalf("LatestFinalizationFor (2nd): %v", err)
	}
	if again.AffectedScenarios[0] != "alpha" {
		t.Fatalf("returned slice aliases stored record: %v", again.AffectedScenarios)
	}
}

// TestInitiativeReviewExecutionAdapter_NoFinalizationIsNotError pins the
// absence-is-not-an-error contract: an item with records but none carrying
// finalization (e.g. research items) returns (nil, nil), and an item with
// no records at all does too. Callers read this as "no scenarios in scope",
// not a failure.
func TestInitiativeReviewExecutionAdapter_NoFinalizationIsNotError(t *testing.T) {
	store := backlog.NewFileStore(t.TempDir())
	records := []execution.Record{
		{
			ExecutionID:  "exec-1",
			BacklogKind:  string(backlog.KindResearch),
			BacklogName:  "investigate",
			Status:       execution.StatusCompleted,
			CreatedAt:    "2026-01-01T00:00:00Z",
			Finalization: nil,
		},
	}
	svc := newExecServiceWithRecords(t, records)
	adapter := newInitiativeReviewExecutionAdapter(svc, store)

	fin, err := adapter.LatestFinalizationFor(backlog.KindResearch, "investigate")
	if err != nil {
		t.Fatalf("records-without-finalization err = %v, want nil", err)
	}
	if fin != nil {
		t.Fatalf("records-without-finalization fin = %+v, want nil", fin)
	}

	fin, err = adapter.LatestFinalizationFor(backlog.KindExecute, "never-ran")
	if err != nil {
		t.Fatalf("no-records err = %v, want nil", err)
	}
	if fin != nil {
		t.Fatalf("no-records fin = %+v, want nil", fin)
	}
}

// TestInitiativeReviewExecutionAdapter_NilReceiver pins the method's own
// nil-receiver guard (distinct from the constructor's nil-deps guard): a
// nil adapter returns (nil, nil) rather than panicking, matching the
// documented degraded-mode contract.
func TestInitiativeReviewExecutionAdapter_NilReceiver(t *testing.T) {
	var nilAdapter *initiativeReviewExecutionAdapter
	fin, err := nilAdapter.LatestFinalizationFor(backlog.KindExecute, "x")
	if fin != nil || err != nil {
		t.Fatalf("nil adapter LatestFinalizationFor = (%v, %v), want (nil, nil)", fin, err)
	}
}

// fakeReviewClient is an honest stand-in for execution.ReviewClient: it
// records the requests it received and returns scripted responses so the
// GCT adapter's request-shaping and result-projection can be asserted
// without a live git-control-tower.
type fakeReviewClient struct {
	triggerReq    execution.ReviewRequest
	triggerJobID  string
	triggerErr    error
	polledJobID   string
	pollResult    *execution.ReviewResult
	pollDone      bool
	pollErr       error
	triggerCalled bool
	pollCalled    bool
}

func (f *fakeReviewClient) TriggerReview(_ context.Context, req execution.ReviewRequest) (string, error) {
	f.triggerCalled = true
	f.triggerReq = req
	return f.triggerJobID, f.triggerErr
}

func (f *fakeReviewClient) PollReview(_ context.Context, jobID string) (*execution.ReviewResult, bool, error) {
	f.pollCalled = true
	f.polledJobID = jobID
	return f.pollResult, f.pollDone, f.pollErr
}

// Ping is part of the ReviewClient interface; the GCT adapter never calls
// it, so it stays a trivial honest stub.
func (f *fakeReviewClient) Ping(_ context.Context) error { return nil }

// TestInitiativeReviewGCTAdapter_NilClient guards the constructor and the
// no-client method paths: a nil client yields a nil adapter; and the
// method receivers themselves no-op safely (TriggerReview → "", nil;
// PollReview → nil, done=true, nil) when invoked on a nil adapter, which
// is the documented degraded-mode contract.
func TestInitiativeReviewGCTAdapter_NilClient(t *testing.T) {
	if got := newInitiativeReviewGCTAdapter(nil); got != nil {
		t.Fatalf("newInitiativeReviewGCTAdapter(nil) = %v, want nil", got)
	}

	var nilAdapter *initiativeReviewGCTAdapter
	jobID, err := nilAdapter.TriggerReview(context.Background(), "alpha")
	if jobID != "" || err != nil {
		t.Fatalf("nil adapter TriggerReview = (%q, %v), want (\"\", nil)", jobID, err)
	}
	res, done, err := nilAdapter.PollReview(context.Background(), "job-1")
	if res != nil || !done || err != nil {
		t.Fatalf("nil adapter PollReview = (%v, %v, %v), want (nil, true, nil)", res, done, err)
	}
}

// TestInitiativeReviewGCTAdapter_TriggerShapesMinimalRequest pins that the
// adapter builds a minimal ReviewRequest carrying only the scenario name —
// no SandboxID, no ExpectedPaths, no thresholds — so GCT defaults apply
// and verdicts stay comparable across initiative-review runs.
func TestInitiativeReviewGCTAdapter_TriggerShapesMinimalRequest(t *testing.T) {
	client := &fakeReviewClient{triggerJobID: "job-42"}
	adapter := newInitiativeReviewGCTAdapter(client)
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}

	jobID, err := adapter.TriggerReview(context.Background(), "swarm-manager")
	if err != nil {
		t.Fatalf("TriggerReview: %v", err)
	}
	if jobID != "job-42" {
		t.Fatalf("jobID = %q, want job-42", jobID)
	}
	if !client.triggerCalled {
		t.Fatal("expected TriggerReview to reach the client")
	}
	if client.triggerReq.ScenarioName != "swarm-manager" {
		t.Fatalf("request scenario = %q, want swarm-manager", client.triggerReq.ScenarioName)
	}
	if client.triggerReq.SandboxID != "" || len(client.triggerReq.ExpectedPaths) != 0 {
		t.Fatalf("request carried non-minimal fields: %+v", client.triggerReq)
	}
}

// TestInitiativeReviewGCTAdapter_PollProjectsResult pins the projection
// from execution.ReviewResult into initiativereview.GCTResult on the
// done-with-result path: the verdict-relevant fields (JobID,
// Classification, Summary, RawDimensions, ReviewedAt) carry through, and
// the polled job ID is forwarded verbatim.
func TestInitiativeReviewGCTAdapter_PollProjectsResult(t *testing.T) {
	raw := json.RawMessage(`{"build":"green"}`)
	client := &fakeReviewClient{
		pollDone: true,
		pollResult: &execution.ReviewResult{
			JobID:          "job-42",
			Classification: "safe",
			Summary:        "all green",
			RawDimensions:  raw,
			ReviewedAt:     "2026-03-01T00:00:00Z",
		},
	}
	adapter := newInitiativeReviewGCTAdapter(client)

	res, done, err := adapter.PollReview(context.Background(), "job-42")
	if err != nil {
		t.Fatalf("PollReview: %v", err)
	}
	if !done {
		t.Fatal("expected done=true")
	}
	if client.polledJobID != "job-42" {
		t.Fatalf("polled job ID = %q, want job-42", client.polledJobID)
	}
	if res == nil {
		t.Fatal("expected projected result, got nil")
	}
	if res.JobID != "job-42" || res.Classification != "safe" || res.Summary != "all green" || res.ReviewedAt != "2026-03-01T00:00:00Z" {
		t.Fatalf("projected result = %+v", res)
	}
	if string(res.RawDimensions) != string(raw) {
		t.Fatalf("raw dimensions = %s, want %s", res.RawDimensions, raw)
	}
}

// TestInitiativeReviewGCTAdapter_PollNotDoneOrErrorReturnsNil pins the two
// short-circuit branches: a not-yet-done poll returns (nil, false, nil)
// without fabricating a result, and a poll error propagates with a nil
// result. Either way the adapter never invents a GCTResult.
func TestInitiativeReviewGCTAdapter_PollNotDoneOrErrorReturnsNil(t *testing.T) {
	t.Run("not done", func(t *testing.T) {
		client := &fakeReviewClient{pollDone: false, pollResult: nil}
		adapter := newInitiativeReviewGCTAdapter(client)
		res, done, err := adapter.PollReview(context.Background(), "job-1")
		if res != nil || done || err != nil {
			t.Fatalf("not-done poll = (%v, %v, %v), want (nil, false, nil)", res, done, err)
		}
	})

	t.Run("error", func(t *testing.T) {
		wantErr := errors.New("gct down")
		client := &fakeReviewClient{pollDone: true, pollErr: wantErr}
		adapter := newInitiativeReviewGCTAdapter(client)
		res, _, err := adapter.PollReview(context.Background(), "job-1")
		if res != nil {
			t.Fatalf("error poll result = %v, want nil", res)
		}
		if !errors.Is(err, wantErr) {
			t.Fatalf("error poll err = %v, want %v", err, wantErr)
		}
	})
}

// Compile-time proof the adapters still satisfy the initiativereview
// interfaces they are constructed to fill — a guard against silent
// interface drift that would otherwise only surface at wiring time.
var (
	_ initiativereview.BacklogLoader   = (*initiativeReviewBacklogAdapter)(nil)
	_ initiativereview.ExecutionLookup = (*initiativeReviewExecutionAdapter)(nil)
	_ initiativereview.GCTClient       = (*initiativeReviewGCTAdapter)(nil)
	_ execution.ReviewClient           = (*fakeReviewClient)(nil)
)
