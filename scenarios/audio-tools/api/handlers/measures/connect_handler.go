package measures

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/measures"
)

type connectHandler struct{ deps Deps }

func NewConnectHandler(d Deps) *connectHandler { return &connectHandler{deps: d} }

func (h *connectHandler) CountBYOKCredentials(ctx context.Context, req *connect.Request[measuresv1.CountBYOKCredentialsRequest]) (*connect.Response[measuresv1.CountBYOKCredentialsResponse], error) {
	if h.deps.BYOK == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("BYOK counter not configured"))
	}
	n, err := countInWindow(ctx, req.Msg.GetWindow(), h.deps.Clock.Now(), h.deps.BYOK.CountCreatedBetween)
	if err != nil {
		return nil, countError(err)
	}
	return connect.NewResponse(&measuresv1.CountBYOKCredentialsResponse{Count: n}), nil
}

func (h *connectHandler) CountCorpusClips(ctx context.Context, req *connect.Request[measuresv1.CountCorpusClipsRequest]) (*connect.Response[measuresv1.CountCorpusClipsResponse], error) {
	if h.deps.Corpus == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("corpus counter not configured"))
	}
	n, err := countInWindow(ctx, req.Msg.GetWindow(), h.deps.Clock.Now(), h.deps.Corpus.CountClipsCreatedInWindow)
	if err != nil {
		return nil, countError(err)
	}
	return connect.NewResponse(&measuresv1.CountCorpusClipsResponse{Count: n}), nil
}

func (h *connectHandler) CountExperimentRuns(ctx context.Context, req *connect.Request[measuresv1.CountExperimentRunsRequest]) (*connect.Response[measuresv1.CountExperimentRunsResponse], error) {
	if h.deps.Experiments == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("experiment counter not configured"))
	}
	n, err := countInWindow(ctx, req.Msg.GetWindow(), h.deps.Clock.Now(), func(ctx context.Context, from, to time.Time) (int64, error) {
		counts, err := h.deps.Experiments.CountCreatedInRange(ctx, from, to)
		return counts.Runs, err
	})
	if err != nil {
		return nil, countError(err)
	}
	return connect.NewResponse(&measuresv1.CountExperimentRunsResponse{Count: n}), nil
}

func (h *connectHandler) CountExperiments(ctx context.Context, req *connect.Request[measuresv1.CountExperimentsRequest]) (*connect.Response[measuresv1.CountExperimentsResponse], error) {
	if h.deps.Experiments == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("experiment counter not configured"))
	}
	n, err := countInWindow(ctx, req.Msg.GetWindow(), h.deps.Clock.Now(), func(ctx context.Context, from, to time.Time) (int64, error) {
		counts, err := h.deps.Experiments.CountCreatedInRange(ctx, from, to)
		return counts.Experiments, err
	})
	if err != nil {
		return nil, countError(err)
	}
	return connect.NewResponse(&measuresv1.CountExperimentsResponse{Count: n}), nil
}

func (h *connectHandler) CountSpeakerProfiles(ctx context.Context, req *connect.Request[measuresv1.CountSpeakerProfilesRequest]) (*connect.Response[measuresv1.CountSpeakerProfilesResponse], error) {
	if h.deps.Speakers == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("speaker counter not configured"))
	}
	n, err := countInWindow(ctx, req.Msg.GetWindow(), h.deps.Clock.Now(), h.deps.Speakers.CountCreatedBetween)
	if err != nil {
		return nil, countError(err)
	}
	return connect.NewResponse(&measuresv1.CountSpeakerProfilesResponse{Count: n}), nil
}

func (h *connectHandler) CountWakewordTemplates(ctx context.Context, req *connect.Request[measuresv1.CountWakewordTemplatesRequest]) (*connect.Response[measuresv1.CountWakewordTemplatesResponse], error) {
	if h.deps.Wakewords == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("wakeword counter not configured"))
	}
	n, err := countInWindow(ctx, req.Msg.GetWindow(), h.deps.Clock.Now(), h.deps.Wakewords.CountCreatedBetween)
	if err != nil {
		return nil, countError(err)
	}
	return connect.NewResponse(&measuresv1.CountWakewordTemplatesResponse{Count: n}), nil
}

func (h *connectHandler) CountDuplexAccountingOperations(ctx context.Context, req *connect.Request[measuresv1.CountDuplexAccountingOperationsRequest]) (*connect.Response[measuresv1.CountDuplexAccountingOperationsResponse], error) {
	if h.deps.DuplexAccounting == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("duplex accounting counter not configured"))
	}
	n, err := countInWindow(ctx, req.Msg.GetWindow(), h.deps.Clock.Now(), h.deps.DuplexAccounting.CountCreatedBetween)
	if err != nil {
		return nil, countError(err)
	}
	return connect.NewResponse(&measuresv1.CountDuplexAccountingOperationsResponse{Count: n}), nil
}

func countError(err error) error {
	if connect.CodeOf(err) != connect.CodeUnknown {
		return err
	}
	return connect.NewError(connect.CodeInternal, err)
}
