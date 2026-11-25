package jobs

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
	jobsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs/jobs_v1connect"
	"image-tools/cli/internal/actor"

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

// rate mirrors JobsService.RateJob: the operator's 1-5 rating of an output.
func (h *handlers) rate(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	raw := strings.TrimSpace(ctx.Flag("score"))
	if raw == "" {
		return fmt.Errorf("--score is required: a whole number from 1 (poor) to 5 (excellent)")
	}
	score, err := strconv.Atoi(raw)
	if err != nil || score < 1 || score > 5 {
		return fmt.Errorf("invalid --score %q: expected a whole number from 1 to 5", raw)
	}
	resp, err := h.client.RateJob(context.Background(), connect.NewRequest(&jobsv1.RateJobRequest{
		JobId: id, Score: int32(score), Note: ctx.Flag("note"), Actor: actor.Name(),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("rate job %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Rating == nil {
		return fmt.Errorf("server returned no rating")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Rated job %s %d of 5.", id, score)},
		Changes: []string{formatRating(resp.Msg.Rating)},
		NextCommand: []string{
			"`jobs ratings --model <id>` — list the ratings behind a model's verdict",
			fmt.Sprintf("`jobs unrate %s` — remove this rating", id),
		},
	})
}

// unrate mirrors JobsService.ClearJobRating.
func (h *handlers) unrate(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.ClearJobRating(context.Background(), connect.NewRequest(&jobsv1.ClearJobRatingRequest{JobId: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("remove rating of job %q", id), err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Removed the rating of job %s.", id)},
	})
}

// ratings mirrors JobsService.ListJobRatings.
func (h *handlers) ratings(ctx cliapp.RunContext) error {
	limit, err := parseLimit(ctx.Flag("limit"))
	if err != nil {
		return err
	}
	resp, err := h.client.ListJobRatings(context.Background(), connect.NewRequest(&jobsv1.ListJobRatingsRequest{
		ModelId:   strings.TrimSpace(ctx.Flag("model")),
		Operation: strings.TrimSpace(ctx.Flag("operation")),
		Limit:     limit,
	}))
	if err != nil {
		return cliapp.WrapAPIError("list job ratings", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Ratings))
	for _, r := range resp.Msg.Ratings {
		results = append(results, formatRating(r))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d rating(s).", len(resp.Msg.Ratings))},
		ResultsHeading: "Ratings",
		Results:        results,
		RetrievalHints: []string{"`jobs rate <id> --score N --note ...` — rate a result", "`models get <id>` — see how ratings moved a verdict"},
	})
}

func formatRating(r *jobsv1.JobRating) string {
	line := fmt.Sprintf("%s — %d of 5 model=%s op=%s by %s", r.JobId, r.Score, orNone(r.ModelId), orNone(r.Operation), orNone(r.RatedBy))
	if r.RatedAt != nil {
		line += " at " + r.RatedAt.AsTime().Format(time.RFC3339)
	}
	if r.Note != "" {
		line += " — " + r.Note
	}
	return line
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
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
	// ParseInt with bitSize=32 bounds the result to int32 range, so the
	// conversion below cannot overflow (avoids gosec G109).
	n, err := strconv.ParseInt(raw, 10, 32)
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
	detail := fmt.Sprintf("%s — op=%s queue=%s state=%s progress=%d%%", j.Id, j.Operation, jobQueueLabel(j.SchedulingQueue, j.Lane), jobStateLabel(j.State), j.Progress)
	if j.ResultRef != "" {
		detail += " result=" + j.ResultRef
	}
	if j.Error != "" {
		detail += " error=" + j.Error
	}
	if created != "" {
		detail += " [created=" + created + "]"
	}
	if j.Rating != nil {
		detail += fmt.Sprintf(" rating=%d/5", j.Rating.Score)
	}
	for _, key := range []string{"failure_kind", "safety_review", "safety_action", "safety_reason", "safety_review_detail", "safety_refused_refs", "safety_shadow_refs"} {
		if value := strings.TrimSpace(j.ResultMeta[key]); value != "" {
			detail += " " + key + "=" + value
		}
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
	case jobsv1.JobState_JOB_STATE_ADMITTING:
		return "admitting"
	case jobsv1.JobState_JOB_STATE_CANCELING:
		return "canceling"
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

func jobQueueLabel(queue jobsv1.JobQueue, legacyLane jobsv1.JobLane) string {
	if queue == jobsv1.JobQueue_JOB_QUEUE_UNSPECIFIED {
		switch legacyLane {
		case jobsv1.JobLane_JOB_LANE_GPU:
			queue = jobsv1.JobQueue_JOB_QUEUE_LOCAL_SERIAL
		case jobsv1.JobLane_JOB_LANE_CPU:
			queue = jobsv1.JobQueue_JOB_QUEUE_CPU_POOL
		case jobsv1.JobLane_JOB_LANE_NETWORK:
			queue = jobsv1.JobQueue_JOB_QUEUE_NETWORK_POOL
		}
	}
	switch queue {
	case jobsv1.JobQueue_JOB_QUEUE_CAPACITY:
		return "capacity"
	case jobsv1.JobQueue_JOB_QUEUE_LOCAL_SERIAL:
		return "local-serial"
	case jobsv1.JobQueue_JOB_QUEUE_CPU_POOL:
		return "cpu-pool"
	case jobsv1.JobQueue_JOB_QUEUE_NETWORK_POOL:
		return "network-pool"
	default:
		return "unspecified"
	}
}

// download fetches one result of a job through the blob endpoint by job id.
func (h *handlers) download(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	idx := 0
	if raw := ctx.Flag("index"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 {
			return fmt.Errorf("--index must be a non-negative integer")
		}
		idx = v
	}
	out := ctx.Flag("out")
	if out == "" {
		return fmt.Errorf("--out is required")
	}
	resp, err := h.client.GetJob(context.Background(), connect.NewRequest(&jobsv1.GetJobRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get job %q", id), err, nil)
	}
	job := resp.Msg.GetJob()
	refs := job.GetResultRefs()
	if len(refs) == 0 && job.GetResultRef() != "" {
		refs = []string{job.GetResultRef()}
	}
	if idx >= len(refs) {
		return fmt.Errorf("job %s has %d result(s); --index %d is out of range", id, len(refs), idx)
	}
	if err := downloadResult(h.core, refs[idx], out); err != nil {
		return err
	}
	return ctx.RenderMutation(cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Downloaded result %d of job %s", idx, id)},
		Changes: []string{fmt.Sprintf("wrote %s", out)},
	})
}

// downloadResult GETs a managed blob and writes it to out.
func downloadResult(core *cliapp.ScenarioApp, ref, out string) error {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	url := strings.TrimRight(baseURL, "/") + "/api/v1/blobs/" + ref
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download result: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download result failed (%d)", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return fmt.Errorf("write output %q: %w", out, err)
	}
	return nil
}
