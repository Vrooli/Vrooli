package facts

import (
	"context"
	"errors"
	"testing"
	"time"

	"code-facts/internal/indexcontrol"
	"connectrpc.com/connect"
	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"
)

type fakeIndexController struct{ calls []string }

type allowIndexAuthorizer struct{}

func (allowIndexAuthorizer) AuthorizeIndexControl(context.Context, string, string) error { return nil }

func (controller *fakeIndexController) Status(context.Context) (*factsv1.IndexStatus, error) {
	controller.calls = append(controller.calls, "status")
	return &factsv1.IndexStatus{ActiveGeneration: "g1", State: "ready", SearchDocuments: 42}, nil
}

func (controller *fakeIndexController) Reconcile(context.Context, string) (*factsv1.IndexControlResponse, error) {
	controller.calls = append(controller.calls, "reconcile")
	return &factsv1.IndexControlResponse{Message: "reconciled"}, nil
}

func (controller *fakeIndexController) Reindex(context.Context, string) (*factsv1.IndexControlResponse, error) {
	controller.calls = append(controller.calls, "reindex")
	return &factsv1.IndexControlResponse{Message: "reindexed", Job: &factsv1.IndexJob{Id: "job-1"}}, nil
}

type fakeJobStore struct{ job indexcontrol.Job }

func (store *fakeJobStore) Create(context.Context, indexcontrol.Job) error { return nil }
func (store *fakeJobStore) Update(context.Context, indexcontrol.Job) error { return nil }
func (store *fakeJobStore) Get(_ context.Context, id string) (indexcontrol.Job, error) {
	if id != store.job.ID {
		return indexcontrol.Job{}, errors.New("not found")
	}
	return store.job, nil
}
func (store *fakeJobStore) ListActive(context.Context) ([]indexcontrol.Job, error) { return nil, nil }
func (store *fakeJobStore) RequestCancel(context.Context, string, time.Time) error { return nil }
func (store *fakeJobStore) RecoverInterrupted(context.Context, time.Time) ([]indexcontrol.Job, error) {
	return nil, nil
}

func (controller *fakeIndexController) Cancel(context.Context, string) (*factsv1.IndexControlResponse, error) {
	controller.calls = append(controller.calls, "cancel")
	return &factsv1.IndexControlResponse{Message: "cancelled"}, nil
}

func (controller *fakeIndexController) Promote(context.Context, string) (*factsv1.IndexControlResponse, error) {
	controller.calls = append(controller.calls, "promote")
	return &factsv1.IndexControlResponse{Message: "promoted"}, nil
}

func (controller *fakeIndexController) Rollback(context.Context, string) (*factsv1.IndexControlResponse, error) {
	controller.calls = append(controller.calls, "rollback")
	return &factsv1.IndexControlResponse{Message: "rolled back"}, nil
}

func (controller *fakeIndexController) Cleanup(context.Context, bool) (*factsv1.IndexControlResponse, error) {
	controller.calls = append(controller.calls, "cleanup")
	return &factsv1.IndexControlResponse{Message: "cleaned"}, nil
}

func TestIndexControlsRequireConfirmationBeforeMutation(t *testing.T) {
	controller := &fakeIndexController{}
	handler := NewConnectHandler(Deps{Index: controller, IndexAuthorizer: allowIndexAuthorizer{}})
	if _, err := handler.Reindex(context.Background(), connect.NewRequest(&factsv1.ReindexRequest{Generation: "g2"})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unconfirmed reindex code=%s err=%v", connect.CodeOf(err), err)
	}
	if _, err := handler.PromoteIndexGeneration(context.Background(), connect.NewRequest(&factsv1.PromoteIndexGenerationRequest{Generation: "g2"})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unconfirmed promote code=%s err=%v", connect.CodeOf(err), err)
	}
	if _, err := handler.RollbackIndexGeneration(context.Background(), connect.NewRequest(&factsv1.RollbackIndexGenerationRequest{Generation: "g1"})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unconfirmed rollback code=%s err=%v", connect.CodeOf(err), err)
	}
	if _, err := handler.CleanupIndex(context.Background(), connect.NewRequest(&factsv1.CleanupIndexRequest{})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unconfirmed cleanup code=%s err=%v", connect.CodeOf(err), err)
	}
	if len(controller.calls) != 0 {
		t.Fatalf("unconfirmed operations reached controller: %v", controller.calls)
	}
}

func TestIndexMutationFailsClosedWithoutAuthorizer(t *testing.T) {
	controller := &fakeIndexController{}
	handler := NewConnectHandler(Deps{Index: controller})
	_, err := handler.ReconcileIndex(context.Background(), connect.NewRequest(&factsv1.ReconcileIndexRequest{}))
	if connect.CodeOf(err) != connect.CodePermissionDenied || len(controller.calls) != 0 {
		t.Fatalf("unauthorized mutation code=%s calls=%v err=%v", connect.CodeOf(err), controller.calls, err)
	}
}

