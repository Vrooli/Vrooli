package experiment

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	intexp "audio-tools/internal/experiment"
	"audio-tools/internal/logx"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	errServiceNotConfigured   = errors.New("experiment service not configured")
	errSpeakerProfileRequired = errors.New("speaker experiments require a target_profile_id; enroll a voice profile with `audio-tools stt speaker-enroll --file <clip> --activate true` and list profile ids with `audio-tools stt speaker-status`")
)

// ExperimentManager is the handler-owned seam over internal/experiment.Manager.
type ExperimentManager interface {
	Submit(ctx context.Context, spec intexp.SubmitSpec) (intexp.Experiment, error)
	Get(ctx context.Context, id string) (intexp.Experiment, error)
	Wait(ctx context.Context, id string) (intexp.Experiment, error)
	List(ctx context.Context, filter intexp.ListFilter) ([]intexp.Experiment, error)
	Cancel(id string) error
	Subscribe(id string) (<-chan intexp.ProgressEvent, func(), error)
}

type Deps struct {
	Logger  logx.Logger
	Manager ExperimentManager
	Service *intexp.Service
}

type connectHandler struct{ deps Deps }

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		panic("experiment.NewConnectHandler requires Deps.Logger")
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) StartExperiment(ctx context.Context, req *connect.Request[experimentv1.StartExperimentRequest]) (*connect.Response[experimentv1.StartExperimentResponse], error) {
	if h.deps.Manager == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errServiceNotConfigured)
	}
	if h.deps.Service == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errServiceNotConfigured)
	}
	recipe := req.Msg.GetRecipe()
	if recipe == nil {
		recipe = &experimentv1.ExperimentRecipe{}
	}
	if err := validateRecipe(recipe); err != nil {
		if errors.Is(err, errSpeakerProfileRequired) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	recipeJSON, err := protojson.Marshal(recipe)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	exp, err := h.deps.Manager.Submit(ctx, intexp.SubmitSpec{
		Name:             req.Msg.GetName(),
		RecipeJSON:       recipeJSON,
		EstimatedSeconds: int(req.Msg.GetEstimatedSeconds()),
	})
	if err != nil {
		return nil, experimentError(err, h.deps.Logger, "StartExperiment")
	}
	return connect.NewResponse(&experimentv1.StartExperimentResponse{
		Experiment:       domainToProto(exp),
		EstimatedSeconds: req.Msg.GetEstimatedSeconds(),
	}), nil
}

func (h *connectHandler) GetExperiment(ctx context.Context, req *connect.Request[experimentv1.GetExperimentRequest]) (*connect.Response[experimentv1.GetExperimentResponse], error) {
	exp, runs, err := h.getWithRuns(ctx, req.Msg.GetId())
	if err != nil {
		return nil, experimentError(err, h.deps.Logger, "GetExperiment")
	}
	return connect.NewResponse(&experimentv1.GetExperimentResponse{Experiment: domainToProto(exp), Runs: runsToProto(runs)}), nil
}

func (h *connectHandler) WaitExperiment(ctx context.Context, req *connect.Request[experimentv1.WaitExperimentRequest]) (*connect.Response[experimentv1.WaitExperimentResponse], error) {
	if h.deps.Manager == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errServiceNotConfigured)
	}
	exp, err := h.deps.Manager.Wait(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, connect.NewError(connect.CodeCanceled, err)
		}
		return nil, experimentError(err, h.deps.Logger, "WaitExperiment")
	}
	runs, err := h.deps.Service.ListRuns(ctx, exp.ID)
	if err != nil {
		return nil, experimentError(err, h.deps.Logger, "WaitExperiment")
	}
	return connect.NewResponse(&experimentv1.WaitExperimentResponse{Experiment: domainToProto(exp), Runs: runsToProto(runs)}), nil
}

