package lifecycle

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	_ "modernc.org/sqlite"
)

func testProvider(t *testing.T) (*sql.DB, *Provider) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	p, err := New(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	return db, p
}

func TestQuiescenceRunsWhenOwnerWorkHasNoManagedComponent(t *testing.T) {
	_, p := testProvider(t)
	ctx := context.Background()
	active := true
	quiesceCalled := false
	p.SetOwnerInventory(func(context.Context) ([]*commonv1.LifecycleWorkItem, []*commonv1.LifecycleBlocker, bool) {
		if active {
			return []*commonv1.LifecycleWorkItem{{Id: "observer/cast", Kind: "observer", Status: "active"}}, []*commonv1.LifecycleBlocker{{Code: "observer_pending", Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_BLOCKING, Observed: true}}, true
		}
		return []*commonv1.LifecycleWorkItem{{Id: "owner/observers", Kind: "observer", Status: "idle"}}, nil, true
	})
	p.SetComponentMaintenance(func(context.Context, map[string]string) ([]*commonv1.LifecycleManagedComponent, []*commonv1.LifecycleWorkItem, []*commonv1.LifecycleBlocker, bool) {
		return nil, nil, nil, true
	}, func(context.Context) error {
		quiesceCalled = true
		active = false
		return nil
	})
	prepared, err := p.Prepare(ctx, connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "op", Reason: "replace"}))
	if err != nil {
		t.Fatal(err)
	}
	standing := prepared.Msg.GetStanding()
	if !standing.GetInventoryComplete() || standing.GetDrained() || !standing.GetQuiescencePending() || len(standing.GetManagedComponents()) != 0 {
		t.Fatalf("prepared standing lost zero-component quiescence requirement: %+v", standing)
	}
	quiesced, err := p.Drain(ctx, connect.NewRequest(&commonv1.LifecycleDrainRequest{OperationId: "op", FenceToken: standing.GetFenceToken(), Revision: standing.GetRevision(), QuiesceComponents: true}))
	if err != nil {
		t.Fatal(err)
	}
	result := quiesced.Msg.GetStanding()
	if !quiesceCalled || !result.GetComponentStopReady() || result.GetQuiescencePending() || !result.GetDrained() || len(result.GetManagedComponents()) != 0 {
		t.Fatalf("owner work was not quiesced without OS components: %+v", result)
	}
}

