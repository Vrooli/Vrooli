package validation

import (
	"context"
	"errors"
	"log"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/validationrunner"

	"connectrpc.com/connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/validation"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Deps struct {
	Repository catalog.Repository
	Service    *validationrunner.Service
	Logger     *log.Logger
}

type connectHandler struct {
	deps Deps
	err  error
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.Service == nil {
		runner, err := validationrunner.NewEngineRunner("")
		if err != nil {
			return &connectHandler{deps: d, err: err}
		} else {
			d.Service = validationrunner.NewService(d.Repository, runner)
		}
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) RunTemplateValidation(ctx context.Context, req *connect.Request[validationv1.RunTemplateValidationRequest]) (*connect.Response[validationv1.RunTemplateValidationResponse], error) {
	if h.err != nil {
		h.deps.Logger.Printf("validation.RunTemplateValidation: engine unavailable: %v", h.err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("template engine unavailable"))
	}
	run, err := h.deps.Service.RunValidation(ctx, validationrunner.ValidateRequest{
		TemplateID: req.Msg.TemplateId,
		Mode:       modeFromProto(req.Msg.Mode),
	})
	if err != nil {
		h.deps.Logger.Printf("validation.RunTemplateValidation(%q): %v", req.Msg.TemplateId, err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("run template validation"))
	}
	return connect.NewResponse(&validationv1.RunTemplateValidationResponse{Run: runToProto(run)}), nil
}

func (h *connectHandler) ListValidationRuns(ctx context.Context, req *connect.Request[validationv1.ListValidationRunsRequest]) (*connect.Response[validationv1.ListValidationRunsResponse], error) {
	runs, err := h.deps.Repository.ListValidationRuns(ctx, req.Msg.TemplateId)
	if err != nil {
		h.deps.Logger.Printf("validation.ListValidationRuns: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("list validation runs"))
	}
	resp := &validationv1.ListValidationRunsResponse{Runs: make([]*validationv1.ValidationRun, 0, len(runs))}
	for _, run := range runs {
		resp.Runs = append(resp.Runs, runToProto(run))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetValidationRun(ctx context.Context, req *connect.Request[validationv1.GetValidationRunRequest]) (*connect.Response[validationv1.GetValidationRunResponse], error) {
	run, err := h.deps.Repository.GetValidationRun(ctx, req.Msg.Id)
	if err != nil {
		var notFound catalog.ErrNotFound
		if errors.As(err, &notFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		h.deps.Logger.Printf("validation.GetValidationRun(%q): %v", req.Msg.Id, err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("get validation run"))
	}
	return connect.NewResponse(&validationv1.GetValidationRunResponse{Run: runToProto(run)}), nil
}

func (h *connectHandler) RecordFleetDrift(ctx context.Context, _ *connect.Request[validationv1.RecordFleetDriftRequest]) (*connect.Response[validationv1.RecordFleetDriftResponse], error) {
	if h.err != nil {
		h.deps.Logger.Printf("validation.RecordFleetDrift: engine unavailable: %v", h.err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("template engine unavailable"))
	}
	snapshot, err := h.deps.Service.RecordFleetDrift(ctx)
	if err != nil {
		h.deps.Logger.Printf("validation.RecordFleetDrift: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("record fleet drift"))
	}
	return connect.NewResponse(&validationv1.RecordFleetDriftResponse{Snapshot: driftToProto(snapshot)}), nil
}

func (h *connectHandler) ListDriftSnapshots(ctx context.Context, req *connect.Request[validationv1.ListDriftSnapshotsRequest]) (*connect.Response[validationv1.ListDriftSnapshotsResponse], error) {
	snapshots, err := h.deps.Repository.ListDriftSnapshots(ctx, req.Msg.TemplateId)
	if err != nil {
		h.deps.Logger.Printf("validation.ListDriftSnapshots: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("list drift snapshots"))
	}
	resp := &validationv1.ListDriftSnapshotsResponse{Snapshots: make([]*validationv1.DriftSnapshot, 0, len(snapshots))}
	for _, snapshot := range snapshots {
		resp.Snapshots = append(resp.Snapshots, driftToProto(snapshot))
	}
	return connect.NewResponse(resp), nil
}

func modeFromProto(mode validationv1.ValidationMode) catalog.ValidationMode {
	switch mode {
	case validationv1.ValidationMode_VALIDATION_MODE_DEEP:
		return catalog.ModeDeep
	case validationv1.ValidationMode_VALIDATION_MODE_DRIFT:
		return catalog.ModeDrift
	case validationv1.ValidationMode_VALIDATION_MODE_SHALLOW:
		return catalog.ModeShallow
	default:
		return catalog.ModeShallow
	}
}

func modeToProto(mode catalog.ValidationMode) validationv1.ValidationMode {
	switch mode {
	case catalog.ModeShallow:
		return validationv1.ValidationMode_VALIDATION_MODE_SHALLOW
	case catalog.ModeDeep:
		return validationv1.ValidationMode_VALIDATION_MODE_DEEP
	case catalog.ModeDrift:
		return validationv1.ValidationMode_VALIDATION_MODE_DRIFT
	default:
		return validationv1.ValidationMode_VALIDATION_MODE_UNSPECIFIED
	}
}

func runToProto(run catalog.ValidationRun) *validationv1.ValidationRun {
	out := &validationv1.ValidationRun{
		Id:         run.ID,
		TemplateId: run.TemplateID,
		Mode:       modeToProto(run.Mode),
		Target:     run.Target,
		Status:     run.Status,
		StartedAt:  timestamppb.New(run.StartedAt),
		FinishedAt: timestamppb.New(run.FinishedAt),
		Trigger:    run.Trigger,
	}
	for _, phase := range run.PhaseResults {
		out.PhaseResults = append(out.PhaseResults, &validationv1.PhaseResult{
			Phase:        phase.Phase,
			Status:       phase.Status,
			FindingCount: phase.FindingCount,
		})
	}
	for _, finding := range run.Findings {
		out.Findings = append(out.Findings, &validationv1.ValidationFinding{
			Key:      finding.Key,
			Severity: finding.Severity,
			Summary:  finding.Summary,
			Source:   finding.Source,
		})
	}
	return out
}

func driftToProto(snapshot catalog.DriftSnapshot) *validationv1.DriftSnapshot {
	return &validationv1.DriftSnapshot{
		Id:         snapshot.ID,
		TemplateId: snapshot.TemplateID,
		Target:     snapshot.Target,
		Status:     snapshot.Status,
		DriftCount: snapshot.DriftCount,
		CapturedAt: timestamppb.New(snapshot.CapturedAt),
	}
}