func (h *connectHandler) ListExperiments(ctx context.Context, req *connect.Request[experimentv1.ListExperimentsRequest]) (*connect.Response[experimentv1.ListExperimentsResponse], error) {
	if h.deps.Manager == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errServiceNotConfigured)
	}
	if req.Msg.GetLimit() < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("limit must be non-negative"))
	}
	if req.Msg.GetOffset() < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("offset must be non-negative"))
	}
	list, err := h.deps.Manager.List(ctx, intexp.ListFilter{
		Status: statusFromProto(req.Msg.GetStatus()),
		Limit:  int(req.Msg.GetLimit()),
		Offset: int(req.Msg.GetOffset()),
	})
	if err != nil {
		return nil, experimentError(err, h.deps.Logger, "ListExperiments")
	}
	resp := &experimentv1.ListExperimentsResponse{Experiments: make([]*experimentv1.Experiment, 0, len(list))}
	for _, exp := range list {
		resp.Experiments = append(resp.Experiments, domainToProto(exp))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) CancelExperiment(ctx context.Context, req *connect.Request[experimentv1.CancelExperimentRequest]) (*connect.Response[experimentv1.CancelExperimentResponse], error) {
	if h.deps.Manager == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errServiceNotConfigured)
	}
	id := req.Msg.GetId()
	if err := h.deps.Manager.Cancel(id); err != nil {
		return nil, experimentError(err, h.deps.Logger, "CancelExperiment")
	}
	exp, err := h.deps.Manager.Get(ctx, id)
	if err != nil {
		return nil, experimentError(err, h.deps.Logger, "CancelExperiment")
	}
	return connect.NewResponse(&experimentv1.CancelExperimentResponse{Experiment: domainToProto(exp)}), nil
}

func (h *connectHandler) StreamExperimentEvents(ctx context.Context, req *connect.Request[experimentv1.StreamExperimentEventsRequest], stream *connect.ServerStream[experimentv1.ExperimentEvent]) error {
	if h.deps.Manager == nil {
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("experiment service not configured"))
	}
	ch, unsub, err := h.deps.Manager.Subscribe(req.Msg.GetId())
	if err != nil {
		return experimentError(err, h.deps.Logger, "StreamExperimentEvents")
	}
	defer unsub()
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-ch:
			if !ok {
				return nil
			}
			if err := stream.Send(eventToProto(ev)); err != nil {
				return err
			}
		}
	}
}

func (h *connectHandler) GetExperimentReport(ctx context.Context, req *connect.Request[experimentv1.GetExperimentReportRequest]) (*connect.Response[experimentv1.GetExperimentReportResponse], error) {
	exp, report, runs, err := h.report(ctx, req.Msg.GetId())
	if err != nil {
		return nil, experimentError(err, h.deps.Logger, "GetExperimentReport")
	}
	return connect.NewResponse(&experimentv1.GetExperimentReportResponse{
		Experiment: domainToProto(exp),
		Report:     report,
		Runs:       runsToProto(runs),
	}), nil
}

func (h *connectHandler) CompareExperiments(ctx context.Context, req *connect.Request[experimentv1.CompareExperimentsRequest]) (*connect.Response[experimentv1.CompareExperimentsResponse], error) {
	if h.deps.Manager == nil || h.deps.Service == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errServiceNotConfigured)
	}
	resp := &experimentv1.CompareExperimentsResponse{Experiments: make([]*experimentv1.ComparedExperiment, 0, len(req.Msg.GetIds()))}
	for _, id := range req.Msg.GetIds() {
		exp, err := h.deps.Manager.Get(ctx, id)
		if err != nil {
			// A genuinely missing experiment id is still an error; but an
			// experiment that exists without a report (running/failed) must
			// appear in the comparison with a nil report rather than failing
			// the whole call and hiding the other experiments.
			return nil, experimentError(err, h.deps.Logger, "CompareExperiments")
		}
		runs, err := h.deps.Service.ListRuns(ctx, exp.ID)
		if err != nil {
			return nil, experimentError(err, h.deps.Logger, "CompareExperiments")
		}
		resp.Experiments = append(resp.Experiments, &experimentv1.ComparedExperiment{
			Experiment: domainToProto(exp),
			Report:     h.reportOrNil(ctx, exp),
			Runs:       runsToProto(runs),
		})
	}
	return connect.NewResponse(resp), nil
}

// reportOrNil returns the experiment's parsed report, or nil when none has been
// persisted yet (or it cannot be parsed). It never errors: callers that compare
// across a mix of finished and unfinished experiments rely on the nil signal.
func (h *connectHandler) reportOrNil(ctx context.Context, exp intexp.Experiment) *evalv1.EvalReport {
	if h.deps.Service == nil {
		return nil
	}
	data, err := h.deps.Service.GetReport(ctx, exp)
	if err != nil {
		return nil
	}
	report := &evalv1.EvalReport{}
	if err := protojson.Unmarshal(data, report); err != nil {
		h.deps.Logger.Printf("experiment.CompareExperiments: unparseable report for %s: %v", exp.ID, err)
		return nil
	}
	return report
}

