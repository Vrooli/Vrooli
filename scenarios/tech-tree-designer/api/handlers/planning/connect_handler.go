package planning

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	planningdomain "tech-tree-designer/internal/planning"

	planningv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/planning"
	planningconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/planning/planning_v1connect"
)

type Handler struct {
	planningconnect.UnimplementedPlanningServiceHandler
	service *planningdomain.Service
}

func NewHandler(service *planningdomain.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreatePlannedScenario(ctx context.Context, req *connect.Request[planningv1.CreatePlannedScenarioRequest]) (*connect.Response[planningv1.PlannedScenario], error) {
	scenario, err := h.service.Create(ctx, planningdomain.CreateInput{
		Slug:            req.Msg.GetSlug(),
		DisplayName:     req.Msg.GetDisplayName(),
		Sector:          req.Msg.GetSector(),
		Tier:            req.Msg.GetTier(),
		TargetStability: req.Msg.GetTargetStability(),
	})
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(scenarioToProto(scenario)), nil
}

func (h *Handler) ListPlannedScenarios(ctx context.Context, req *connect.Request[planningv1.ListPlannedScenariosRequest]) (*connect.Response[planningv1.ListPlannedScenariosResponse], error) {
	scenarios, err := h.service.List(ctx, planningdomain.ListFilter{Sector: req.Msg.GetSector(), Tier: req.Msg.GetTier()})
	if err != nil {
		return nil, toConnectError(err)
	}
	resp := &planningv1.ListPlannedScenariosResponse{Scenarios: make([]*planningv1.PlannedScenario, 0, len(scenarios))}
	for _, scenario := range scenarios {
		resp.Scenarios = append(resp.Scenarios, scenarioToProto(scenario))
	}
	return connect.NewResponse(resp), nil
}

func (h *Handler) GetPlannedScenario(ctx context.Context, req *connect.Request[planningv1.GetPlannedScenarioRequest]) (*connect.Response[planningv1.PlannedScenario], error) {
	scenario, err := h.service.Get(ctx, req.Msg.GetSlug())
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(scenarioToProto(scenario)), nil
}

func (h *Handler) PutPlannedProtoFile(ctx context.Context, req *connect.Request[planningv1.PutPlannedProtoFileRequest]) (*connect.Response[planningv1.PlannedProtoFile], error) {
	file, err := h.service.PutFile(ctx, planningdomain.PutFileInput{Slug: req.Msg.GetSlug(), Path: req.Msg.GetPath(), Text: req.Msg.GetText()})
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(fileToProto(file)), nil
}

func (h *Handler) DeletePlannedProtoFile(ctx context.Context, req *connect.Request[planningv1.DeletePlannedProtoFileRequest]) (*connect.Response[planningv1.DeletePlannedProtoFileResponse], error) {
	deleted, err := h.service.DeleteFile(ctx, req.Msg.GetSlug(), req.Msg.GetPath())
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&planningv1.DeletePlannedProtoFileResponse{Deleted: deleted}), nil
}

func (h *Handler) ValidatePlannedScenario(ctx context.Context, req *connect.Request[planningv1.ValidatePlannedScenarioRequest]) (*connect.Response[planningv1.ValidatePlannedScenarioResponse], error) {
	passed, findings, err := h.service.Validate(ctx, req.Msg.GetSlug())
	if err != nil {
		return nil, toConnectError(err)
	}
	resp := &planningv1.ValidatePlannedScenarioResponse{
		Slug:     req.Msg.GetSlug(),
		Passed:   passed,
		Findings: make([]*planningv1.PlanFinding, 0, len(findings)),
	}
	for _, finding := range findings {
		resp.Findings = append(resp.Findings, findingToProto(finding))
	}
	return connect.NewResponse(resp), nil
}

func (h *Handler) MaterializePlannedScenario(ctx context.Context, req *connect.Request[planningv1.MaterializePlannedScenarioRequest]) (*connect.Response[planningv1.MaterializePlannedScenarioResponse], error) {
	result, err := h.service.Materialize(ctx, req.Msg.GetSlug())
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&planningv1.MaterializePlannedScenarioResponse{
		Slug:         result.Slug,
		WrittenPaths: result.WrittenPaths,
		Generated:    result.Generated,
	}), nil
}

func toConnectError(err error) error {
	var invalid planningdomain.ErrInvalidArgument
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w", err))
	}
	var missing planningdomain.ErrScenarioNotFound
	if errors.As(err, &missing) {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("%w", err))
	}
	var missingFile planningdomain.ErrProtoFileNotFound
	if errors.As(err, &missingFile) {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("%w", err))
	}
	return connect.NewError(connect.CodeInternal, fmt.Errorf("%w", err))
}

func scenarioToProto(s planningdomain.Scenario) *planningv1.PlannedScenario {
	out := &planningv1.PlannedScenario{
		Slug:            s.Slug,
		DisplayName:     s.DisplayName,
		Sector:          s.Sector,
		Tier:            s.Tier,
		TargetStability: s.TargetStability,
		CreatedAt:       s.CreatedAt.Format(timeFormat),
		UpdatedAt:       s.UpdatedAt.Format(timeFormat),
		Files:           make([]*planningv1.PlannedProtoFile, 0, len(s.Files)),
	}
	for _, file := range s.Files {
		out.Files = append(out.Files, fileToProto(file))
	}
	return out
}

func fileToProto(file planningdomain.ProtoFile) *planningv1.PlannedProtoFile {
	return &planningv1.PlannedProtoFile{
		Path:      file.Path,
		Text:      file.Text,
		UpdatedAt: file.UpdatedAt.Format(timeFormat),
	}
}

func findingToProto(f planningdomain.PlanFinding) *planningv1.PlanFinding {
	return &planningv1.PlanFinding{
		Severity:   f.Severity,
		Code:       f.Code,
		Location:   f.Location,
		Message:    f.Message,
		Suggestion: f.Suggestion,
	}
}
