package experiment

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"connectrpc.com/connect"

	intexp "audio-tools/internal/experiment"
	"audio-tools/internal/logx"
	"audio-tools/internal/protoint"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	errServiceNotConfigured = errors.New("experiment service not configured")
)

const (
	// A known-duration experiment gets 25% headroom plus a small fixed setup
	// allowance. This leaves room for normal decode/finalization variation while
	// making an experiment that has stopped making useful progress fail visibly
	// instead of occupying the single async worker indefinitely.
	experimentRuntimeSlackFraction = 4
	experimentRuntimeMinSlack      = 2 * time.Minute
	experimentRuntimeUnknown       = 30 * time.Minute
	// A single provider-cell qualification may intentionally reach the trust
	// floor's 60-minute rung, but ad-hoc recipe multiplication must not turn
	// into an accidental multi-hour worker reservation. Split broader matrices
	// into explicit persisted cells so their progress and cancellation remain
	// observable.
	experimentMaxEstimatedSeconds int32 = 60 * 60
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
	Logger              logx.Logger
	Manager             ExperimentManager
	Service             *intexp.Service
	EstimateClipSeconds func(context.Context, []string) (int32, error)
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
	if err := intexp.ValidateRecipe(recipe); err != nil {
		if errors.Is(err, intexp.ErrSpeakerProfileRequired) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	recipeJSON, err := protojson.Marshal(recipe)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	estimatedSeconds, err := h.estimateExperimentSeconds(ctx, recipe, req.Msg.GetEstimatedSeconds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if req.Msg.GetDryRun() {
		// Resolve + validate only: return the recipe preview without enqueuing.
		// The preview mirrors what a real start would persist (pre-materialization),
		// with no id and an unspecified status so it can't be mistaken for a run.
		preview := &experimentv1.Experiment{
			Name:        req.Msg.GetName(),
			Status:      experimentv1.ExperimentStatus_EXPERIMENT_STATUS_UNSPECIFIED,
			Recipe:      recipe,
			MachineJson: string(experimentMachineJSON(recipeJSON)),
		}
		return connect.NewResponse(&experimentv1.StartExperimentResponse{
			Experiment:       preview,
			EstimatedSeconds: estimatedSeconds,
			DryRun:           true,
		}), nil
	}
	exp, err := h.deps.Manager.Submit(ctx, intexp.SubmitSpec{
		Name:             req.Msg.GetName(),
		RecipeJSON:       recipeJSON,
		MachineJSON:      experimentMachineJSON(recipeJSON),
		EstimatedSeconds: int(estimatedSeconds),
		MaxRuntime:       experimentRuntimeBudget(estimatedSeconds),
	})
	if err != nil {
		return nil, experimentError(err, h.deps.Logger, "StartExperiment")
	}
	return connect.NewResponse(&experimentv1.StartExperimentResponse{
		Experiment:       domainToProto(exp),
		EstimatedSeconds: estimatedSeconds,
	}), nil
}

func experimentRuntimeBudget(estimatedSeconds int32) time.Duration {
	if estimatedSeconds <= 0 {
		return experimentRuntimeUnknown
	}
	base := time.Duration(estimatedSeconds) * time.Second
	slack := base / experimentRuntimeSlackFraction
	if slack < experimentRuntimeMinSlack {
		slack = experimentRuntimeMinSlack
	}
	return base + slack
}

func (h *connectHandler) estimateExperimentSeconds(ctx context.Context, recipe *experimentv1.ExperimentRecipe, clientOverride int32) (int32, error) {
	if recipe == nil {
		if clientOverride > experimentMaxEstimatedSeconds {
			return 0, experimentDurationLimitError(int64(clientOverride))
		}
		return clientOverride, nil
	}
	durationSeconds := estimatedRecipeDurationSeconds(recipe)
	if durationSeconds <= 0 && h.deps.EstimateClipSeconds != nil {
		estimated, err := h.deps.EstimateClipSeconds(ctx, recipe.GetClipIds())
		if err != nil {
			return 0, err
		}
		durationSeconds = estimated
	}
	if durationSeconds <= 0 {
		if clientOverride > experimentMaxEstimatedSeconds {
			return 0, experimentDurationLimitError(int64(clientOverride))
		}
		return clientOverride, nil
	}
	// Cells are independent provider/strategy/lane work units. Their
	// repeat_count must contribute to admission cost; counting only the number
	// of cells would let a repeated real-time recipe understate its duration.
	cellRuns := int64(0)
	if cells := recipe.GetCells(); len(cells) > 0 {
		for _, cell := range cells {
			repeats := cell.GetRepeatCount()
			if repeats < 1 {
				repeats = 1
			}
			cellRuns += int64(repeats)
		}
	} else {
		strategies := len(recipe.GetStrategies())
		if strategies == 0 {
			strategies = 3
		}
		repeats := int(recipe.GetRealtimeRepeats()) + 1
		if repeats < 1 {
			repeats = 1
		}
		cellRuns = int64(strategies * repeats)
	}
	conditions := estimatedConditionCount(recipe)
	total := int64(durationSeconds) * cellRuns * int64(conditions)
	if total < 1 {
		total = 1
	}
	// The recipe-derived duration is authoritative. A caller may reserve more
	// time for a known external setup cost, but it may not shrink the derived
	// budget and turn a long run into a misleadingly short one.
	if clientOverride > 0 && int64(clientOverride) > total {
		total = int64(clientOverride)
	}
	if total > int64(experimentMaxEstimatedSeconds) {
		return 0, experimentDurationLimitError(total)
	}
	return protoint.FromInt64(total), nil
}

func experimentDurationLimitError(seconds int64) error {
	return fmt.Errorf("estimated experiment duration %ds exceeds the %ds qualification ceiling; split the matrix into explicit provider cells", seconds, experimentMaxEstimatedSeconds)
}

func estimatedRecipeDurationSeconds(recipe *experimentv1.ExperimentRecipe) int32 {
	if recipe.GetRealizedDurationMs() > 0 {
		return protoint.FromInt64((recipe.GetRealizedDurationMs() + 999) / 1000)
	}
	longForm := recipe.GetLongForm()
	if longForm == nil {
		return 0
	}
	if sweep := longForm.GetSweepDurationsSeconds(); len(sweep) > 0 {
		var total int32
		for _, seconds := range sweep {
			if seconds > 0 {
				total += seconds
			}
		}
		return total
	}
	if longForm.GetEnabled() && longForm.GetTargetDurationSeconds() > 0 {
		return longForm.GetTargetDurationSeconds()
	}
	return 0
}

func estimatedConditionCount(recipe *experimentv1.ExperimentRecipe) int {
	count := 1
	if realized := len(recipe.GetRealizedAugmentationConditions()); realized > 0 {
		count *= realized
	} else if aug := recipe.GetAugmentation(); aug != nil {
		noise := len(aug.GetNoiseTypes())
		snr := len(aug.GetSnrDb())
		voices := len(aug.GetCompetingVoiceIds())
		switch {
		case noise > 0 && snr > 0:
			count *= noise * snr
		case noise > 0:
			count *= noise
		case snr > 0:
			count *= snr
		}
		if voices > 0 {
			count *= voices
		}
	}
	if realized := len(recipe.GetRealizedSpeakerConditions()); realized > 0 {
		count *= realized
	} else if speaker := recipe.GetSpeaker(); speaker != nil && speaker.GetAblationEnabled() {
		count *= 4
	}
	if count < 1 {
		return 1
	}
	return count
}

func experimentMachineJSON(recipeJSON []byte) []byte {
	host, _ := os.Hostname()
	sum := sha256.Sum256(recipeJSON)
	data, err := json.Marshal(map[string]any{
		"host":          host,
		"goos":          runtime.GOOS,
		"goarch":        runtime.GOARCH,
		"recipe_sha256": fmt.Sprintf("%x", sum[:]),
	})
	if err != nil {
		return []byte("{}")
	}
	return data
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

func (h *connectHandler) DeleteExperiment(ctx context.Context, req *connect.Request[experimentv1.DeleteExperimentRequest]) (*connect.Response[experimentv1.DeleteExperimentResponse], error) {
	if h.deps.Manager == nil || h.deps.Service == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errServiceNotConfigured)
	}
	exp, err := h.deps.Manager.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, experimentError(err, h.deps.Logger, "DeleteExperiment")
	}
	if !exp.Status.Terminal() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("cannot delete experiment %s while status is %s", exp.ID, exp.Status))
	}
	deletedReport, err := h.deps.Service.DeleteExperiment(ctx, exp)
	if err != nil {
		return nil, experimentError(err, h.deps.Logger, "DeleteExperiment")
	}
	return connect.NewResponse(&experimentv1.DeleteExperimentResponse{Id: exp.ID, DeletedReport: deletedReport}), nil
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

