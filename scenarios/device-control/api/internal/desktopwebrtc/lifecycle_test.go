package desktopwebrtc

import (
	"context"
	"errors"
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
)

func TestLifecycleInventoryTracksActivePendingAndCompletedSessions(t *testing.T) {
	p := &fakeProvider{ready: desktopv1.ReadinessState_READINESS_STATE_READY}
	m := NewManager(p)
	s, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: surface(), DisplayId: "main", Control: true})
	if err != nil {
		t.Fatal(err)
	}
	work, blockers, complete := m.LifecycleInventory(context.Background())
	if !complete || len(work) != 1 || work[0].Status != "active" || len(blockers) != 1 {
		t.Fatalf("active inventory: work=%v blockers=%v complete=%v", work, blockers, complete)
	}

	request := &desktopv1.CloseSessionRequest{Session: s.Ref}
	m.mu.Lock()
	state := m.sessions[s.Ref.SessionId]
	state.revoked, state.cleanupPending = true, true
	state.captureDone = make(chan struct{})
	m.mu.Unlock()
	work, blockers, complete = m.LifecycleInventory(context.Background())
	if !complete || work[0].Status != "stopping" || len(blockers) != 1 {
		t.Fatalf("pending inventory: work=%v blockers=%v complete=%v", work, blockers, complete)
	}

	close(state.captureDone)
	if _, err := m.Close(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	work, blockers, complete = m.LifecycleInventory(context.Background())
	if !complete || work[0].Status != "completed" || len(blockers) != 0 {
		t.Fatalf("completed inventory: work=%v blockers=%v complete=%v", work, blockers, complete)
	}
}

func TestLifecycleInventoryKeepsFailedProviderCloseUnknown(t *testing.T) {
	p := &fakeProvider{ready: desktopv1.ReadinessState_READINESS_STATE_READY, closeErr: errors.New("provider close failed")}
	m := NewManager(p)
	s, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: surface(), DisplayId: "main", Control: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Close(context.Background(), &desktopv1.CloseSessionRequest{Session: s.Ref}); err == nil {
		t.Fatal("expected provider close failure")
	}
	work, blockers, complete := m.LifecycleInventory(context.Background())
	if !complete || work[0].Status != "uncertain" || len(blockers) != 1 || blockers[0].Disposition != commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN {
		t.Fatalf("failed cleanup inventory: work=%v blockers=%v complete=%v", work, blockers, complete)
	}
}