func TestStaticIndexAuthorizerRequiresMatchingHeader(t *testing.T) {
	controller := &fakeIndexController{}
	handler := NewConnectHandler(Deps{Index: controller, IndexAuthorizer: NewStaticIndexAuthorizer("secret")})

	for _, presented := range []string{"", "wrong"} {
		req := connect.NewRequest(&factsv1.ReconcileIndexRequest{})
		req.Header().Set(IndexControlTokenHeader, presented)
		_, err := handler.ReconcileIndex(context.Background(), req)
		if connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Fatalf("token %q code=%s err=%v", presented, connect.CodeOf(err), err)
		}
	}
	req := connect.NewRequest(&factsv1.ReconcileIndexRequest{})
	req.Header().Set(IndexControlTokenHeader, "secret")
	if _, err := handler.ReconcileIndex(context.Background(), req); err != nil {
		t.Fatalf("matching token rejected: %v", err)
	}
	if len(controller.calls) != 1 || controller.calls[0] != "reconcile" {
		t.Fatalf("authorized calls=%v", controller.calls)
	}
}

func TestCompositeIndexAuthorizerAcceptsMintedLeafToken(t *testing.T) {
	authorizer := NewCompositeIndexAuthorizer("", func(token string) bool { return token == "minted" })
	if err := authorizer.AuthorizeIndexControl(context.Background(), "reindex", "minted"); err != nil {
		t.Fatalf("minted token rejected: %v", err)
	}
	if err := authorizer.AuthorizeIndexControl(context.Background(), "reindex", "wrong"); err == nil {
		t.Fatal("wrong token accepted")
	}
}

func TestSharedSearchControlMapsDryRunJobStatusAndCancellation(t *testing.T) {
	controller := &fakeIndexController{}
	jobs := &fakeJobStore{job: indexcontrol.Job{ID: "job-1", State: "running", Progress: 7, Total: 42}}
	handler := &SearchControlHandler{Controller: controller, Authorizer: allowIndexAuthorizer{}, Jobs: jobs}

	dryRun, err := handler.Reindex(context.Background(), connect.NewRequest(&controlv1.ReindexRequest{DryRun: true}))
	if err != nil || dryRun.Msg.GetPlannedUpserts() != 42 || !dryRun.Msg.GetDryRun() {
		t.Fatalf("dry-run response=%+v err=%v", dryRun, err)
	}
	started, err := handler.Reindex(context.Background(), connect.NewRequest(&controlv1.ReindexRequest{ShadowCollection: "g2"}))
	if err != nil || started.Msg.GetJobId() != "job-1" {
		t.Fatalf("reindex response=%+v err=%v", started, err)
	}
	status, err := handler.ReindexStatus(context.Background(), connect.NewRequest(&controlv1.ReindexStatusRequest{JobId: "job-1"}))
	if err != nil || status.Msg.GetState() != "running" || status.Msg.GetProcessed() != 7 {
		t.Fatalf("status response=%+v err=%v", status, err)
	}
	cancelled, err := handler.ReindexCancel(context.Background(), connect.NewRequest(&controlv1.ReindexCancelRequest{JobId: "job-1"}))
	if err != nil || !cancelled.Msg.GetCancelled() {
		t.Fatalf("cancel response=%+v err=%v", cancelled, err)
	}
}

func TestIndexControlsExposeTypedStatusAndConfirmedOperations(t *testing.T) {
	controller := &fakeIndexController{}
	handler := NewConnectHandler(Deps{Index: controller, IndexAuthorizer: allowIndexAuthorizer{}})
	status, err := handler.GetIndexStatus(context.Background(), connect.NewRequest(&factsv1.GetIndexStatusRequest{}))
	if err != nil || status.Msg.GetActiveGeneration() != "g1" {
		t.Fatalf("status mismatch: %+v err=%v", status, err)
	}
	if _, err := handler.ReconcileIndex(context.Background(), connect.NewRequest(&factsv1.ReconcileIndexRequest{})); err != nil {
		t.Fatal(err)
	}
	if _, err := handler.Reindex(context.Background(), connect.NewRequest(&factsv1.ReindexRequest{Generation: "g2", Confirmed: true})); err != nil {
		t.Fatal(err)
	}
	if _, err := handler.CancelIndexJob(context.Background(), connect.NewRequest(&factsv1.CancelIndexJobRequest{JobId: "job"})); err != nil {
		t.Fatal(err)
	}
	if _, err := handler.PromoteIndexGeneration(context.Background(), connect.NewRequest(&factsv1.PromoteIndexGenerationRequest{Generation: "g2", Confirmed: true})); err != nil {
		t.Fatal(err)
	}
	if _, err := handler.RollbackIndexGeneration(context.Background(), connect.NewRequest(&factsv1.RollbackIndexGenerationRequest{Generation: "g1", Confirmed: true})); err != nil {
		t.Fatal(err)
	}
	if _, err := handler.CleanupIndex(context.Background(), connect.NewRequest(&factsv1.CleanupIndexRequest{DryRun: true})); err != nil {
		t.Fatal(err)
	}
	want := []string{"status", "reconcile", "reindex", "cancel", "promote", "rollback", "cleanup"}
	if len(controller.calls) != len(want) {
		t.Fatalf("controller calls=%v want=%v", controller.calls, want)
	}
	for index := range want {
		if controller.calls[index] != want[index] {
			t.Fatalf("controller calls=%v want=%v", controller.calls, want)
		}
	}
}