func TestManagedComponentQuiescenceAndExitStayBehindSameClosedFence(t *testing.T) {
	_, p := testProvider(t)
	ctx := context.Background()
	quiesced := false
	p.SetOwnerInventory(func(context.Context) ([]*commonv1.LifecycleWorkItem, []*commonv1.LifecycleBlocker, bool) {
		return nil, nil, true
	})
	p.SetComponentMaintenance(func(_ context.Context, exits map[string]string) ([]*commonv1.LifecycleManagedComponent, []*commonv1.LifecycleWorkItem, []*commonv1.LifecycleBlocker, bool) {
		components := []*commonv1.LifecycleManagedComponent{{Name: "desktop-helper", Generation: "generation"}}
		if exits["desktop-helper"] == "generation" {
			return components, nil, nil, true
		}
		return components, nil, []*commonv1.LifecycleBlocker{{Code: "component_exit_unobserved", Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN}}, true
	}, func(context.Context) error { quiesced = true; return nil })
	prepared, err := p.Prepare(ctx, connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "op", Reason: "replace"}))
	if err != nil {
		t.Fatal(err)
	}
	s := prepared.Msg.Standing
	if !s.InventoryComplete || s.Drained || s.ComponentStopReady {
		t.Fatalf("enumeration confused with readiness: %+v", s)
	}
	request := func(gen string, quiesce bool) *connect.Request[commonv1.LifecycleDrainRequest] {
		m := &commonv1.LifecycleDrainRequest{OperationId: "op", FenceToken: s.FenceToken, Revision: s.Revision, QuiesceComponents: quiesce}
		if gen != "" {
			m.ComponentExits = []*commonv1.LifecycleManagedComponent{{Name: "desktop-helper", Generation: gen}}
		}
		return connect.NewRequest(m)
	}
	if _, err = p.Drain(ctx, request("generation", false)); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("exit accepted before quiescence: %v", err)
	}
	q, err := p.Drain(ctx, request("", true))
	if err != nil {
		t.Fatal(err)
	}
	if !quiesced || !q.Msg.Standing.ComponentStopReady || q.Msg.Standing.Drained || !q.Msg.Standing.AdmissionClosed {
		t.Fatalf("unsafe quiescence: %+v", q.Msg.Standing)
	}
	if _, err = p.Drain(ctx, request("other", false)); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("wrong generation accepted: %v", err)
	}
	missingProofCtx, cancelMissingProof := context.WithTimeout(ctx, time.Millisecond)
	_, missingProofErr := p.Drain(missingProofCtx, request("", false))
	cancelMissingProof()
	if connect.CodeOf(missingProofErr) != connect.CodeDeadlineExceeded {
		t.Fatalf("missing owner completion became drained: %v", missingProofErr)
	}
	stillClosed, err := p.Status(ctx, connect.NewRequest(&commonv1.LifecycleStatusRequest{OperationId: "op", FenceToken: s.FenceToken, Revision: s.Revision}))
	if err != nil || !stillClosed.Msg.Standing.AdmissionClosed || stillClosed.Msg.Standing.Drained {
		t.Fatalf("missing stop proof lost closed fence: %v", err)
	}
	wrong := request("generation", false)
	wrong.Msg.Revision++
	if _, err = p.Drain(ctx, wrong); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("wrong revision accepted: %v", err)
	}
	final, err := p.Drain(ctx, request("generation", false))
	if err != nil {
		t.Fatal(err)
	}
	if !final.Msg.Standing.Drained || !final.Msg.Standing.AdmissionClosed || final.Msg.Standing.FenceToken != s.FenceToken || final.Msg.Standing.Revision != s.Revision {
		t.Fatalf("unsafe final drain: %+v", final.Msg.Standing)
	}
	if _, err = p.admit(ctx); err == nil {
		t.Fatal("component completion reopened ordinary admission")
	}
	if _, err = p.Resume(ctx, connect.NewRequest(&commonv1.LifecycleResumeRequest{OperationId: "op", FenceToken: s.FenceToken, Revision: s.Revision})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("quiesced instance reopened admission without replacement: %v", err)
	}
	status, err := p.Status(ctx, connect.NewRequest(&commonv1.LifecycleStatusRequest{OperationId: "op", FenceToken: s.FenceToken, Revision: s.Revision}))
	if err != nil || !status.Msg.Standing.AdmissionClosed {
		t.Fatalf("maintenance traffic unavailable after close: %v", err)
	}
	adopted, err := p.Prepare(ctx, connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "adopted", Reason: "recovery", EmergencyOverride: true, OverrideReason: "operator recovery"}))
	if err != nil {
		t.Fatal(err)
	}
	a := adopted.Msg.Standing
	if _, err = p.Resume(ctx, connect.NewRequest(&commonv1.LifecycleResumeRequest{OperationId: a.OperationId, FenceToken: a.FenceToken, Revision: a.Revision})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("adopted fence reopened an already stopped owner: %v", err)
	}
}

func TestComponentQuiescenceDoesNotHideIncompleteOwnerInventory(t *testing.T) {
	_, p := testProvider(t)
	p.SetOwnerInventory(func(context.Context) ([]*commonv1.LifecycleWorkItem, []*commonv1.LifecycleBlocker, bool) {
		return nil, nil, false
	})
	p.SetComponentMaintenance(func(context.Context, map[string]string) ([]*commonv1.LifecycleManagedComponent, []*commonv1.LifecycleWorkItem, []*commonv1.LifecycleBlocker, bool) {
		return []*commonv1.LifecycleManagedComponent{{Name: "desktop-helper", Generation: "g"}}, nil, nil, true
	}, func(context.Context) error { return nil })
	prepared, err := p.Prepare(context.Background(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "op", Reason: "replace"}))
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Msg.Standing.InventoryComplete {
		t.Fatal("component enumeration hid missing owner inventory")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	s := prepared.Msg.Standing
	_, err = p.Drain(ctx, connect.NewRequest(&commonv1.LifecycleDrainRequest{OperationId: "op", FenceToken: s.FenceToken, Revision: s.Revision, QuiesceComponents: true}))
	if connect.CodeOf(err) != connect.CodeDeadlineExceeded {
		t.Fatalf("incomplete inventory quiesced: %v", err)
	}
}