// validateRecipe rejects recipes that would queue only to fail or skip every
// condition, returning an actionable message instead of a late opaque failure.
func validateRecipe(recipe *experimentv1.ExperimentRecipe) error {
	if recipe.GetRealtimeRepeats() < 0 {
		return errors.New("realtime_repeats must be non-negative")
	}
	if recipe.GetChunkMs() < 0 {
		return errors.New("chunk_ms must be non-negative")
	}
	if recipe.GetSeed() < 0 {
		return errors.New("seed must be non-negative")
	}
	if recipe.GetDroppedSpanThresholdWords() < 0 {
		return errors.New("dropped_span_threshold_words must be non-negative")
	}
	if recipe.GetLatencyTailSeconds() < 0 {
		return errors.New("latency_tail_seconds must be non-negative")
	}
	for i, strategy := range recipe.GetStrategies() {
		switch strategy.GetKind() {
		case "", "batch", "vad_segment", "overlap_agree":
		default:
			return fmt.Errorf("strategies[%d].kind %q is not supported", i, strategy.GetKind())
		}
		if strategy.GetOverlapWindowMs() < 0 {
			return fmt.Errorf("strategies[%d].overlap_window_ms must be non-negative", i)
		}
		if strategy.GetOverlapCommitRuns() < 0 {
			return fmt.Errorf("strategies[%d].overlap_commit_runs must be non-negative", i)
		}
		if strategy.GetVadSilenceMs() < 0 {
			return fmt.Errorf("strategies[%d].vad_silence_ms must be non-negative", i)
		}
		if strategy.GetOverlapMaxWindowMs() < 0 {
			return fmt.Errorf("strategies[%d].overlap_max_window_ms must be non-negative", i)
		}
	}
	if longForm := recipe.GetLongForm(); longForm != nil {
		if longForm.GetTargetDurationSeconds() < 0 {
			return errors.New("long_form.target_duration_seconds must be non-negative")
		}
		if longForm.GetGapMs() < 0 {
			return errors.New("long_form.gap_ms must be non-negative")
		}
		for i, duration := range longForm.GetSweepDurationsSeconds() {
			if duration < 0 {
				return fmt.Errorf("long_form.sweep_durations_seconds[%d] must be non-negative", i)
			}
		}
	}
	if aug := recipe.GetAugmentation(); aug != nil {
		for i, snr := range aug.GetSnrDb() {
			if snr < 0 {
				return fmt.Errorf("augmentation.snr_db[%d] must be non-negative", i)
			}
		}
	}
	s := recipe.GetSpeaker()
	if s == nil {
		return nil
	}
	wantsSpeaker := s.GetExtractionEnabled() || s.GetVerificationEnabled() || s.GetAblationEnabled()
	if wantsSpeaker && s.GetTargetProfileId() == "" {
		return errSpeakerProfileRequired
	}
	return nil
}

func (h *connectHandler) getWithRuns(ctx context.Context, id string) (intexp.Experiment, []intexp.Run, error) {
	if h.deps.Manager == nil || h.deps.Service == nil {
		return intexp.Experiment{}, nil, errServiceNotConfigured
	}
	exp, err := h.deps.Manager.Get(ctx, id)
	if err != nil {
		return intexp.Experiment{}, nil, err
	}
	runs, err := h.deps.Service.ListRuns(ctx, id)
	if err != nil {
		return intexp.Experiment{}, nil, err
	}
	return exp, runs, nil
}

func (h *connectHandler) report(ctx context.Context, id string) (intexp.Experiment, *evalv1.EvalReport, []intexp.Run, error) {
	exp, runs, err := h.getWithRuns(ctx, id)
	if err != nil {
		return intexp.Experiment{}, nil, nil, err
	}
	data, err := h.deps.Service.GetReport(ctx, exp)
	if err != nil {
		return intexp.Experiment{}, nil, nil, err
	}
	report := &evalv1.EvalReport{}
	if err := protojson.Unmarshal(data, report); err != nil {
		return intexp.Experiment{}, nil, nil, err
	}
	return exp, report, runs, nil
}

func experimentError(err error, logger logx.Logger, op string) error {
	var nf intexp.ErrNotFound
	if errors.As(err, &nf) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	if errors.Is(err, intexp.ErrNotStarted) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if errors.Is(err, errServiceNotConfigured) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	logger.Printf("experiment.%s: %v", op, err)
	return connect.NewError(connect.CodeInternal, err)
}
