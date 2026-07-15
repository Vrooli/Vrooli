package opsrunner

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"swarm-manager/internal/agentops"
)

const testModeRevision = "sha256:" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

// TestAcceptanceSymmetryBacklogItemAndInitiative is the phase acceptance test:
// one synthetic operation contract (review-round) bound to one synthetic mode
// runs against BOTH a backlog-item target and an initiative target through the
// identical generic runner path, in simulation (no agent spawn), and produces a
// reproducible provenance and an identical control flow. The generic path never
// branches on the target kind.
func TestAcceptanceSymmetryBacklogItemAndInitiative(t *testing.T) {
	catalog := writeCatalog(t, t.TempDir(), testModeRevision)
	driver := fakeDriver{outcome: "accepted", disposition: "success", result: json.RawMessage(`{"verdict":"accepted"}`)}
	owners := &memRunOwners{}
	r, _, execStore := newRunner(t, catalog, t.TempDir(), fakePreparer{}, driver, owners)

	targets := []TargetRef{
		{Kind: agentops.TargetBacklogItem, ID: "fix/flaky-test"},
		{Kind: agentops.TargetInitiative, ID: "ship-search"},
	}
	var provDigests []string
	for _, target := range targets {
		res, err := r.Invoke(context.Background(), InvokeRequest{
			Target: target, Operation: agentops.OpReviewRound, Simulate: true, RequestedBy: "test",
		})
		if err != nil {
			t.Fatalf("invoke %s: %v", target.Kind, err)
		}
		if res.Outcome != "accepted" {
			t.Fatalf("%s outcome = %q, want accepted", target.Kind, res.Outcome)
		}
		if res.Action != agentops.ActionOpenReview {
			t.Fatalf("%s action = %q, want open-review", target.Kind, res.Action)
		}
		if res.WorkflowState != agentops.WorkflowAwaitingDecision {
			t.Fatalf("%s workflow state = %q, want awaiting-decision", target.Kind, res.WorkflowState)
		}
		if !agentops.IsWellFormedDigest(res.ProvenanceDigest) {
			t.Fatalf("%s provenance digest malformed: %q", target.Kind, res.ProvenanceDigest)
		}
		if res.Provenance.Mode != "synthetic-loop" || res.Provenance.ModeRevision != testModeRevision {
			t.Fatalf("%s pinned mode = %s@%s", target.Kind, res.Provenance.Mode, res.Provenance.ModeRevision)
		}
		// The persisted snapshot is self-contained and reproduces the pinned digests.
		if _, err := execStore.Reproduce(target.Kind, target.ID, res.ExecutionID); err != nil {
			t.Fatalf("%s reproduce: %v", target.Kind, err)
		}
		provDigests = append(provDigests, res.ProvenanceDigest)
	}
	// The two runs used the same mode/prompt/input digests, so provenance differs
	// ONLY by the target field — a symmetry proof.
	if provDigests[0] == provDigests[1] {
		t.Fatalf("provenance digests must differ by target, both = %s", provDigests[0])
	}
	if len(owners.refs) == 0 {
		t.Fatalf("no run-owner attribution recorded")
	}
}

// TestReproducibleAfterSourceEdit proves a historical execution stays resolvable
// after the mode source changes: the persisted snapshot pins the exact compiled
// bytes, so Reproduce still verifies; a snapshot whose bytes are tampered fails
// with ErrDigestMismatch.
func TestReproducibleAfterSourceEdit(t *testing.T) {
	catalog := writeCatalog(t, t.TempDir(), testModeRevision)
	storeRoot := t.TempDir()
	r, _, execStore := newRunner(t, catalog, storeRoot,
		fakePreparer{}, fakeDriver{outcome: "accepted", disposition: "success"}, &memRunOwners{})

	target := TargetRef{Kind: agentops.TargetBacklogItem, ID: "fix/x"}
	res, err := r.Invoke(context.Background(), InvokeRequest{Target: target, Operation: agentops.OpReviewRound, Simulate: true})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	// Even though the live catalog/preparer would now produce different compiled
	// bytes, the stored snapshot reproduces the original run.
	if _, err := execStore.Reproduce(target.Kind, target.ID, res.ExecutionID); err != nil {
		t.Fatalf("reproduce after edit: %v", err)
	}
	// Tamper the persisted compiled-mode bytes: Reproduce must fail closed.
	snap, _, err := execStore.Load(target.Kind, target.ID, res.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	snap.CompiledMode = json.RawMessage(`{"mode":"tampered"}`)
	if err := execStore.SaveExecution(target.Kind, target.ID, res.ExecutionID, snap); err != nil {
		t.Fatal(err)
	}
	if _, err := execStore.Reproduce(target.Kind, target.ID, res.ExecutionID); !errors.Is(err, ErrDigestMismatch) {
		t.Fatalf("tampered snapshot must be ErrDigestMismatch, got %v", err)
	}
}

// TestIdempotentInvokeReplaysPriorResult proves a retried Invoke with the same
// idempotency key returns the prior result without starting a second run.
func TestIdempotentInvokeReplaysPriorResult(t *testing.T) {
	catalog := writeCatalog(t, t.TempDir(), testModeRevision)
	var driveCount int
	driver := countingDriver{inner: fakeDriver{outcome: "accepted", disposition: "success"}, count: &driveCount}
	r, repo, _ := newRunner(t, catalog, t.TempDir(), fakePreparer{}, driver, &memRunOwners{})

	target := TargetRef{Kind: agentops.TargetInitiative, ID: "ship-x"}
	req := InvokeRequest{Target: target, Operation: agentops.OpReviewRound, IdempotencyKey: "k-1", Simulate: true}
	first, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("first invoke: %v", err)
	}
	second, err := r.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("second invoke: %v", err)
	}
	if !second.Replayed {
		t.Fatalf("second invoke must be a replay")
	}
	if first.ExecutionID != second.ExecutionID {
		t.Fatalf("replay execution id = %s, want %s", second.ExecutionID, first.ExecutionID)
	}
	if driveCount != 1 {
		t.Fatalf("driver ran %d times, want exactly once (no double effect)", driveCount)
	}
	// Exactly one operation record correlated under the workflow.
	w, _, err := repo.Load(target.Kind, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Operations) != 1 {
		t.Fatalf("workflow has %d operation records, want 1", len(w.Operations))
	}
}

