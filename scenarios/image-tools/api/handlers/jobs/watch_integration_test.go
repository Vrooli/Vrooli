package jobs_test

import (
	"context"
	"log"
	"testing"
	"time"

	"connectrpc.com/connect"
	apidb "github.com/vrooli/api-core/database"

	jobsH "image-tools/handlers/jobs"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/server"

	db "github.com/vrooli/api-core/databasetest"
	httpxtest "github.com/vrooli/api-core/servertest"

	"github.com/vrooli/api-core/schedule"

	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
	jobsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs/jobs_v1connect"
)

// TestWatchJobStreamsToTerminal exercises the full streaming path: a real
// durable Manager runs a job that emits progress and succeeds, and the
// Connect-RPC WatchJob server stream delivers events ending in a terminal one.
func TestWatchJobStreamsToTerminal(t *testing.T) {
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(internaljobs.Schema)); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}

	mgr := internaljobs.New(d, internaljobs.Config{
		Clock: schedule.System(),
		Runner: func(_ context.Context, job internaljobs.Job, emit func(int, string)) (internaljobs.Result, error) {
			emit(50, "halfway")
			return internaljobs.Result{Ref: "out/" + job.ID}, nil
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("start manager: %v", err)
	}
	defer mgr.Close()

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		jobsH.Module(mgr, log.Default()),
	)
	live := httpxtest.NewLiveServer(t, srv)
	client := jobsconnect.NewJobsServiceClient(live.Client, live.URL)

	job, err := mgr.Submit(context.Background(), internaljobs.Spec{Operation: "resize", Lane: internaljobs.LaneCPU})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	// Block-once wait proves durability + terminal verdict over the wire.
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer waitCancel()
	waited, err := client.WaitJob(waitCtx, connect.NewRequest(&jobsv1.WaitJobRequest{Id: job.ID}))
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if waited.Msg.Job.State != jobsv1.JobState_JOB_STATE_SUCCEEDED {
		t.Fatalf("state = %v, want SUCCEEDED", waited.Msg.Job.State)
	}
	if waited.Msg.Job.ResultRef != "out/"+job.ID {
		t.Fatalf("result_ref = %q", waited.Msg.Job.ResultRef)
	}

	// WatchJob on a finished job replays a terminal event then closes the stream.
	watchCtx, watchCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer watchCancel()
	stream, err := client.WatchJob(watchCtx, connect.NewRequest(&jobsv1.WatchJobRequest{Id: job.ID}))
	if err != nil {
		t.Fatalf("watch: %v", err)
	}
	var lastState jobsv1.JobState
	var got int
	for stream.Receive() {
		got++
		lastState = stream.Msg().State
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream err: %v", err)
	}
	if got == 0 {
		t.Fatal("expected at least one progress event")
	}
	if lastState != jobsv1.JobState_JOB_STATE_SUCCEEDED {
		t.Fatalf("last streamed state = %v, want SUCCEEDED", lastState)
	}
}

// TestWatchJobUnknownIsNotFound confirms the error mapping over the wire.
func TestWatchJobUnknownIsNotFound(t *testing.T) {
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(internaljobs.Schema)); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	mgr := internaljobs.New(d, internaljobs.Config{
		Clock: schedule.System(),
		Runner: func(context.Context, internaljobs.Job, func(int, string)) (internaljobs.Result, error) {
			return internaljobs.Result{}, nil
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer mgr.Close()

	srv := server.New(server.Deps{Clock: schedule.System(), Logger: log.Default()}, jobsH.Module(mgr, log.Default()))
	live := httpxtest.NewLiveServer(t, srv)
	client := jobsconnect.NewJobsServiceClient(live.Client, live.URL)

	stream, err := client.WatchJob(context.Background(), connect.NewRequest(&jobsv1.WatchJobRequest{Id: "missing"}))
	if err == nil {
		// Some Connect transports surface the error on first Receive.
		stream.Receive()
		err = stream.Err()
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("code = %v, want NotFound (err=%v)", connect.CodeOf(err), err)
	}
}