func (h *connectHandler) RecordQualificationEvidence(ctx context.Context, req *connect.Request[experimentv1.RecordQualificationEvidenceRequest]) (*connect.Response[experimentv1.RecordQualificationEvidenceResponse], error) {
	if h.deps.Service == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errServiceNotConfigured)
	}
	evidence, err := qualificationEvidenceFromProto(req.Msg.GetEvidence())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if len(evidence.MachineJSON) == 0 {
		// A local qualification command normally has no reason to repeat host
		// identity. Stamp it server-side so it can only qualify reports from
		// this same host/OS/architecture tuple.
		evidence.MachineJSON = experimentMachineJSON(nil)
	}
	saved, err := h.deps.Service.RecordQualificationEvidence(ctx, evidence)
	if err != nil {
		return nil, experimentError(err, h.deps.Logger, "RecordQualificationEvidence")
	}
	return connect.NewResponse(&experimentv1.RecordQualificationEvidenceResponse{Evidence: qualificationEvidenceToProto(saved)}), nil
}

func (h *connectHandler) ListQualificationEvidence(ctx context.Context, req *connect.Request[experimentv1.ListQualificationEvidenceRequest]) (*connect.Response[experimentv1.ListQualificationEvidenceResponse], error) {
	if h.deps.Service == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errServiceNotConfigured)
	}
	items, err := h.deps.Service.ListQualificationEvidence(ctx, intexp.QualificationEvidenceFilter{
		EngineID: req.Msg.GetEngineId(), ModelID: req.Msg.GetModelId(), Strategy: req.Msg.GetStrategy(), PolicyProfile: req.Msg.GetPolicyProfile(),
	})
	if err != nil {
		return nil, experimentError(err, h.deps.Logger, "ListQualificationEvidence")
	}
	resp := &experimentv1.ListQualificationEvidenceResponse{Evidence: make([]*experimentv1.QualificationEvidence, 0, len(items))}
	for _, item := range items {
		resp.Evidence = append(resp.Evidence, qualificationEvidenceToProto(item))
	}
	return connect.NewResponse(resp), nil
}

