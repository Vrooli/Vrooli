package models

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	testutil "github.com/vrooli/cli-core/cliapptest"
	"github.com/vrooli/cli-core/cliutil"
	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
	jobsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs/jobs_v1connect"
)

type slowWaitJobsService struct {
	jobsconnect.UnimplementedJobsServiceHandler
	delay time.Duration
}

func (s slowWaitJobsService) WaitJob(context.Context, *connect.Request[jobsv1.WaitJobRequest]) (*connect.Response[jobsv1.WaitJobResponse], error) {
	time.Sleep(s.delay)
	return connect.NewResponse(&jobsv1.WaitJobResponse{Job: &jobsv1.Job{
		Id: "provision-job", State: jobsv1.JobState_JOB_STATE_SUCCEEDED,
	}}), nil
}

func TestWaitJobOutlivesOrdinaryCLIHTTPTimeout(t *testing.T) {
	mux := http.NewServeMux()
	path, handler := jobsconnect.NewJobsServiceHandler(slowWaitJobsService{delay: 30 * time.Millisecond})
	mux.Handle(path, handler)
	core := testutil.NewTestApp(t, mux)
	core.HTTPClient = cliutil.NewHTTPClient(cliutil.HTTPClientOptions{Timeout: time.Millisecond})

	job, err := newHandlers(core).waitJob("provision-job")
	if err != nil {
		t.Fatalf("waitJob should honor the owner terminal result instead of the ordinary HTTP timeout: %v", err)
	}
	if job.GetState() != jobsv1.JobState_JOB_STATE_SUCCEEDED {
		t.Fatalf("state=%s, want succeeded", job.GetState())
	}
}
