package routine

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/routine"
	"nutrition-planner/internal/decimalx"
	internal "nutrition-planner/internal/routine"
	workspace "nutrition-planner/internal/workspace"
)

type connectHandler struct {
	repo   internal.Repository
	ws     workspace.Service
	logger *log.Logger
}

func NewConnectHandler(repo internal.Repository, ws workspace.Service, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{repo: repo, ws: ws, logger: logger}
}
func (h *connectHandler) scope(ctx context.Context, id string) error {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if id == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id is required"))
	}
	if _, err := h.ws.Get(ctx, id, p.Subject); err != nil {
		var nf workspace.ErrNotFound
		if errors.As(err, &nf) {
			return connect.NewError(connect.CodeNotFound, err)
		}
		var forbidden workspace.ErrForbidden
		if errors.As(err, &forbidden) {
			return connect.NewError(connect.CodePermissionDenied, err)
		}
		return connect.NewError(connect.CodeInternal, err)
	}
	return nil
}

func (h *connectHandler) ListTemplates(ctx context.Context, req *connect.Request[v1.ListTemplatesRequest]) (*connect.Response[v1.ListTemplatesResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	items, err := h.repo.List(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListTemplatesResponse{Templates: make([]*v1.Template, 0, len(items))}
	for _, item := range items {
		out.Templates = append(out.Templates, toProto(item))
	}
	return connect.NewResponse(out), nil
}
func (h *connectHandler) CreateTemplate(ctx context.Context, req *connect.Request[v1.CreateTemplateRequest]) (*connect.Response[v1.CreateTemplateResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	t, err := fromCreate(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	created, err := h.repo.Create(ctx, t)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.CreateTemplateResponse{Template: toProto(created)}), nil
}
func (h *connectHandler) UpdateTemplate(ctx context.Context, req *connect.Request[v1.UpdateTemplateRequest]) (*connect.Response[v1.UpdateTemplateResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	q, err := decimalx.Parse(req.Msg.Quantity)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("quantity: invalid decimal"))
	}
	updated, err := h.repo.Update(ctx, internal.UpdateInput{WorkspaceID: req.Msg.WorkspaceId, ID: req.Msg.Id, ExpectedRevision: req.Msg.ExpectedRevision, SlotName: req.Msg.SlotName, RecipeID: req.Msg.RecipeId, Quantity: q, Weekdays: ints(req.Msg.Weekdays), StartDate: req.Msg.StartDate, EndDate: req.Msg.EndDate, Mode: req.Msg.Mode, Active: req.Msg.Active})
	if err != nil {
		var conflict internal.ErrConflict
		if errors.As(err, &conflict) {
			return nil, connect.NewError(connect.CodeAborted, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.UpdateTemplateResponse{Template: toProto(updated)}), nil
}
func (h *connectHandler) GenerateOccurrences(ctx context.Context, req *connect.Request[v1.GenerateOccurrencesRequest]) (*connect.Response[v1.GenerateOccurrencesResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	templates, err := h.repo.List(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	occurrences, err := internal.Generate(templates, req.Msg.FromDate, req.Msg.ToDate)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := &v1.GenerateOccurrencesResponse{Occurrences: make([]*v1.Occurrence, 0, len(occurrences))}
	for _, occurrence := range occurrences {
		out.Occurrences = append(out.Occurrences, &v1.Occurrence{Date: occurrence.Date, SlotName: occurrence.SlotName, RecipeId: occurrence.RecipeID, Quantity: occurrence.Quantity.String(), TemplateId: occurrence.TemplateID, TemplateRevision: occurrence.TemplateRevision, Mode: occurrence.Mode})
	}
	return connect.NewResponse(out), nil
}

func fromCreate(req *v1.CreateTemplateRequest) (internal.Template, error) {
	q, err := decimalx.Parse(req.Quantity)
	if err != nil {
		return internal.Template{}, errors.New("quantity: invalid decimal")
	}
	return internal.Template{WorkspaceID: req.WorkspaceId, SlotName: req.SlotName, RecipeID: req.RecipeId, Quantity: q, Weekdays: ints(req.Weekdays), StartDate: req.StartDate, EndDate: req.EndDate, Mode: req.Mode, Active: req.Active}, nil
}
func toProto(t internal.Template) *v1.Template {
	return &v1.Template{Id: t.ID, Revision: t.Revision, SlotName: t.SlotName, RecipeId: t.RecipeID, Quantity: t.Quantity.String(), Weekdays: ints32(t.Weekdays), StartDate: t.StartDate, EndDate: t.EndDate, Mode: t.Mode, Active: t.Active}
}
func ints(v []int32) []int {
	out := make([]int, 0, len(v))
	for _, item := range v {
		out = append(out, int(item))
	}
	return out
}
func ints32(v []int) []int32 {
	out := make([]int32, 0, len(v))
	for _, item := range v {
		out = append(out, int32(item))
	}
	return out
}
