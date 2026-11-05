package handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/repository"
)

func newLifecycleTestProvider(t *testing.T, store lifecycleFenceStore, instance string) *providerLifecycle {
	t.Helper()
	provider, err := newProviderLifecycle(t.Context(), process.NewTracker(schedule.System()), store, "workspace-sandbox", "workspace-sandbox", instance)
	if err != nil {
		t.Fatalf("newProviderLifecycle: %v", err)
	}
	return provider
}

func newLifecycleTestRepo(t *testing.T) *repository.SandboxRepository {
	t.Helper()
	db := databasetest.NewSQLite(t)
	clk := schedule.System()
	if err := repository.EnsureSchema(t.Context(), db, clk); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	return repository.NewSandboxRepository(db, clk)
}

func TestProviderLifecycleFencePersistsAcrossReplacementAndRequiresExactResume(t *testing.T) {
	repo := newLifecycleTestRepo(t)
	first := newLifecycleTestProvider(t, repo, "instance-1")

	release, err := first.beginProcess(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := first.Prepare(t.Context(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "operation-1", Reason: "test replacement"}))
	if err != nil {
		t.Fatal(err)
	}
	standing := prepared.Msg.GetStanding()
	if !standing.GetAdmissionClosed() || standing.GetOperationId() != "operation-1" || standing.GetFenceToken() == "" || standing.GetRevision() <= 0 {
		t.Fatalf("prepared standing = %+v", standing)
	}
	if _, err := first.beginProcess(t.Context()); err == nil {
		t.Fatal("new admission was accepted after prepare")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	if _, err := first.Drain(ctx, connect.NewRequest(&commonv1.LifecycleDrainRequest{OperationId: "operation-1", FenceToken: standing.GetFenceToken(), Revision: standing.GetRevision()})); connect.CodeOf(err) != connect.CodeDeadlineExceeded {
		t.Fatalf("blocked drain code = %v err=%v", connect.CodeOf(err), err)
	}
	release()
	drained, err := first.Drain(t.Context(), connect.NewRequest(&commonv1.LifecycleDrainRequest{OperationId: "operation-1", FenceToken: standing.GetFenceToken(), Revision: standing.GetRevision()}))
	if err != nil || !drained.Msg.GetStanding().GetDrained() {
		t.Fatalf("drain standing = %+v err=%v", drained.Msg.GetStanding(), err)
	}

	// A new API instance loads the held operation from the authoritative row.
	second := newLifecycleTestProvider(t, repo, "instance-2")
	status, err := second.Status(t.Context(), connect.NewRequest(&commonv1.LifecycleStatusRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	held := status.Msg.GetStanding()
	if held.GetOperationId() != "operation-1" || held.GetFenceToken() != standing.GetFenceToken() || held.GetRevision() != standing.GetRevision() || !held.GetAdmissionClosed() {
		t.Fatalf("replacement status did not project persisted hold: %+v", held)
	}
	if held.GetProviderInstanceId() != "instance-2" || held.GetInstanceId() != "instance-2" {
		t.Fatalf("status instance identity = provider:%q instance:%q", held.GetProviderInstanceId(), held.GetInstanceId())
	}
	if _, err := second.beginProcess(t.Context()); err == nil {
		t.Fatal("replacement instance admitted a process before exact resume")
	}
	if _, err := second.Resume(t.Context(), connect.NewRequest(&commonv1.LifecycleResumeRequest{OperationId: "operation-1", FenceToken: "stale", Revision: standing.GetRevision()})); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("stale-token resume code = %v err=%v", connect.CodeOf(err), err)
	}
	if _, err := second.Resume(t.Context(), connect.NewRequest(&commonv1.LifecycleResumeRequest{OperationId: "operation-1", FenceToken: standing.GetFenceToken(), Revision: standing.GetRevision() + 1})); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("stale-revision resume code = %v err=%v", connect.CodeOf(err), err)
	}
	resumed, err := second.Resume(t.Context(), connect.NewRequest(&commonv1.LifecycleResumeRequest{OperationId: "operation-1", FenceToken: standing.GetFenceToken(), Revision: standing.GetRevision()}))
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Msg.GetStanding().GetAdmissionClosed() || resumed.Msg.GetStanding().GetOperationId() != "" {
		t.Fatalf("resume standing = %+v", resumed.Msg.GetStanding())
	}
	third := newLifecycleTestProvider(t, repo, "instance-3")
	if _, err := third.beginProcess(t.Context()); err != nil {
		t.Fatalf("admission remained closed after persisted resume: %v", err)
	}
}

func TestProviderLifecycleSameOperationPrepareIsIdempotent(t *testing.T) {
	repo := newLifecycleTestRepo(t)
	provider := newLifecycleTestProvider(t, repo, "instance-1")
	request := connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "operation-2", Reason: "retry"})
	first, err := provider.Prepare(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := provider.Prepare(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Msg.GetStanding().GetRevision() != second.Msg.GetStanding().GetRevision() || first.Msg.GetStanding().GetFenceToken() != second.Msg.GetStanding().GetFenceToken() {
		t.Fatalf("idempotent prepare changed fence: first=%+v second=%+v", first.Msg.GetStanding(), second.Msg.GetStanding())
	}
}

func TestProviderLifecycleStoreErrorsFailClosed(t *testing.T) {
	repo := newLifecycleTestRepo(t)
	store := &unavailableLifecycleStore{lifecycleFenceStore: repo}
	provider := newLifecycleTestProvider(t, store, "instance-1")
	store.unavailable = true
	if _, err := provider.beginProcess(t.Context()); err == nil {
		t.Fatal("process admission succeeded while durable fence state was unavailable")
	}
	if _, err := provider.Prepare(t.Context(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "operation-3", Reason: "test"})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("prepare store error code=%v err=%v", connect.CodeOf(err), err)
	}
	if _, err := newProviderLifecycle(t.Context(), process.NewTracker(schedule.System()), store, "workspace-sandbox", "workspace-sandbox", "instance-2"); err == nil {
		t.Fatal("provider startup succeeded while durable fence state was unavailable")
	}
}

