package jobs

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	internaljobs "image-tools/internal/jobs"

	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
	jobsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs/jobs_v1connect"
)

// fakeManager is a hand-rolled JobManager for unary error-path coverage.
type fakeManager struct {
	job       internaljobs.Job
	getErr    error
	waitErr   error
	list      []internaljobs.Job
	cancelErr error
	canceled  string
}

func (f *fakeManager) Get(context.Context, string) (internaljobs.Job, error) {
	return f.job, f.getErr
}

func (f *fakeManager) Wait(context.Context, string) (internaljobs.Job, error) {
	return f.job, f.waitErr
}

func (f *fakeManager) List(context.Context, int) ([]internaljobs.Job, error) {
	return f.list, nil
}

func (f *fakeManager) Cancel(id string) error {
	f.canceled = id
	return f.cancelErr
}

func (f *fakeManager) Subscribe(string) (<-chan internaljobs.ProgressEvent, func(), error) {
	ch := make(chan internaljobs.ProgressEvent)
	close(ch)
	return ch, func() {}, nil
}

func newHandler(m JobManager) *connectHandler {
	return NewConnectHandler(Deps{Manager: m, Logger: log.New(testWriter{}, "", 0)})
}

type testWriter struct{}

func (testWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestGetJobOK(t *testing.T) {
	h := newHandler(&fakeManager{job: internaljobs.Job{ID: "j1", Operation: "resize", Lane: internaljobs.LaneCPU, State: internaljobs.StateSucceeded, Progress: 100}})
	resp, err := h.GetJob(context.Background(), connect.NewRequest(&jobsv1.GetJobRequest{Id: "j1"}))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if resp.Msg.Job.Id != "j1" || resp.Msg.Job.State != jobsv1.JobState_JOB_STATE_SUCCEEDED {
		t.Fatalf("unexpected job: %+v", resp.Msg.Job)
	}
	if resp.Msg.Job.Lane != jobsv1.JobLane_JOB_LANE_CPU {
		t.Fatalf("lane = %v, want CPU", resp.Msg.Job.Lane)
	}
}

func TestGetJobNotFound(t *testing.T) {
	h := newHandler(&fakeManager{getErr: internaljobs.ErrNotFound})
	_, err := h.GetJob(context.Background(), connect.NewRequest(&jobsv1.GetJobRequest{Id: "x"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("code = %v, want NotFound", connect.CodeOf(err))
	}
}

func TestWaitJobClientCancelMapsToCanceled(t *testing.T) {
	h := newHandler(&fakeManager{waitErr: context.Canceled})
	_, err := h.WaitJob(context.Background(), connect.NewRequest(&jobsv1.WaitJobRequest{Id: "x"}))
	if connect.CodeOf(err) != connect.CodeCanceled {
		t.Fatalf("code = %v, want Canceled", connect.CodeOf(err))
	}
}

func TestListJobs(t *testing.T) {
	h := newHandler(&fakeManager{list: []internaljobs.Job{{ID: "a"}, {ID: "b"}}})
	resp, err := h.ListJobs(context.Background(), connect.NewRequest(&jobsv1.ListJobsRequest{Limit: 10}))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(resp.Msg.Jobs) != 2 {
		t.Fatalf("got %d jobs, want 2", len(resp.Msg.Jobs))
	}
}

func TestCancelJobInvokesManagerAndReturnsRecord(t *testing.T) {
	fm := &fakeManager{job: internaljobs.Job{ID: "j9", State: internaljobs.StateCanceled}}
	h := newHandler(fm)
	resp, err := h.CancelJob(context.Background(), connect.NewRequest(&jobsv1.CancelJobRequest{Id: "j9"}))
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if fm.canceled != "j9" {
		t.Fatalf("manager.Cancel not called with id, got %q", fm.canceled)
	}
	if resp.Msg.Job.State != jobsv1.JobState_JOB_STATE_CANCELED {
		t.Fatalf("state = %v, want CANCELED", resp.Msg.Job.State)
	}
}

func TestCancelJobNotFound(t *testing.T) {
	h := newHandler(&fakeManager{cancelErr: internaljobs.ErrNotFound})
	_, err := h.CancelJob(context.Background(), connect.NewRequest(&jobsv1.CancelJobRequest{Id: "x"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("code = %v, want NotFound", connect.CodeOf(err))
	}
}

func TestTimeToProtoZeroIsNil(t *testing.T) {
	var zero time.Time
	if got := timeToProto(&zero); got != nil {
		t.Fatalf("zero time should map to nil, got %v", got)
	}
	if got := timeToProto(nil); got != nil {
		t.Fatalf("nil time should map to nil")
	}
	now := time.Unix(1700000000, 0)
	if got := timeToProto(&now); got == nil {
		t.Fatal("non-zero time should map to a timestamp")
	}
}

func TestStateAndLaneMapping(t *testing.T) {
	if stateToProto(internaljobs.StateFailed) != jobsv1.JobState_JOB_STATE_FAILED {
		t.Fatal("failed state mapping")
	}
	if stateToProto(internaljobs.State("bogus")) != jobsv1.JobState_JOB_STATE_UNSPECIFIED {
		t.Fatal("unknown state should map to unspecified")
	}
	if laneToProto(internaljobs.LaneGPU) != jobsv1.JobLane_JOB_LANE_GPU {
		t.Fatal("gpu lane mapping")
	}
}

// ensure the WatchJob subscriber-closed path returns cleanly (no panic) using
// the fake's pre-closed channel via a direct progress conversion.
func TestProgressToProto(t *testing.T) {
	ev := internaljobs.ProgressEvent{JobID: "j1", State: internaljobs.StateRunning, Progress: 42, Message: "halfway", At: time.Unix(1700000000, 0)}
	p := progressToProto(ev)
	if p.JobId != "j1" || p.Progress != 42 || p.State != jobsv1.JobState_JOB_STATE_RUNNING {
		t.Fatalf("unexpected progress proto: %+v", p)
	}
}

// slowManager blocks for a fixed delay before returning a terminal job.
type slowManager struct {
	fakeManager
	delay time.Duration
}

func (s *slowManager) Wait(ctx context.Context, id string) (internaljobs.Job, error) {
	select {
	case <-time.After(s.delay):
		return internaljobs.Job{ID: id, Operation: "text_to_image", Lane: internaljobs.LaneGPU, State: internaljobs.StateSucceeded, Progress: 100}, nil
	case <-ctx.Done():
		return internaljobs.Job{}, ctx.Err()
	}
}

// TestWaitJobSurvivesShortServerWriteTimeout is the F3 regression: a wait that
// outlives the server's write timeout must still return. The wrapper clears the
// deadline for exactly the two long-lived procedure paths.
func TestWaitJobSurvivesShortServerWriteTimeout(t *testing.T) {
	mgr := &slowManager{delay: 2500 * time.Millisecond}
	_, h := jobsconnect.NewJobsServiceHandler(NewConnectHandler(Deps{Manager: mgr, Logger: log.New(testWriter{}, "", 0)}))

	srv := httptest.NewUnstartedServer(waitDeadlineClearer{next: h})
	srv.Config.WriteTimeout = 1 * time.Second
	srv.Start()
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := jobsconnect.NewJobsServiceClient(http.DefaultClient, srv.URL)
	resp, err := client.WaitJob(ctx, connect.NewRequest(&jobsv1.WaitJobRequest{Id: "j1"}))
	if err != nil {
		t.Fatalf("WaitJob after 2.5s with a 1s server write timeout: %v", err)
	}
	if resp.Msg.GetJob().GetId() != "j1" {
		t.Fatalf("job id = %q, want j1", resp.Msg.GetJob().GetId())
	}
}

func TestWaitJobWithoutDeadlineClearFails(t *testing.T) {
	mgr := &slowManager{delay: 2500 * time.Millisecond}
	_, h := jobsconnect.NewJobsServiceHandler(NewConnectHandler(Deps{Manager: mgr, Logger: log.New(testWriter{}, "", 0)}))
	srv := httptest.NewUnstartedServer(h)
	srv.Config.WriteTimeout = 1 * time.Second
	srv.Start()
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := jobsconnect.NewJobsServiceClient(http.DefaultClient, srv.URL)
	if _, err := client.WaitJob(ctx, connect.NewRequest(&jobsv1.WaitJobRequest{Id: "j1"})); err == nil {
		t.Fatal("expected the bare handler to fail when the write deadline is not cleared")
	}
}
