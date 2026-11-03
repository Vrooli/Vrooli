package control

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	devicedomain "device-control/internal/devices"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type lifecyclePairingSession struct {
	started  chan struct{}
	release  chan struct{}
	closeErr error
	closes   atomic.Int32
}

func (*lifecyclePairingSession) Complete(context.Context, string) ([]byte, error) { return nil, nil }

func (p *lifecyclePairingSession) Close() error {
	if p.closes.Add(1) == 1 {
		close(p.started)
	}
	<-p.release
	return p.closeErr
}

func TestLifecycleInventoryReportsActiveOwnersAndClearsAfterCompletion(t *testing.T) {
	s := newService(nil, nil)
	now := time.Now().UTC()
	s.sessionMu.Lock()
	s.sessions["lease-1"] = Session{ID: "lease-1", State: "held", ExpiresAt: now.Add(time.Minute)}
	s.activeCancels["lease-1"] = func() {}
	s.sessionMu.Unlock()
	s.mu.Lock()
	s.externalRecordings["recording-1"] = externalRecording{DeviceID: "device-1"}
	s.observerCancels["device-1"] = &observerJob{}
	s.pendingPairings["pairing-1"] = pendingPairing{deviceID: "device-2"}
	s.mu.Unlock()

	work, blockers, complete := s.LifecycleInventory(context.Background())
	if !complete {
		t.Fatal("known Device Control in-process owners should be completely inventoried")
	}
	for _, kind := range []string{"flows-and-cleanup", "durable-leases-and-recovery", "recordings", "observers", "pairing", "agents"} {
		if !hasLifecycleWork(work, kind) {
			t.Errorf("missing lifecycle owner %q in %+v", kind, work)
		}
	}
	if len(blockers) < 5 {
		t.Fatalf("active operation, lease, recording, observer and pairing should block: %+v", blockers)
	}

	s.sessionMu.Lock()
	delete(s.activeCancels, "lease-1")
	s.sessions["lease-1"] = Session{ID: "lease-1", State: "released", ExpiresAt: now}
	s.sessionMu.Unlock()
	s.mu.Lock()
	delete(s.externalRecordings, "recording-1")
	delete(s.observerCancels, "device-1")
	delete(s.pendingPairings, "pairing-1")
	s.mu.Unlock()

	work, blockers, complete = s.LifecycleInventory(context.Background())
	if !complete || len(blockers) != 0 {
		t.Fatalf("terminal owner state should have no blockers: complete=%v blockers=%+v", complete, blockers)
	}
	for _, item := range work {
		if item.GetStatus() != "idle" && item.GetStatus() != "completed" {
			t.Errorf("owner %q remains nonterminal after cleanup: %+v", item.GetKind(), item)
		}
	}
}

