package orchestrator

import (
	"context"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/lifecycle"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	commonconnect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
	"google.golang.org/protobuf/types/known/durationpb"
	"storage-manager/internal/cleanup"
	"storage-manager/internal/policy"
	"storage-manager/internal/providers"
	cleanupfakes "storage-manager/internal/testutil/cleanup"
)

func TestLifecycleTracksDetachedCensusAndFencesAdmission(t *testing.T) {
	p := &blockingProvider{started: make(chan struct{}), release: make(chan struct{})}
	registry, err := providers.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(registry, NewMemoryStore(), cleanupfakes.Clock{Time: time.Now()})
	if err := svc.store.SavePolicy(context.Background(), Policy{Version: "test", Profile: policy.ProfileConservative, Providers: map[string]cleanup.ProviderPolicy{"blocking": {Enabled: true, ApprovalMode: cleanup.ApprovalModeNone}}}); err != nil {
		t.Fatal(err)
	}
	id, err := svc.StartCensus(context.Background(), cleanup.ObservationScope{})
	if err != nil {
		t.Fatal(err)
	}
	<-p.started
	inv, err := svc.Snapshot(context.Background())
	if err != nil || !inv.Complete || inv.Remaining != 1 || len(inv.Work) != 1 || inv.Work[0].Id != id || inv.Work[0].Kind != "census" {
		t.Fatalf("inventory: %+v, %v", inv, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := svc.Quiesce(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("quiesce: %v", err)
	}
	if _, err := svc.StartCensus(context.Background(), cleanup.ObservationScope{}); !errors.Is(err, ErrLifecycleQuiesced) {
		t.Fatalf("census admitted during fence: %v", err)
	}
	if _, err := svc.StartRecovery(context.Background(), "test", "/", 50, 1, 10, true); !errors.Is(err, ErrLifecycleQuiesced) {
		t.Fatalf("recovery admitted during fence: %v", err)
	}
	drained := make(chan error, 1)
	go func() { drained <- svc.Quiesce(context.Background()) }()
	select {
	case err := <-drained:
		t.Fatalf("drained before worker completed: %v", err)
	default:
	}
	close(p.release)
	if err := <-drained; err != nil {
		t.Fatal(err)
	}
	inv, err = svc.Snapshot(context.Background())
	if err != nil || inv.Remaining != 0 {
		t.Fatalf("drained inventory: %+v, %v", inv, err)
	}
	if err := svc.Resume(context.Background()); err != nil {
		t.Fatal(err)
	}
	newID, err := svc.StartCensus(context.Background(), cleanup.ObservationScope{})
	if err != nil {
		t.Fatalf("admission not reopened: %v", err)
	}
	if _, err := svc.WaitCensus(context.Background(), newID); err != nil {
		t.Fatal(err)
	}
}

func TestLifecycleTransportDrainsDetachedCensus(t *testing.T) {
	p := &blockingProvider{started: make(chan struct{}), release: make(chan struct{})}
	svc := newPressureService(t, p)
	id, err := svc.StartCensus(context.Background(), cleanup.ObservationScope{})
	if err != nil {
		t.Fatal(err)
	}
	<-p.started
	provider, err := lifecycle.NewProvider("storage-manager", "test-instance", lifecycle.Hooks{Snapshot: svc.Snapshot, Quiesce: svc.Quiesce, Resume: svc.Resume})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(provider.Mount(nil))
	defer server.Close()
	client := commonconnect.NewLifecycleMaintenanceServiceClient(server.Client(), server.URL)
	prepared, err := client.Prepare(context.Background(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "op", Reason: "test"}))
	if err != nil {
		t.Fatal(err)
	}
	standing := prepared.Msg.Standing
	if standing.Remaining != 1 || len(standing.Work) != 1 || standing.Work[0].Id != id {
		t.Fatalf("prepared inventory: %+v", standing)
	}
	drain := &commonv1.LifecycleDrainRequest{OperationId: "op", FenceToken: standing.FenceToken, Revision: standing.Revision, QuiesceComponents: true, Timeout: durationpb.New(20 * time.Millisecond)}
	if _, err := client.Drain(context.Background(), connect.NewRequest(drain)); err == nil {
		t.Fatal("drain succeeded while worker blocked")
	}
	if _, err := svc.StartCensus(context.Background(), cleanup.ObservationScope{}); !errors.Is(err, ErrLifecycleQuiesced) {
		t.Fatalf("admission after timeout: %v", err)
	}
	close(p.release)
	if _, err := svc.WaitCensus(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	drain.Timeout = durationpb.New(time.Second)
	drained, err := client.Drain(context.Background(), connect.NewRequest(drain))
	if err != nil || !drained.Msg.Standing.Drained {
		t.Fatalf("retry drain: %+v %v", drained, err)
	}
	if _, err := client.Resume(context.Background(), connect.NewRequest(&commonv1.LifecycleResumeRequest{OperationId: "op", FenceToken: standing.FenceToken, Revision: standing.Revision})); err != nil {
		t.Fatal(err)
	}
	id, err = svc.StartCensus(context.Background(), cleanup.ObservationScope{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.WaitCensus(context.Background(), id); err != nil {
		t.Fatal(err)
	}
}

func TestLifecycleAdmissionRaceAccountsForAcceptedWork(t *testing.T) {
	for i := 0; i < 100; i++ {
		svc := NewService(nil, nil, nil)
		started := make(chan error, 1)
		go func() { started <- svc.beginLifecycleWork("run", "recovery") }()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := svc.Quiesce(ctx)
		admission := <-started
		inv, snapErr := svc.Snapshot(context.Background())
		if snapErr != nil {
			t.Fatal(snapErr)
		}
		if admission == nil {
			if !errors.Is(err, context.Canceled) || inv.Remaining != 1 || inv.Work[0].Id != "run" {
				t.Fatalf("accepted work missing: quiesce=%v inventory=%+v", err, inv)
			}
			svc.endLifecycleWork("run")
		} else if !errors.Is(admission, ErrLifecycleQuiesced) || err != nil || inv.Remaining != 0 {
			t.Fatalf("rejected work mismatch: admission=%v quiesce=%v inventory=%+v", admission, err, inv)
		}
	}
}

func TestLifecycleTracksDetachedRecovery(t *testing.T) {
	p := &blockingProvider{started: make(chan struct{}), release: make(chan struct{})}
	svc := newPressureService(t, p)
	svc.SetRecoveryLockPath(filepath.Join(t.TempDir(), "recovery.lock"))
	svc.SetEventPublisher(nil)
	run, err := svc.StartRecovery(context.Background(), "test", "/", 99, 1, 10, true)
	if err != nil {
		t.Fatal(err)
	}
	<-p.started
	inv, err := svc.Snapshot(context.Background())
	if err != nil || inv.Remaining != 1 || len(inv.Work) != 1 || inv.Work[0].Id != run.ID || inv.Work[0].Kind != "recovery" {
		t.Fatalf("recovery inventory: %+v %v", inv, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := svc.Quiesce(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("quiesce: %v", err)
	}
	close(p.release)
	if err := svc.Quiesce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.WaitRecovery(context.Background(), run.ID); err != nil {
		t.Fatal(err)
	}
	inv, err = svc.Snapshot(context.Background())
	if err != nil || inv.Remaining != 0 {
		t.Fatalf("completed recovery inventory: %+v %v", inv, err)
	}
}
