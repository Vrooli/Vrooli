package recovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/coreactivation"
)

func nativeRecoveryFixture(t *testing.T) (*CodingAgentBroker, CodingAgentExecution) {
	t.Helper()
	b, err := NewCodingAgentBroker(filepath.Join(t.TempDir(), "receipts.json"), nil)
	if err != nil {
		t.Fatal(err)
	}
	id := "00000000-0000-4000-8000-000000000001"
	e := CodingAgentExecution{Version: 1, ID: id, Request: CodingAgentRequest{Scenario: "web-console", Requester: "test", WorkingDir: t.TempDir(), Claim: filepath.Join("scenarios", "web-console")}, BootID: "00000000-0000-4000-8000-000000000002", Deadline: time.Now().Add(time.Minute), Fence: "ready"}
	dir, _ := b.executionDir(id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	worker, _ := b.workerPath(id)
	if err := os.WriteFile(worker, []byte("retained worker bytes"), 0o700); err != nil {
		t.Fatal(err)
	}
	e.WorkerSHA, err = coreactivation.ImageDigest(context.Background(), worker)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := b.nativeSpec(e)
	if err != nil {
		t.Fatal(err)
	}
	e.Intent, err = platform.NewOperationIntent("coding-agent-recovery", id, spec)
	if err != nil {
		t.Fatal(err)
	}
	b.records = []CodingAgentRecord{{ID: id, Scenario: e.Request.Scenario, Requester: "test", Outcome: "scheduled", CreatedAt: time.Now(), DeadlineAt: e.Deadline}}
	if err := b.persistLocked(); err != nil {
		t.Fatal(err)
	}
	if err := b.writeExecution(e); err != nil {
		t.Fatal(err)
	}
	b.native = nativeCodingOwner("")
	return b, e
}

func nativeRecoveryRef(t *testing.T, e CodingAgentExecution) platform.OperationRef {
	t.Helper()
	unit, err := e.Intent.UnitName()
	if err != nil {
		t.Fatal(err)
	}
	r := platform.OperationRef{Intent: e.Intent, Unit: unit, Scope: platform.ScopeRef{Kind: platform.ScopeKindCgroup, Path: "/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-operations.slice/" + unit}, RootPID: 42, RootStartTime: "123", NativeInvocationID: strings.Repeat("a", 32), ScopeDevice: 1, ScopeInode: 2}
	if err := r.ValidateIdentity(); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestNativeCodingDispatchLostAcknowledgementCannotResubmit(t *testing.T) {
	b, e := nativeRecoveryFixture(t)
	calls := 0
	b.native.start = func(ctx context.Context, intent platform.OperationIntent, spec platform.OperationSpec) (*platform.OwnedOperation, error) {
		calls++
		committed, err := b.readExecution(e.ID)
		if err != nil || !committed.DispatchCommitted {
			t.Fatal("native dispatch preceded durable intent")
		}
		return nil, errors.New("lost acknowledgement")
	}
	if err := b.dispatchNative(context.Background(), &e); err == nil {
		t.Fatal("ambiguous dispatch reported success")
	}
	loaded, err := b.readExecution(e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.dispatchNative(context.Background(), &loaded); err == nil {
		t.Fatal("consumed dispatch was replayed")
	}
	if calls != 1 {
		t.Fatalf("dispatch count=%d", calls)
	}
}

func TestNativeCodingAdmissionResumesFirstSubmissionWithoutReplayingAmbiguousStart(t *testing.T) {
	for _, committed := range []bool{false, true} {
		name := "before-dispatch"
		if committed {
			name = "dispatch-acknowledgement-lost"
		}
		t.Run(name, func(t *testing.T) {
			b, e := nativeRecoveryFixture(t)
			e.DispatchCommitted = committed
			if err := b.writeExecution(e); err != nil {
				t.Fatal(err)
			}
			ref := nativeRecoveryRef(t, e)
			absent, started, calls := false, false, 0
			b.native.lookup = func(context.Context, platform.OperationIntent) (platform.OperationRef, error) {
				if started {
					return ref, nil
				}
				return platform.OperationRef{}, platform.ErrOperationOutcomeLost
			}
			b.native.absent = func(context.Context, platform.OperationIntent) (bool, error) { return absent, nil }
			b.native.inspect = func(context.Context, platform.OperationRef) (platform.OperationObservation, error) {
				return platform.OperationObservation{State: platform.OperationRunning, RootAlive: true, Populated: true}, nil
			}
			b.native.start = func(ctx context.Context, intent platform.OperationIntent, spec platform.OperationSpec) (*platform.OwnedOperation, error) {
				calls++
				if intent != e.Intent || intent.AttemptID != e.ID {
					t.Fatal("recovery replaced the original dispatch identity")
				}
				if err := intent.ValidateSpec(spec); err != nil {
					t.Fatal("recovery changed the admitted native specification")
				}
				loaded, err := b.readExecution(e.ID)
				if err != nil || !loaded.DispatchCommitted || loaded.Fence != "ready" || !loaded.Deadline.Equal(e.Deadline) {
					t.Fatal("resumed dispatch lacks durable original fence/deadline")
				}
				started = true
				return &platform.OwnedOperation{Ref: ref}, nil
			}
			_, _ = b.ReconcileNative(context.Background(), 32)
			if calls != 0 {
				t.Fatal("partial native absence allowed redispatch")
			}
			absent = true
			if _, err := b.ReconcileNative(context.Background(), 32); err != nil {
				t.Fatalf("admitted dispatch did not resume: %v", err)
			}
			want := 1
			if committed {
				want = 0
			}
			if calls != want {
				t.Fatalf("first submission lost or uncertain start replayed: starts=%d want=%d", calls, want)
			}
			if _, err := b.ReconcileNative(context.Background(), 32); err != nil || calls != want {
				t.Fatalf("live dispatch was repeated: calls=%d error=%v", calls, err)
			}
		})
	}
}

func TestNativeCodingExpiredOutcomeLostRetiresOnlyWithCompleteAbsence(t *testing.T) {
	b, e := nativeRecoveryFixture(t)
	e.DispatchCommitted = true
	e.Deadline = time.Now().Add(-time.Second)
	if err := b.writeExecution(e); err != nil {
		t.Fatal(err)
	}
	b.native.lookup = func(context.Context, platform.OperationIntent) (platform.OperationRef, error) {
		return platform.OperationRef{}, platform.ErrOperationOutcomeLost
	}
	absent := false
	b.native.absent = func(context.Context, platform.OperationIntent) (bool, error) { return absent, nil }
	b.native.start = func(context.Context, platform.OperationIntent, platform.OperationSpec) (*platform.OwnedOperation, error) {
		t.Fatal("expired repair was dispatched")
		return nil, nil
	}
	if _, err := b.ReconcileNative(context.Background(), 32); err == nil {
		t.Fatal("partial absence became expiration proof")
	}
	absent = true
	if _, err := b.ReconcileNative(context.Background(), 32); err != nil {
		t.Fatalf("actual platform absence contract cannot retire expired work: %v", err)
	}
	loaded, err := b.readExecution(e.ID)
	if err != nil || !loaded.ExitProven || !loaded.Retired || loaded.Fence != "revoked" {
		t.Fatalf("expired inputs not retired: %+v %v", loaded, err)
	}
	rows, err := b.Snapshot()
	if err != nil || rows[0].Outcome != "failed" || rows[0].FinishedAt.IsZero() {
		t.Fatal("retirement invented success or left an unresolved claim")
	}
}

func TestNativeCodingUnadmittedPreparationRevokesBeforeRetiringWithoutDispatch(t *testing.T) {
	b, e := nativeRecoveryFixture(t)
	b.records = nil
	if err := b.persistLocked(); err != nil {
		t.Fatal(err)
	}
	dir, _ := b.executionDir(e.ID)
	temporary := filepath.Join(dir, ".vrooli-owned-interrupted-copy")
	if err := os.WriteFile(temporary, []byte("partial copy"), 0o600); err != nil {
		t.Fatal(err)
	}
	b.native.start = func(context.Context, platform.OperationIntent, platform.OperationSpec) (*platform.OwnedOperation, error) {
		t.Fatal("unadmitted orphan was dispatched")
		return nil, nil
	}
	b.native.lookup = func(context.Context, platform.OperationIntent) (platform.OperationRef, error) {
		return platform.OperationRef{}, platform.ErrOperationOutcomeLost
	}
	absent := false
	b.native.absent = func(context.Context, platform.OperationIntent) (bool, error) {
		current, err := b.readExecution(e.ID)
		if err != nil || current.Fence != "revoked" {
			t.Fatal("orphan absence preceded durable revoke")
		}
		return absent, nil
	}
	if _, err := b.ReconcileNative(context.Background(), 1); err == nil {
		t.Fatal("orphan partial absence supplied exit proof")
	}
	if _, err := os.Stat(temporary); err != nil {
		t.Fatal("uncertain orphan inputs were removed")
	}
	absent = true
	if n, err := b.ReconcileNative(context.Background(), 1); err != nil || n != 1 {
		t.Fatalf("orphan retirement failed: %d %v", n, err)
	}
	current, err := b.readExecution(e.ID)
	if err != nil || !current.Retired || !current.ExitProven || current.DispatchCommitted {
		t.Fatalf("orphan retirement invented admission or lost evidence: %+v %v", current, err)
	}
	if _, err := os.Stat(temporary); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("interrupted worker copy remains")
	}
	rows, err := b.Snapshot()
	if err != nil || len(rows) != 0 {
		t.Fatal("orphan retirement invented an admission receipt")
	}
}

func TestNativeCodingMissingDescriptorPreservesPublishedWorker(t *testing.T) {
	b, e := nativeRecoveryFixture(t)
	b.records = nil
	if err := b.persistLocked(); err != nil {
		t.Fatal(err)
	}
	dir, _ := b.executionDir(e.ID)
	if err := os.Remove(filepath.Join(dir, "execution.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := b.ReconcileNative(context.Background(), 1); err == nil {
		t.Fatal("unidentified published image was removed")
	}
	worker, _ := b.workerPath(e.ID)
	if _, err := os.Stat(worker); err != nil {
		t.Fatal("unidentified input was discarded")
	}
}

func TestNativeCodingEmptyPreparationAndAtomicTemporaryAreRetired(t *testing.T) {
	b, e := nativeRecoveryFixture(t)
	b.records = nil
	if err := b.persistLocked(); err != nil {
		t.Fatal(err)
	}
	dir, _ := b.executionDir(e.ID)
	worker, _ := b.workerPath(e.ID)
	for _, p := range []string{worker, filepath.Join(dir, "execution.json")} {
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, ".vrooli-owned-interrupted-metadata"), []byte("partial metadata"), 0o600); err != nil {
		t.Fatal(err)
	}
	if count, err := b.ReconcileNative(context.Background(), 1); err != nil || count != 1 {
		t.Fatalf("empty preparation not retired: %d %v", count, err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("empty preparation directory remains")
	}
}

func TestNativeCodingMissingLedgerDoesNotBecomeFreshBudget(t *testing.T) {
	b, e := nativeRecoveryFixture(t)
	if err := os.Remove(b.path); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Snapshot(); err == nil {
		t.Fatal("missing initialized admission ledger became a fresh budget")
	}
	if _, err := b.ReconcileNative(context.Background(), 1); err == nil {
		t.Fatal("missing admission inventory allowed native effects")
	}
	current, err := b.readExecution(e.ID)
	if err != nil || current.Fence != "ready" || current.ExitProven {
		t.Fatal("ledger loss changed native ownership")
	}
	worker, _ := b.workerPath(e.ID)
	if _, err := os.Stat(worker); err != nil {
		t.Fatal("ledger loss discarded retained input")
	}
}

func TestNativeCodingPendingStartRemainsEligibleUntilDeadline(t *testing.T) {
	b, e := nativeRecoveryFixture(t)
	e.DispatchCommitted = true
	if err := b.writeExecution(e); err != nil {
		t.Fatal(err)
	}
	ref := nativeRecoveryRef(t, e)
	ref.RootPID = 0
	ref.RootStartTime = ""
	ref.ScopeDevice = 0
	ref.ScopeInode = 0
	b.native.lookup = func(context.Context, platform.OperationIntent) (platform.OperationRef, error) { return ref, nil }
	b.native.inspect = func(context.Context, platform.OperationRef) (platform.OperationObservation, error) {
		return platform.OperationObservation{State: platform.OperationOutcomeUnknown}, nil
	}
	b.native.stop = func(context.Context, platform.OperationRef) (platform.OperationStopReceipt, error) {
		t.Fatal("accepted pending start was canceled before its deadline")
		return platform.OperationStopReceipt{}, nil
	}
	if _, err := b.ReconcileNative(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	current, err := b.readExecution(e.ID)
	if err != nil || current.Fence != "ready" {
		t.Fatal("pending start lost its original fence")
	}
}

func TestNativeCodingAbsentUnitStillRequiresCapturedRootExit(t *testing.T) {
	b, e := nativeRecoveryFixture(t)
	ref := nativeRecoveryRef(t, e)
	e.DispatchCommitted = true
	e.Fence = "claimed"
	e.Ref = &ref
	e.Deadline = time.Now().Add(-time.Second)
	if err := b.writeExecution(e); err != nil {
		t.Fatal(err)
	}
	b.native.lookup = func(context.Context, platform.OperationIntent) (platform.OperationRef, error) {
		return platform.OperationRef{}, platform.ErrOperationOutcomeLost
	}
	b.native.absent = func(context.Context, platform.OperationIntent) (bool, error) { return true, nil }
	released := false
	b.native.released = func(_ context.Context, r platform.OperationRef, stamp string) (bool, error) {
		if r.RootPID != ref.RootPID || stamp != e.BootID+":"+ref.RootStartTime {
			t.Fatal("release observation lost captured incarnation")
		}
		return released, nil
	}
	b.native.release = func(context.Context, platform.OperationRef) error { return platform.ErrOperationOutcomeLost }
	if _, err := b.ReconcileNative(context.Background(), 1); err == nil {
		t.Fatal("unit absence erased a captured root")
	}
	current, _ := b.readExecution(e.ID)
	if current.ExitProven {
		t.Fatal("live captured root supplied exit proof")
	}
	released = true
	if _, err := b.ReconcileNative(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	current, _ = b.readExecution(e.ID)
	if !current.Retired {
		t.Fatal("captured root exit did not permit retirement")
	}
}

func TestNativeCodingRetirementPreservesUnexpectedContentBeforeRemovingImage(t *testing.T) {
	b, e := nativeRecoveryFixture(t)
	e.Fence = "revoked"
	e.ExitProven = true
	dir, _ := b.executionDir(e.ID)
	worker, _ := b.workerPath(e.ID)
	unexpected := filepath.Join(dir, "unowned.txt")
	if err := os.WriteFile(unexpected, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := b.retireNativeWorker(context.Background(), e); err == nil {
		t.Fatal("unexpected input was silently discarded")
	}
	for _, p := range []string{worker, unexpected} {
		if _, err := os.Stat(p); err != nil {
			t.Fatal("retirement partially removed inputs before validating inventory")
		}
	}
}

func TestNativeCodingChildAndLeaseRefuseChangedAdmission(t *testing.T) {
	for _, field := range []string{"requester", "reason", "missing"} {
		t.Run(field, func(t *testing.T) {
			b, e := nativeRecoveryFixture(t)
			ref := nativeRecoveryRef(t, e)
			e.Fence = "claimed"
			e.DispatchCommitted = true
			e.Ref = &ref
			e.LeaseID = "fixture-lease"
			if err := b.writeExecution(e); err != nil {
				t.Fatal(err)
			}
			b.records[0].Outcome = "running"
			switch field {
			case "requester":
				b.records[0].Requester = "different-owner"
			case "reason":
				b.records[0].Reason = "different-task"
			case "missing":
				b.records = nil
			}
			if err := b.persistLocked(); err != nil {
				t.Fatal(err)
			}
			if err := b.GuardNativeChild(context.Background(), e.ID); err == nil || !strings.Contains(err.Error(), "admission") {
				t.Fatalf("changed admission reached native child identity check: %v", err)
			}
			if err := b.BindNativeLease(context.Background(), e.ID, e.LeaseID, true); err == nil || !strings.Contains(err.Error(), "admission") {
				t.Fatalf("changed admission reached native lease binding: %v", err)
			}
		})
	}
}

func TestNativeCodingRecoveryRevokesBeforeExclusionAndRetiresOnce(t *testing.T) {
	b, e := nativeRecoveryFixture(t)
	ref := nativeRecoveryRef(t, e)
	e.DispatchCommitted = true
	e.Fence = "claimed"
	e.Ref = &ref
	e.Deadline = time.Now().Add(-time.Second)
	if err := b.writeExecution(e); err != nil {
		t.Fatal(err)
	}
	stops := 0
	b.native.lookup = func(context.Context, platform.OperationIntent) (platform.OperationRef, error) { return ref, nil }
	b.native.inspect = func(context.Context, platform.OperationRef) (platform.OperationObservation, error) {
		return platform.OperationObservation{State: platform.OperationStoppedOutcomeKnown}, nil
	}
	b.native.stop = func(context.Context, platform.OperationRef) (platform.OperationStopReceipt, error) {
		stops++
		durable, err := b.readExecution(e.ID)
		if err != nil || durable.Fence != "revoked" {
			t.Fatal("native stop preceded durable revocation")
		}
		return platform.OperationStopReceipt{NativeQuiescent: true}, nil
	}
	b.native.release = func(context.Context, platform.OperationRef) error { return nil }
	if _, err := b.ReconcileNative(context.Background(), 32); err != nil {
		t.Fatal(err)
	}
	r, err := b.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if r[0].Outcome != "failed" || r[0].FinishedAt.IsZero() {
		t.Fatalf("native exit did not close pending work: %+v", r[0])
	}
	loaded, err := b.readExecution(e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.ExitProven || !loaded.Retired || loaded.Fence != "revoked" {
		t.Fatal("retirement lacks exclusion proof")
	}
	worker, _ := b.workerPath(e.ID)
	if _, err := os.Stat(worker); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("retired worker image remains")
	}
	if count, err := b.ReconcileNative(context.Background(), 32); err != nil || count != 0 || stops != 1 {
		t.Fatalf("terminal retirement replayed effects: %d %v", count, err)
	}
}

func TestNativeCodingRecoveryUnknownManagerPreservesPendingInputs(t *testing.T) {
	b, e := nativeRecoveryFixture(t)
	e.DispatchCommitted = true
	if err := b.writeExecution(e); err != nil {
		t.Fatal(err)
	}
	b.native.lookup = func(context.Context, platform.OperationIntent) (platform.OperationRef, error) {
		return platform.OperationRef{}, errors.New("manager unavailable")
	}
	b.native.stop = func(context.Context, platform.OperationRef) (platform.OperationStopReceipt, error) {
		t.Fatal("unknown identity was stopped")
		return platform.OperationStopReceipt{}, nil
	}
	if _, err := b.ReconcileNative(context.Background(), 32); err == nil {
		t.Fatal("manager outage became exit proof")
	}
	r, _ := b.Snapshot()
	if r[0].Outcome != "scheduled" || !r[0].FinishedAt.IsZero() {
		t.Fatal("unknown work was closed")
	}
	worker, _ := b.workerPath(e.ID)
	if _, err := os.Stat(worker); err != nil {
		t.Fatal("unknown worker image was retired")
	}
	loaded, _ := b.readExecution(e.ID)
	if loaded.ExitProven {
		t.Fatal("unknown native state supplied exit proof")
	}
}

func TestNativeCodingLostDispatchNeedsDeadlineAndCompleteAbsence(t *testing.T) {
	b, e := nativeRecoveryFixture(t)
	e.DispatchCommitted = true
	if err := b.writeExecution(e); err != nil {
		t.Fatal(err)
	}
	b.native.lookup = func(context.Context, platform.OperationIntent) (platform.OperationRef, error) {
		return platform.OperationRef{}, platform.ErrOperationNotFound
	}
	absent := false
	b.native.absent = func(context.Context, platform.OperationIntent) (bool, error) { return absent, nil }
	if _, err := b.ReconcileNative(context.Background(), 32); err != nil {
		t.Fatal(err)
	}
	loaded, _ := b.readExecution(e.ID)
	if loaded.Fence != "ready" {
		t.Fatal("in-flight dispatch was revoked before its deadline")
	}
	e.Deadline = time.Now().Add(-time.Second)
	if err := b.writeExecution(e); err != nil {
		t.Fatal(err)
	}
	if _, err := b.ReconcileNative(context.Background(), 32); err == nil {
		t.Fatal("unit lookup alone became absence proof")
	}
	absent = true
	if _, err := b.ReconcileNative(context.Background(), 32); err != nil {
		t.Fatal(err)
	}
	loaded, _ = b.readExecution(e.ID)
	if !loaded.ExitProven || !loaded.Retired {
		t.Fatal("complete absence did not finish retirement")
	}
	if _, err := b.ClaimNativeExecution(context.Background(), e.ID); err == nil {
		t.Fatal("late native worker claimed a revoked execution")
	}
}

func TestNativeCodingSidecarRejectsAmbiguousStateAndClaimTraversal(t *testing.T) {
	b, e := nativeRecoveryFixture(t)
	dir, _ := b.executionDir(e.ID)
	path := filepath.Join(dir, "execution.json")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range []string{strings.Replace(string(original), `"version":1`, `"version":1,"version":1`, 1), strings.Replace(string(original), `"worker_sha256":"`+e.WorkerSHA+`"`, `"worker_sha256":"`+strings.Repeat("x", 64)+`"`, 1), strings.Replace(string(original), `"claim":"scenarios/web-console"`, `"claim":"../other-owner"`, 1)} {
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := b.readExecution(e.ID); err == nil {
			t.Fatal("ambiguous native metadata became authority")
		}
	}
}

func TestNativeCodingRequestRejectsLossyJSONPathsAndRequesterNUL(t *testing.T) {
	_, e := nativeRecoveryFixture(t)
	for _, field := range []string{"working-dir", "requester", "reason", "requester-nul", "reason-nul"} {
		r := e.Request
		switch field {
		case "working-dir":
			r.WorkingDir = string([]byte{'/', 0xff})
		case "requester":
			r.Requester = string([]byte{0xff})
		case "reason":
			r.Reason = string([]byte{0xff})
		case "requester-nul":
			r.Requester = "owner\x00other"
		case "reason-nul":
			r.Reason = "failure\x00other"
		}
		if validNativeCodingRequest(r) {
			t.Fatalf("native request accepted lossy or unusable field %s", field)
		}
	}
}
