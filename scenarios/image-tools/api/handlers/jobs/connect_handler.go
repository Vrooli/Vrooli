package jobs

import (
	"context"
	"errors"
	"log"

	internaljobs "image-tools/internal/jobs"

	"connectrpc.com/connect"

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
	return connect.NewResponse(&jobsv1.GetJobResponse{Job: domainToProto(j)}), nil
}

func (h *connectHandler) WaitJob(ctx context.Context, req *connect.Request[jobsv1.WaitJobRequest]) (*connect.Response[jobsv1.WaitJobResponse], error) {
	j, err := h.deps.Manager.Wait(ctx, req.Msg.GetId())
	if err != nil {
		// A client-canceled wait is not a server fault — the job continues
		// server-side. Surface it as Canceled without logging as an error.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, connect.NewError(connect.CodeCanceled, err)
		}
		return nil, jobError(err, h.deps.Logger, "WaitJob")
	}
	return connect.NewResponse(&jobsv1.WaitJobResponse{Job: domainToProto(j)}), nil
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
	return connect.NewResponse(&jobsv1.CancelJobResponse{Job: domainToProto(j)}), nil
}

func (h *connectHandler) WatchJob(ctx context.Context, req *connect.Request[jobsv1.WatchJobRequest], stream *connect.ServerStream[jobsv1.ProgressEvent]) error {
	ch, unsub, err := h.deps.Manager.Subscribe(req.Msg.GetId())
	if err != nil {
		return jobError(err, h.deps.Logger, "WatchJob")
	}
	defer unsub()

	for {
		select {
		case <-ctx.Done():
			// Client disconnected; the job continues server-side.
			return nil
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

func jobError(err error, logger *log.Logger, op string) error {
	if errors.Is(err, internaljobs.ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	logger.Printf("jobs.%s: %v", op, err)
	return connect.NewError(connect.CodeInternal, err)
}
