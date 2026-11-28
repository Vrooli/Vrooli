package jobs_test

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/lifecycle"
	"github.com/vrooli/api-core/schedule"

	jobsH "image-tools/handlers/jobs"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/server"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	commonconnect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
	jobsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs/jobs_v1connect"
)

// The restart drain through the real maintenance fence: a client waiting on a
// job that is only waiting for capacity must not hold the replacement. The
// drain completes at once, the wait ends with an explanatory error, and the
// job is back on the queue for the next process.
func TestDrainDoesNotWaitForClientsOrJobsThatOnlyWait(t *testing.T) {
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(internaljobs.Schema)); err != nil {
		t.Fatal(err)
	}
	mgr := internaljobs.New(d, internaljobs.Config{
		Clock:     schedule.System(),
		Resumable: func(string) bool { return true },
		Admit: func(ctx context.Context, _ internaljobs.Job) (internaljobs.Admission, error) {
			internaljobs.AdmissionProgress(ctx)(95, "waiting for model smoke capacity")
			<-ctx.Done()
			return internaljobs.Admission{Context: ctx}, ctx.Err()
		},
		Runner: func(context.Context, internaljobs.Job, func(int, string)) (internaljobs.Result, error) {
			return internaljobs.Result{}, nil
		},
	})
	if err := mgr.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()

	provider, err := lifecycle.NewProvider("image-tools", "test-instance", lifecycle.Hooks{
		Quiesce: mgr.Quiesce,
		Resume:  mgr.Resume,
		Snapshot: func(context.Context) (lifecycle.Inventory, error) {
			return lifecycle.Inventory{Remaining: int32(len(mgr.Active())), Complete: true}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	api := server.New(server.Deps{Clock: schedule.System(), Logger: log.Default()}, jobsH.Module(mgr, nil, log.Default()))
	live := httptest.NewServer(provider.Mount(api.Handler()))
	defer live.Close()
	client := jobsconnect.NewJobsServiceClient(http.DefaultClient, live.URL)
	fence := commonconnect.NewLifecycleMaintenanceServiceClient(http.DefaultClient, live.URL)

	job, err := mgr.Submit(context.Background(), internaljobs.Spec{Operation: "edit_instruct", Payload: []byte(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if got, _ := mgr.Get(context.Background(), job.ID); got.State == internaljobs.StateAdmitting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("job never began admission")
		}
		time.Sleep(5 * time.Millisecond)
	}
	waitErr := make(chan error, 1)
	go func() {
		_, err := client.WaitJob(context.Background(), connect.NewRequest(&jobsv1.WaitJobRequest{Id: job.ID}))
		waitErr <- err
	}()
	time.Sleep(100 * time.Millisecond) // the wait is now an admitted request

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	started := time.Now()
	prepared, err := fence.Prepare(ctx, connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "restart-1", Reason: "test"}))
	if err != nil {
		t.Fatal(err)
	}
	standing := prepared.Msg.GetStanding()
	drained, err := fence.Drain(ctx, connect.NewRequest(&commonv1.LifecycleDrainRequest{
		OperationId: "restart-1", FenceToken: standing.GetFenceToken(), Revision: standing.GetRevision(), QuiesceComponents: true,
	}))
	if err != nil {
		t.Fatalf("drain must not wait for a job that only waits: %v", err)
	}
	if took := time.Since(started); took > 3*time.Second {
		t.Fatalf("drain took %s", took)
	}
	if !drained.Msg.GetStanding().GetComponentStopReady() {
		t.Fatalf("components must be ready to stop: %+v", drained.Msg.GetStanding())
	}
	select {
	case err := <-waitErr:
		if connect.CodeOf(err) != connect.CodeUnavailable {
			t.Fatalf("a wait ended by a restart must say so, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the wait on the job was never released")
	}
	got, err := mgr.Get(context.Background(), job.ID)
	if err != nil || got.State != internaljobs.StateQueued {
		t.Fatalf("the waiting job must be queued for the next process: %+v %v", got, err)
	}
}