// reportOrNil returns the experiment's parsed report, or nil when none has been
// persisted yet (or it cannot be parsed). It never errors: callers that compare
// across a mix of finished and unfinished experiments rely on the nil signal.
func (h *connectHandler) reportOrNil(ctx context.Context, exp intexp.Experiment) *evalv1.EvalReport {
	report, err := h.decodedReport(ctx, exp)
	if err != nil {
		h.deps.Logger.Printf("experiment.CompareExperiments: unparseable report for %s: %v", exp.ID, err)
		return nil
	}
	h.aggregatePromotionVerdicts(ctx, exp, report)
	return report
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
	report, err := h.decodedReport(ctx, exp)
	if err != nil {
		return intexp.Experiment{}, nil, nil, err
	}
	h.aggregatePromotionVerdicts(ctx, exp, report)
	return exp, report, runs, nil
}

func (h *connectHandler) decodedReport(ctx context.Context, exp intexp.Experiment) (*evalv1.EvalReport, error) {
	if h.deps.Service == nil {
		return nil, errServiceNotConfigured
	}
	data, err := h.deps.Service.GetReport(ctx, exp)
	if err != nil {
		return nil, err
	}
	report := &evalv1.EvalReport{}
	if err := protojson.Unmarshal(data, report); err != nil {
		return nil, err
	}
	return report, nil
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