func TestLifecycleDurableFenceRequiresExactResumeIdentity(t *testing.T) {
	db, p := testProvider(t)
	ctx := context.Background()
	prepared, err := p.Prepare(ctx, connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "op-1", Reason: "replacement"}))
	if err != nil {
		t.Fatal(err)
	}
	a := prepared.Msg.GetStanding()
	if !a.GetAdmissionClosed() || a.GetInventoryComplete() || a.GetDrained() {
		t.Fatalf("unsafe standing: %+v", a)
	}
	p2, err := New(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	status, err := p2.Status(ctx, connect.NewRequest(&commonv1.LifecycleStatusRequest{OperationId: "op-1", FenceToken: a.GetFenceToken(), Revision: a.GetRevision()}))
	if err != nil {
		t.Fatal(err)
	}
	if status.Msg.GetStanding().GetProviderInstanceId() != a.GetProviderInstanceId() {
		t.Fatal("provider identity changed across reconstruction")
	}
	if _, err = p2.Resume(ctx, connect.NewRequest(&commonv1.LifecycleResumeRequest{OperationId: "op-1", FenceToken: "wrong", Revision: a.GetRevision()})); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("wrong token accepted: %v", err)
	}
	resumed, err := p2.Resume(ctx, connect.NewRequest(&commonv1.LifecycleResumeRequest{OperationId: "op-1", FenceToken: a.GetFenceToken(), Revision: a.GetRevision()}))
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Msg.GetStanding().GetAdmissionClosed() || resumed.Msg.GetStanding().GetRevision() != a.GetRevision()+1 {
		t.Fatalf("unexpected resume: %+v", resumed.Msg.GetStanding())
	}
}

func TestAdmissionMiddlewareClosesEveryIngressExceptLifecycleAndHealth(t *testing.T) {
	_, p := testProvider(t)
	ctx := context.Background()
	if _, err := p.Prepare(ctx, connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "op", Reason: "replacement"})); err != nil {
		t.Fatal(err)
	}
	called := 0
	h := p.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called++; w.WriteHeader(http.StatusNoContent) }))
	for _, path := range []string{"/api/v1/devices", "/api/v1/flows/run", "/api/v1/desktop/session", "/some/unregistered/route"} {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("%s admitted: %d", path, w.Code)
		}
	}
	r := httptest.NewRequest(http.MethodPost, "/common.v1.LifecycleMaintenanceService/Status", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if called != 1 || w.Code != http.StatusNoContent {
		t.Fatalf("maintenance route not admitted: called=%d status=%d", called, w.Code)
	}
	// A replacement must report healthy behind its durable closed fence so the
	// control plane can resume it; probes are served without being counted.
	for _, path := range []string{"/health", "/api/v1/health"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusNoContent {
			t.Errorf("%s was refused while admission is closed: %d", path, w.Code)
		}
	}
	if p.standing().GetAdmitting() != 0 {
		t.Fatal("health probes must not be counted as admitted work")
	}
}

func TestPrepareIsIdempotentForSameOperation(t *testing.T) {
	_, p := testProvider(t)
	ctx := context.Background()
	req := connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "op", Reason: "replacement"})
	one, err := p.Prepare(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	two, err := p.Prepare(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if one.Msg.GetStanding().GetFenceToken() != two.Msg.GetStanding().GetFenceToken() || one.Msg.GetStanding().GetRevision() != two.Msg.GetStanding().GetRevision() {
		t.Fatal("retry changed fence identity")
	}
}

func TestStandingTracksOwnerCompletionAndKeepsUnknownDesktopBlocking(t *testing.T) {
	_, p := testProvider(t)
	active := true
	p.SetOwnerInventory(func(context.Context) ([]*commonv1.LifecycleWorkItem, []*commonv1.LifecycleBlocker, bool) {
		if active {
			return []*commonv1.LifecycleWorkItem{{Id: "recording/test", Kind: "recordings", Status: "active"}}, []*commonv1.LifecycleBlocker{{Code: "owner_work_pending", Message: "recording stop not verified", Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_BLOCKING}}, true
		}
		return []*commonv1.LifecycleWorkItem{{Id: "owner/recordings", Kind: "recordings", Status: "idle"}}, nil, true
	})
	prepared, err := p.Prepare(context.Background(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "op", Reason: "replacement"}))
	if err != nil {
		t.Fatal(err)
	}
	standing := prepared.Msg.GetStanding()
	if standing.GetInventoryComplete() || standing.GetDrained() {
		t.Fatal("active recording or unknown desktop owners were reported drained")
	}
	if !hasLifecycleBlocker(standing, "owner_work_pending") || !hasLifecycleBlocker(standing, "owner_inventory_unavailable") {
		t.Fatalf("expected live recording and unknown desktop blockers, got %+v", standing.GetBlockers())
	}

	active = false
	status, err := p.Status(context.Background(), connect.NewRequest(&commonv1.LifecycleStatusRequest{OperationId: "op", FenceToken: standing.GetFenceToken(), Revision: standing.GetRevision()}))
	if err != nil {
		t.Fatal(err)
	}
	standing = status.Msg.GetStanding()
	if hasLifecycleBlocker(standing, "owner_work_pending") {
		t.Fatal("completed recording still blocks lifecycle standing")
	}
	if standing.GetInventoryComplete() || standing.GetDrained() || !hasLifecycleBlocker(standing, "owner_inventory_unavailable") {
		t.Fatal("unknown desktop owners must continue to block inventory completeness and drain")
	}
}