func TestQuiesceObserversWaitsForExitReceipt(t *testing.T) {
	s := newService(nil, nil)
	observerCtx, cancel := context.WithCancel(context.Background())
	job := &observerJob{cancel: cancel, done: make(chan struct{})}
	s.mu.Lock()
	s.observerCancels["device-1"] = job
	s.mu.Unlock()

	quiesced := make(chan error, 1)
	go func() { quiesced <- s.QuiesceObservers(context.Background()) }()
	select {
	case <-observerCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("quiesce did not request observer cancellation")
	}
	select {
	case err := <-quiesced:
		t.Fatalf("quiesce returned before the worker exit receipt: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	s.mu.Lock()
	if !s.observerCancels["device-1"].stopping {
		t.Fatal("observer inventory did not report stopping during cancellation")
	}
	delete(s.observerCancels, "device-1")
	s.mu.Unlock()
	close(job.done)
	select {
	case err := <-quiesced:
		if err != nil {
			t.Fatalf("quiesce returned error after observer exit: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("quiesce did not return after observer exit receipt")
	}
}

func TestQuiesceObserversPreservesBlockerOnIncompleteJob(t *testing.T) {
	s := newService(nil, nil)
	s.mu.Lock()
	s.observerCancels["device-1"] = &observerJob{cancel: func() {}}
	s.mu.Unlock()
	if err := s.QuiesceObservers(context.Background()); err == nil {
		t.Fatal("quiesce accepted an observer without a completion receipt")
	}
	_, blockers, complete := s.LifecycleInventory(context.Background())
	if !complete || len(blockers) == 0 {
		t.Fatalf("incomplete observer ownership must remain visible: complete=%v blockers=%+v", complete, blockers)
	}
	s.mu.Lock()
	s.startObserverLocked(devicedomain.Record{ID: "late-device", StrategyID: "polling-state"})
	_, admitted := s.observerCancels["late-device"]
	s.mu.Unlock()
	if admitted {
		t.Fatal("observer started after lifecycle quiescence began")
	}
}

func TestLeaseFinishRetainsActiveOwnerUntilOperationCleanup(t *testing.T) {
	s := newService(nil, nil)
	session := Session{ID: "lease-1", DeviceID: "device-1", State: "held", LeaseToken: "token", ExpiresAt: time.Now().Add(time.Minute)}
	s.sessionMu.Lock()
	s.sessions[session.ID] = session
	s.sessionMu.Unlock()

	operationCtx, cleanup, err := s.registerSessionOperation(context.Background(), session, session.DeviceID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Release(session.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-operationCtx.Done():
	default:
		t.Fatal("lease finish did not cancel the active operation")
	}
	s.sessionMu.Lock()
	_, stillOwned := s.activeCancels[session.ID]
	s.sessionMu.Unlock()
	if !stillOwned {
		t.Fatal("lease finish removed active ownership before operation cleanup unwound")
	}
	cleanup()
	cleanup() // The cleanup completion receipt must be idempotent.
	s.sessionMu.Lock()
	_, stillOwned = s.activeCancels[session.ID]
	s.sessionMu.Unlock()
	if stillOwned {
		t.Fatal("operation cleanup did not release active ownership")
	}
}

func TestExpiredPairingRemainsVisibleUntilCloseReceipt(t *testing.T) {
	s := newService(nil, nil)
	session := &lifecyclePairingSession{started: make(chan struct{}), release: make(chan struct{}), closeErr: errors.New("close uncertain")}
	s.mu.Lock()
	s.pendingPairings["pairing-expired"] = pendingPairing{deviceID: "tv", session: session, expiresAt: time.Now().Add(-time.Second)}
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		s.expirePairing("pairing-expired")
		close(done)
	}()
	<-session.started
	work, blockers, complete := s.LifecycleInventory(context.Background())
	if !complete || len(blockers) == 0 || lifecyclePairingStatus(work, "pairing/pairing-expired") != "closing" {
		t.Fatalf("pairing disappeared while Close was pending: complete=%v blockers=%+v work=%+v", complete, blockers, work)
	}
	if _, err := s.CompletePairDevice(context.Background(), "tv", "pairing-expired", []byte("835B64")); err == nil {
		t.Fatal("concurrent completion was allowed to take ownership of an expiring session")
	}
	if got := session.closes.Load(); got != 1 {
		t.Fatalf("Close called %d times during concurrent expiry/completion; want 1", got)
	}
	close(session.release)
	<-done

	work, blockers, complete = s.LifecycleInventory(context.Background())
	if !complete || len(blockers) == 0 || lifecyclePairingStatus(work, "pairing/pairing-expired") != "uncertain" {
		t.Fatalf("failed Close must remain an explicit lifecycle blocker: complete=%v blockers=%+v work=%+v", complete, blockers, work)
	}
	if _, err := s.CompletePairDevice(context.Background(), "tv", "pairing-expired", []byte("835B64")); err == nil {
		t.Fatal("an uncertain close must not be replayed as a pairing completion")
	}
	if got := session.closes.Load(); got != 1 {
		t.Fatalf("uncertain pairing Close was retried %d times; want one attempt", got)
	}
}

func TestExpiredPairingLeavesInventoryIdleOnlyAfterSuccessfulClose(t *testing.T) {
	s := newService(nil, nil)
	session := &lifecyclePairingSession{started: make(chan struct{}), release: make(chan struct{})}
	s.mu.Lock()
	s.pendingPairings["pairing-expired"] = pendingPairing{deviceID: "tv", session: session, expiresAt: time.Now().Add(-time.Second)}
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		s.expirePairing("pairing-expired")
		close(done)
	}()
	<-session.started
	work, blockers, complete := s.LifecycleInventory(context.Background())
	if !complete || len(blockers) == 0 || lifecyclePairingStatus(work, "pairing/pairing-expired") != "closing" {
		t.Fatalf("pairing disappeared while Close was pending: complete=%v blockers=%+v work=%+v", complete, blockers, work)
	}
	close(session.release)
	<-done

	work, blockers, complete = s.LifecycleInventory(context.Background())
	if !complete || len(blockers) != 0 || lifecyclePairingStatus(work, "pairing/pairing-expired") != "" {
		t.Fatalf("verified Close should clear the owner: complete=%v blockers=%+v work=%+v", complete, blockers, work)
	}
}

func lifecyclePairingStatus(work []*commonv1.LifecycleWorkItem, id string) string {
	for _, item := range work {
		if item.GetId() == id {
			return item.GetStatus()
		}
	}
	return ""
}

func hasLifecycleWork(work []*commonv1.LifecycleWorkItem, kind string) bool {
	for _, item := range work {
		if item.GetKind() == kind {
			return true
		}
	}
	return false
}