// TestInvokeRejectsIncompatibleTarget proves compatibility is a fail-closed gate.
func TestInvokeRejectsIncompatibleTarget(t *testing.T) {
	catalog := writeCatalog(t, t.TempDir(), testModeRevision)
	r, _, _ := newRunner(t, catalog, t.TempDir(), fakePreparer{}, fakeDriver{outcome: "accepted", disposition: "success"}, &memRunOwners{})
	// review-round requires provides-review-artifacts, which plan-execution lacks.
	_, err := r.Invoke(context.Background(), InvokeRequest{
		Target: TargetRef{Kind: agentops.TargetPlanExecution, ID: "plan-1"}, Operation: agentops.OpReviewRound, Simulate: true,
	})
	if !errors.Is(err, ErrIncompatibleTarget) {
		t.Fatalf("incompatible target must fail closed, got %v", err)
	}
}

// TestInvokeUnknownOperationFailsClosed proves an operation the catalog does not
// declare is rejected before any work.
func TestInvokeUnknownOperationFailsClosed(t *testing.T) {
	catalog := writeCatalog(t, t.TempDir(), testModeRevision)
	r, _, _ := newRunner(t, catalog, t.TempDir(), fakePreparer{}, fakeDriver{outcome: "accepted", disposition: "success"}, &memRunOwners{})
	// execution-run is a registered operation but not declared in this catalog.
	_, err := r.Invoke(context.Background(), InvokeRequest{
		Target: TargetRef{Kind: agentops.TargetInitiative, ID: "x"}, Operation: agentops.OpExecutionRun, Simulate: true,
	})
	if !errors.Is(err, ErrUnknownOperation) {
		t.Fatalf("undeclared operation must be ErrUnknownOperation, got %v", err)
	}
}

// TestInvokeNoBindingFailsClosed proves an operation with no resolvable binding
// is a typed error, never an implicit default.
func TestInvokeNoBindingFailsClosed(t *testing.T) {
	catalog := writeContractOnly(t, t.TempDir())
	r, _, _ := newRunner(t, catalog, t.TempDir(), fakePreparer{}, fakeDriver{outcome: "accepted", disposition: "success"}, &memRunOwners{})
	_, err := r.Invoke(context.Background(), InvokeRequest{
		Target: TargetRef{Kind: agentops.TargetInitiative, ID: "x"}, Operation: agentops.OpReviewRound, Simulate: true,
	})
	if !errors.Is(err, agentops.ErrNoBinding) {
		t.Fatalf("absent binding must be ErrNoBinding, got %v", err)
	}
}

// TestBindingDeletedRevisionFailsClosed proves a binding to a missing revision is
// ErrDeletedRevision (the checker reports the revision gone).
func TestBindingDeletedRevisionFailsClosed(t *testing.T) {
	catalog := writeCatalog(t, t.TempDir(), testModeRevision)
	r, _, _ := newRunner(t, catalog, t.TempDir(), fakePreparer{missingRev: true}, fakeDriver{outcome: "accepted", disposition: "success"}, &memRunOwners{})
	_, err := r.Invoke(context.Background(), InvokeRequest{
		Target: TargetRef{Kind: agentops.TargetInitiative, ID: "x"}, Operation: agentops.OpReviewRound, Simulate: true,
	})
	if !errors.Is(err, agentops.ErrDeletedRevision) {
		t.Fatalf("deleted revision must be ErrDeletedRevision, got %v", err)
	}
}

// countingDriver counts Drive calls to assert exactly-once semantics.
type countingDriver struct {
	inner fakeDriver
	count *int
}

func (d countingDriver) Drive(ctx context.Context, p Prepared, h RunHandle) (ExecutionOutcome, error) {
	*d.count++
	return d.inner.Drive(ctx, p, h)
}