func TestPrepareEmergencyOverrideRequiresReasonAndAdoptsClosedFence(t *testing.T) {
	_, p := testProvider(t)
	ctx := context.Background()
	first, err := p.Prepare(ctx, connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "first", Reason: "replacement"}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Prepare(ctx, connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "second", Reason: "emergency", EmergencyOverride: true})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("emergency override without reason accepted: %v", err)
	}
	second, err := p.Prepare(ctx, connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "second", Reason: "emergency", EmergencyOverride: true, OverrideReason: "operator decision"}))
	if err != nil {
		t.Fatal(err)
	}
	if second.Msg.GetStanding().GetOperationId() != "second" || second.Msg.GetStanding().GetRevision() <= first.Msg.GetStanding().GetRevision() || second.Msg.GetStanding().GetFenceToken() == first.Msg.GetStanding().GetFenceToken() {
		t.Fatalf("override did not adopt a fresh exact fence identity: %s", fmt.Sprint(second.Msg.GetStanding()))
	}
}

func hasLifecycleBlocker(standing *commonv1.LifecycleStanding, code string) bool {
	for _, blocker := range standing.GetBlockers() {
		if blocker.GetCode() == code {
			return true
		}
	}
	return false
}

// A provider that has no work left but still owes owner quiescence must not
// claim to be drained: the control plane rejects a drained standing that also
// reports pending quiescence, which stranded a replacement behind a closed fence.
func TestIdleOwnersWithQuiescenceOwedAreNotDrainedUntilQuiesced(t *testing.T) {
	_, p := testProvider(t)
	ctx := context.Background()
	p.SetOwnerInventory(func(context.Context) ([]*commonv1.LifecycleWorkItem, []*commonv1.LifecycleBlocker, bool) {
		return []*commonv1.LifecycleWorkItem{{Id: "owner/observers", Kind: "observers", Status: "idle"}}, nil, true
	})
	p.SetComponentMaintenance(func(context.Context, map[string]string) ([]*commonv1.LifecycleManagedComponent, []*commonv1.LifecycleWorkItem, []*commonv1.LifecycleBlocker, bool) {
		return nil, nil, nil, true
	}, func(context.Context) error { return nil })

	prepared, err := p.Prepare(ctx, connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "op", Reason: "replace"}))
	if err != nil {
		t.Fatal(err)
	}
	standing := prepared.Msg.GetStanding()
	if standing.GetDrained() || !standing.GetQuiescencePending() {
		t.Fatalf("idle owners with owed quiescence reported drained=%v quiescencePending=%v", standing.GetDrained(), standing.GetQuiescencePending())
	}
	if standing.GetFencePersistence() != commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_OPERATION_DURABLE {
		t.Fatalf("the DB-backed fence must declare durable persistence so a replacement can resume it: %v", standing.GetFencePersistence())
	}
	quiesced, err := p.Drain(ctx, connect.NewRequest(&commonv1.LifecycleDrainRequest{OperationId: "op", FenceToken: standing.GetFenceToken(), Revision: standing.GetRevision(), QuiesceComponents: true}))
	if err != nil {
		t.Fatal(err)
	}
	if result := quiesced.Msg.GetStanding(); !result.GetDrained() || result.GetQuiescencePending() {
		t.Fatalf("quiescing did not complete the drain: %+v", result)
	}
}
