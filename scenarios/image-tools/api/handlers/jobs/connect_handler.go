package jobs

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/verdicts"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/lifecycle"
	"google.golang.org/protobuf/types/known/timestamppb"

	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
)

// JobManager is the narrow surface the handler depends on — the methods of
// internal/jobs.Manager it actually calls. Declared at the consumer so tests
// inject a fake without standing up the full Manager.
type JobManager interface {
	Get(ctx context.Context, id string) (internaljobs.Job, error)
	Wait(ctx context.Context, id string) (internaljobs.Job, error)
	List(ctx context.Context, limit int) ([]internaljobs.Job, error)
	Cancel(id string) error
	Subscribe(id string) (<-chan internaljobs.ProgressEvent, func(), error)
}

// Deps wires the seams the Connect jobs handler needs.
type Deps struct {
	Manager JobManager
	// Ratings persists the operator's output ratings. Nil leaves jobs unrated
	// and the rating verbs unimplemented.
	Ratings *verdicts.Ratings
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the JobsService handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetJob(ctx context.Context, req *connect.Request[jobsv1.GetJobRequest]) (*connect.Response[jobsv1.GetJobResponse], error) {
	j, err := h.deps.Manager.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, jobError(err, h.deps.Logger, "GetJob")
	}
	return connect.NewResponse(&jobsv1.GetJobResponse{Job: h.withRating(ctx, domainToProto(j))}), nil
}

func (h *connectHandler) WaitJob(ctx context.Context, req *connect.Request[jobsv1.WaitJobRequest]) (*connect.Response[jobsv1.WaitJobResponse], error) {
	waitCtx, endWait := endWhenClosing(ctx)
	defer endWait()
	j, err := h.deps.Manager.Wait(waitCtx, req.Msg.GetId())
	if err != nil {
		if ctx.Err() == nil && waitCtx.Err() != nil {
			return nil, restartingError(req.Msg.GetId())
		}
		// A client-canceled wait is not a server fault — the job continues
		// server-side. Surface it as Canceled without logging as an error.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, connect.NewError(connect.CodeCanceled, err)
		}
		return nil, jobError(err, h.deps.Logger, "WaitJob")
	}
	return connect.NewResponse(&jobsv1.WaitJobResponse{Job: h.withRating(ctx, domainToProto(j))}), nil
}

func (h *connectHandler) ListJobs(ctx context.Context, req *connect.Request[jobsv1.ListJobsRequest]) (*connect.Response[jobsv1.ListJobsResponse], error) {
	list, err := h.deps.Manager.List(ctx, int(req.Msg.GetLimit()))
	if err != nil {
		return nil, jobError(err, h.deps.Logger, "ListJobs")
	}
	resp := &jobsv1.ListJobsResponse{Jobs: make([]*jobsv1.Job, 0, len(list))}
	for _, j := range list {
		resp.Jobs = append(resp.Jobs, domainToProto(j))
	}
	h.withRatings(ctx, resp.Jobs)
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) CancelJob(_ context.Context, req *connect.Request[jobsv1.CancelJobRequest]) (*connect.Response[jobsv1.CancelJobResponse], error) {
	id := req.Msg.GetId()
	if err := h.deps.Manager.Cancel(id); err != nil {
		return nil, jobError(err, h.deps.Logger, "CancelJob")
	}
	// Cancel is best-effort + asynchronous for a running job; return the current
	// record so the caller sees the post-cancel state (or canceling-in-flight).
	j, err := h.deps.Manager.Get(context.Background(), id)
	if err != nil {
		return nil, jobError(err, h.deps.Logger, "CancelJob")
	}
	return connect.NewResponse(&jobsv1.CancelJobResponse{Job: h.withRating(context.Background(), domainToProto(j))}), nil
}

func (h *connectHandler) WatchJob(ctx context.Context, req *connect.Request[jobsv1.WatchJobRequest], stream *connect.ServerStream[jobsv1.ProgressEvent]) error {
	ch, unsub, err := h.deps.Manager.Subscribe(req.Msg.GetId())
	if err != nil {
		return jobError(err, h.deps.Logger, "WatchJob")
	}
	defer unsub()
	closing := lifecycle.Closing(ctx)

	for {
		select {
		case <-ctx.Done():
			// Client disconnected; the job continues server-side.
			return nil
		case <-closing:
			return restartingError(req.Msg.GetId())
		case ev, ok := <-ch:
			if !ok {
				// Channel closed: job reached a terminal state.
				return nil
			}
			if err := stream.Send(progressToProto(ev)); err != nil {
				return err
			}
		}
	}
}

// endWhenClosing returns a context that ends when the client leaves or when the
// process starts draining for a restart. A wait is an admitted request, and
// admitted requests hold a drain: without this a client waiting on a job holds
// the replacement until the job ends.
func endWhenClosing(ctx context.Context) (context.Context, context.CancelFunc) {
	waitCtx, cancel := context.WithCancel(ctx)
	if closing := lifecycle.Closing(ctx); closing != nil {
		go func() {
			select {
			case <-closing:
				cancel()
			case <-waitCtx.Done():
			}
		}()
	}
	return waitCtx, cancel
}

