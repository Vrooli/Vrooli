package ai

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

type slowAIJobsService struct {
	jobsconnect.UnimplementedJobsServiceHandler
	delay time.Duration
}

func (s slowAIJobsService) WaitJob(context.Context, *connect.Request[jobsv1.WaitJobRequest]) (*connect.Response[jobsv1.WaitJobResponse], error) {
	time.Sleep(s.delay)
	return connect.NewResponse(&jobsv1.WaitJobResponse{Job: &jobsv1.Job{
		Id: "edit-job", State: jobsv1.JobState_JOB_STATE_SUCCEEDED,
	}}), nil
}

func TestAIWaitJobOutlivesOrdinaryCLIHTTPTimeout(t *testing.T) {
	mux := http.NewServeMux()
	path, handler := jobsconnect.NewJobsServiceHandler(slowAIJobsService{delay: 30 * time.Millisecond})
	mux.Handle(path, handler)
	core := testutil.NewTestApp(t, mux)
	core.HTTPClient = cliutil.NewHTTPClient(cliutil.HTTPClientOptions{Timeout: time.Millisecond})

	job, written, err := waitAndDownload(core, "edit-job", "")
	if err != nil {
		t.Fatalf("WaitJob should await the terminal result beyond the ordinary HTTP timeout: %v", err)
	}
	if job.GetState() != jobsv1.JobState_JOB_STATE_SUCCEEDED || written != "" {
		t.Fatalf("result = state:%s path:%q, want succeeded and no download path", job.GetState(), written)
	}
}