func TestProviderLifecycleAmbiguousPrepareWriteKeepsDurableFenceClosed(t *testing.T) {
	repo := newLifecycleTestRepo(t)
	store := &writeThenErrorLifecycleStore{lifecycleFenceStore: repo}
	provider := newLifecycleTestProvider(t, store, "instance-1")
	if _, err := provider.Prepare(t.Context(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "operation-ambiguous", Reason: "lost acknowledgement"})); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("prepare code=%v err=%v", connect.CodeOf(err), err)
	}
	if _, err := provider.beginProcess(t.Context()); err == nil {
		t.Fatal("admission opened after ambiguous prepare write")
	}
	replacement := newLifecycleTestProvider(t, repo, "instance-2")
	status, err := replacement.Status(t.Context(), connect.NewRequest(&commonv1.LifecycleStatusRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !status.Msg.GetStanding().GetAdmissionClosed() || status.Msg.GetStanding().GetOperationId() != "operation-ambiguous" {
		t.Fatalf("ambiguous prepare was not retained as durable hold: %+v", status.Msg.GetStanding())
	}
}

type unavailableLifecycleStore struct {
	lifecycleFenceStore
	unavailable bool
}

func (s *unavailableLifecycleStore) GetProviderLifecycleFence(ctx context.Context) (repository.ProviderLifecycleFence, error) {
	if s.unavailable {
		return repository.ProviderLifecycleFence{}, errors.New("store unavailable")
	}
	return s.lifecycleFenceStore.GetProviderLifecycleFence(ctx)
}

func (s *unavailableLifecycleStore) CompareAndSwapProviderLifecycleFence(ctx context.Context, expected int64, next repository.ProviderLifecycleFence) error {
	if s.unavailable {
		return errors.New("store unavailable")
	}
	return s.lifecycleFenceStore.CompareAndSwapProviderLifecycleFence(ctx, expected, next)
}

type writeThenErrorLifecycleStore struct{ lifecycleFenceStore }

func (s *writeThenErrorLifecycleStore) CompareAndSwapProviderLifecycleFence(ctx context.Context, expected int64, next repository.ProviderLifecycleFence) error {
	if err := s.lifecycleFenceStore.CompareAndSwapProviderLifecycleFence(ctx, expected, next); err != nil {
		return err
	}
	return errors.New("write acknowledgement lost")
}

func TestProviderLifecycleEmergencyOverrideRequiresReason(t *testing.T) {
	provider := newLifecycleTestProvider(t, newLifecycleTestRepo(t), "instance-1")
	if _, err := provider.Prepare(t.Context(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "operation-4", Reason: "emergency", EmergencyOverride: true})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("missing override reason code = %v", connect.CodeOf(err))
	}
}