// restartingError tells a waiting client why its wait ended. The job itself is
// untouched: it keeps computing (and is given drain time) or, if it was only
// waiting, is back on the queue and resumes after the restart.
func restartingError(id string) error {
	return connect.NewError(connect.CodeUnavailable, fmt.Errorf("image-tools is restarting: the wait on job %s ended, not the job; check it with `image-tools jobs get %s` (a job that had not started resumes by itself after the restart)", id, id))
}

func jobError(err error, logger *log.Logger, op string) error {
	if errors.Is(err, internaljobs.ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	logger.Printf("jobs.%s: %v", op, err)
	return connect.NewError(connect.CodeInternal, err)
}

// withRating attaches the job's rating. A rating read failure is logged and the
// job returned unrated: a missing rating must never hide a job.
func (h *connectHandler) withRating(ctx context.Context, job *jobsv1.Job) *jobsv1.Job {
	h.withRatings(ctx, []*jobsv1.Job{job})
	return job
}

func (h *connectHandler) withRatings(ctx context.Context, jobs []*jobsv1.Job) {
	if h.deps.Ratings == nil || len(jobs) == 0 {
		return
	}
	ids := make([]string, 0, len(jobs))
	for _, j := range jobs {
		ids = append(ids, j.GetId())
	}
	ratings, err := h.deps.Ratings.ForJobs(ctx, ids)
	if err != nil {
		h.deps.Logger.Printf("jobs: read ratings: %v", err)
		return
	}
	for _, j := range jobs {
		if r, ok := ratings[j.GetId()]; ok {
			j.Rating = ratingToProto(r)
		}
	}
}

func ratingToProto(r verdicts.Rating) *jobsv1.JobRating {
	return &jobsv1.JobRating{
		JobId: r.JobID, Score: int32(r.Score), Note: r.Note, RatedBy: r.RatedBy,
		RatedAt: timestamppb.New(r.RatedAt), ModelId: r.ModelID, Operation: r.Operation,
	}
}

func (h *connectHandler) RateJob(ctx context.Context, req *connect.Request[jobsv1.RateJobRequest]) (*connect.Response[jobsv1.RateJobResponse], error) {
	if h.deps.Ratings == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("job ratings are unavailable"))
	}
	actor := strings.TrimSpace(req.Msg.GetActor())
	if actor == "" {
		actor = "api"
	}
	rating, err := h.deps.Ratings.Rate(ctx, req.Msg.GetJobId(), int(req.Msg.GetScore()), req.Msg.GetNote(), actor)
	if err != nil {
		return nil, ratingError(err, h.deps.Logger, "RateJob")
	}
	return connect.NewResponse(&jobsv1.RateJobResponse{Rating: ratingToProto(rating)}), nil
}

func (h *connectHandler) ClearJobRating(ctx context.Context, req *connect.Request[jobsv1.ClearJobRatingRequest]) (*connect.Response[jobsv1.ClearJobRatingResponse], error) {
	if h.deps.Ratings == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("job ratings are unavailable"))
	}
	if err := h.deps.Ratings.Clear(ctx, req.Msg.GetJobId()); err != nil {
		return nil, ratingError(err, h.deps.Logger, "ClearJobRating")
	}
	return connect.NewResponse(&jobsv1.ClearJobRatingResponse{}), nil
}

func (h *connectHandler) ListJobRatings(ctx context.Context, req *connect.Request[jobsv1.ListJobRatingsRequest]) (*connect.Response[jobsv1.ListJobRatingsResponse], error) {
	if h.deps.Ratings == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("job ratings are unavailable"))
	}
	list, err := h.deps.Ratings.List(ctx, strings.TrimSpace(req.Msg.GetModelId()), strings.TrimSpace(req.Msg.GetOperation()), int(req.Msg.GetLimit()))
	if err != nil {
		return nil, ratingError(err, h.deps.Logger, "ListJobRatings")
	}
	resp := &jobsv1.ListJobRatingsResponse{Ratings: make([]*jobsv1.JobRating, 0, len(list))}
	for _, r := range list {
		resp.Ratings = append(resp.Ratings, ratingToProto(r))
	}
	return connect.NewResponse(resp), nil
}

// ratingError maps rating-store errors to the status a caller can act on.
func ratingError(err error, logger *log.Logger, op string) error {
	switch {
	case errors.Is(err, verdicts.ErrJobUnknown), errors.Is(err, verdicts.ErrNotRated):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, verdicts.ErrBadScore), errors.Is(err, verdicts.ErrNoteTooLong):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, verdicts.ErrJobNotRateable):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	logger.Printf("jobs.%s: %v", op, err)
	return connect.NewError(connect.CodeInternal, errors.New("rating store failure"))
}
