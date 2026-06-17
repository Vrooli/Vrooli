package jobs

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
	jobsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs/jobs_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client jobsconnect.JobsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: jobsconnect.NewJobsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetJob(context.Background(), connect.NewRequest(&jobsv1.GetJobRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get job %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Job == nil {
		return fmt.Errorf("server returned no job")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Job %s is %s.", resp.Msg.Job.Id, jobStateLabel(resp.Msg.Job.State))},
		ResultsHeading: "Job",
		Results:        []string{formatJob(resp.Msg.Job)},
	})
}

// wait is the block-once verb: the server blocks until the job is terminal, then
// returns it. No polling — a single round trip.
func (h *handlers) wait(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.WaitJob(context.Background(), connect.NewRequest(&jobsv1.WaitJobRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("wait for job %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Job == nil {
		return fmt.Errorf("server returned no job")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Job %s finished: %s.", resp.Msg.Job.Id, jobStateLabel(resp.Msg.Job.State))},
		ResultsHeading: "Job",
		Results:        []string{formatJob(resp.Msg.Job)},
	})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	limit, err := parseLimit(ctx.Flag("limit"))
	if err != nil {
		return err
	}
	resp, err := h.client.ListJobs(context.Background(), connect.NewRequest(&jobsv1.ListJobsRequest{Limit: limit}))
	if err != nil {
		return cliapp.WrapAPIError("list jobs", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no jobs response")
	}
	results := make([]string, 0, len(resp.Msg.Jobs))
	for _, j := range resp.Msg.Jobs {
		results = append(results, formatJob(j))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d job(s).", len(resp.Msg.Jobs))},
		ResultsHeading: "Jobs",
		Results:        results,
		RetrievalHints: []string{
			"`jobs get <id>` — show a single job",
			"`jobs wait <id>` — block until a job is terminal (no polling)",
			"`jobs watch <id>` — stream progress live",
		},
	})
}

func (h *handlers) cancel(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.CancelJob(context.Background(), connect.NewRequest(&jobsv1.CancelJobRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("cancel job %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Job == nil {
		return fmt.Errorf("server returned no job")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Cancel requested for job %s (now %s).", resp.Msg.Job.Id, jobStateLabel(resp.Msg.Job.State))},
		Changes: []string{formatJob(resp.Msg.Job)},
	})
}

// watch streams progress events until the job reaches a terminal state. Each
// event is printed as it arrives; --json emits one ProgressEvent JSON object
// per line.
func (h *handlers) watch(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	stream, err := h.client.WatchJob(context.Background(), connect.NewRequest(&jobsv1.WatchJobRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("watch job %q", id), err, nil)
	}
	for stream.Receive() {
		ev := stream.Msg()
		if err := cliapp.RenderProtoList(ctx, ev, cliapp.ListReport{
			ResultsHeading: "Progress",
			Results:        []string{formatProgress(ev)},
		}); err != nil {
			return err
		}
	}
	if err := stream.Err(); err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("watch job %q", id), err, nil)
	}
	return nil
}

func parseLimit(raw string) (int32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid --limit %q (want a non-negative integer)", raw)
	}
	return int32(n), nil
}

func formatJob(j *jobsv1.Job) string {
	if j == nil {
		return "(nil)"
	}
	created := ""
	if j.CreatedAt != nil {
		created = j.CreatedAt.AsTime().Format(time.RFC3339)
	}
	detail := fmt.Sprintf("%s — op=%s lane=%s state=%s progress=%d%%", j.Id, j.Operation, jobLaneLabel(j.Lane), jobStateLabel(j.State), j.Progress)
	if j.ResultRef != "" {
		detail += " result=" + j.ResultRef
	}
	if j.Error != "" {
		detail += " error=" + j.Error
	}
	if created != "" {
		detail += " [created=" + created + "]"
	}
	return detail
}

func formatProgress(e *jobsv1.ProgressEvent) string {
	if e == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s — %s %d%% %s", e.JobId, jobStateLabel(e.State), e.Progress, e.Message)
}

func jobStateLabel(s jobsv1.JobState) string {
	switch s {
	case jobsv1.JobState_JOB_STATE_QUEUED:
		return "queued"
	case jobsv1.JobState_JOB_STATE_RUNNING:
		return "running"
	case jobsv1.JobState_JOB_STATE_SUCCEEDED:
		return "succeeded"
	case jobsv1.JobState_JOB_STATE_FAILED:
		return "failed"
	case jobsv1.JobState_JOB_STATE_CANCELED:
		return "canceled"
	default:
		return "unspecified"
	}
}

func jobLaneLabel(l jobsv1.JobLane) string {
	switch l {
	case jobsv1.JobLane_JOB_LANE_GPU:
		return "gpu"
	case jobsv1.JobLane_JOB_LANE_CPU:
		return "cpu"
	default:
		return "unspecified"
	}
}
