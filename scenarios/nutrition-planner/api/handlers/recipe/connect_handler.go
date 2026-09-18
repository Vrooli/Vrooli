package recipe

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/identity"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/recipe"
	"google.golang.org/protobuf/types/known/timestamppb"

	internal "nutrition-planner/internal/recipe"
	workspace "nutrition-planner/internal/workspace"
)

type connectHandler struct {
	s      internal.Service
	ws     workspace.Service
	logger *log.Logger
}

func NewConnectHandler(s internal.Service, ws workspace.Service, l *log.Logger) *connectHandler {
	if l == nil {
		l = log.Default()
	}
	return &connectHandler{s: s, ws: ws, logger: l}
}

func actor(ctx context.Context) (string, error) {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return "", connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	return p.Subject, nil
}

func (h *connectHandler) scope(ctx context.Context, workspaceID string) (string, error) {
	owner, e := actor(ctx)
	if e != nil {
		return "", e
	}
	if workspaceID == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id is required"))
	}
	_, e = h.ws.Get(ctx, workspaceID, owner)
	if e != nil {
		var nf workspace.ErrNotFound
		if errors.As(e, &nf) {
			return "", connect.NewError(connect.CodeNotFound, e)
		}
		var f workspace.ErrForbidden
		if errors.As(e, &f) {
			return "", connect.NewError(connect.CodePermissionDenied, e)
		}
		return "", connect.NewError(connect.CodeInternal, e)
	}
	return workspaceID, nil
}

func (h *connectHandler) ListRecipes(ctx context.Context, q *connect.Request[v1.ListRecipesRequest]) (*connect.Response[v1.ListRecipesResponse], error) {
	w, e := h.scope(ctx, q.Msg.WorkspaceId)
	if e != nil {
		return nil, e
	}
	items, e := h.s.List(ctx, w)
	if e != nil {
		return nil, connect.NewError(connect.CodeInternal, e)
	}
	out := &v1.ListRecipesResponse{Recipes: make([]*v1.Recipe, 0, len(items))}
	for _, r := range items {
		out.Recipes = append(out.Recipes, toProto(r))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CreateRecipe(ctx context.Context, q *connect.Request[v1.CreateRecipeRequest]) (*connect.Response[v1.CreateRecipeResponse], error) {
	w, e := h.scope(ctx, q.Msg.WorkspaceId)
	if e != nil {
		return nil, e
	}
	r, e := h.s.Create(ctx, internal.CreateInput{WorkspaceID: w, Name: q.Msg.Name, Notes: q.Msg.Notes, SourceURL: q.Msg.SourceUrl, SourceType: q.Msg.SourceType, OriginalText: q.Msg.OriginalText, IdempotencyKey: q.Msg.IdempotencyKey, Methods: fromProtoMethods(q.Msg.Methods), Groups: q.Msg.Groups, RequiredAppliances: q.Msg.RequiredAppliances, AllergenEvidence: q.Msg.AllergenEvidence})
	if e != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, e)
	}
	return connect.NewResponse(&v1.CreateRecipeResponse{Recipe: toProto(r)}), nil
}

func (h *connectHandler) GetRecipe(ctx context.Context, q *connect.Request[v1.GetRecipeRequest]) (*connect.Response[v1.GetRecipeResponse], error) {
	w, e := h.scope(ctx, q.Msg.WorkspaceId)
	if e != nil {
		return nil, e
	}
	r, e := h.s.Get(ctx, q.Msg.Id, w)
	if e != nil {
		var nf internal.ErrNotFound
		if errors.As(e, &nf) {
			return nil, connect.NewError(connect.CodeNotFound, e)
		}
		var f internal.ErrForbidden
		if errors.As(e, &f) {
			return nil, connect.NewError(connect.CodePermissionDenied, e)
		}
		return nil, connect.NewError(connect.CodeInternal, e)
	}
	return connect.NewResponse(&v1.GetRecipeResponse{Recipe: toProto(r)}), nil
}

func (h *connectHandler) UpdateRecipe(ctx context.Context, q *connect.Request[v1.UpdateRecipeRequest]) (*connect.Response[v1.UpdateRecipeResponse], error) {
	w, e := h.scope(ctx, q.Msg.WorkspaceId)
	if e != nil {
		return nil, e
	}
	r, e := h.s.Update(ctx, internal.UpdateInput{WorkspaceID: w, ID: q.Msg.Id, ExpectedRevision: q.Msg.ExpectedRevision, Name: q.Msg.Name, Notes: q.Msg.Notes, SourceURL: q.Msg.SourceUrl, SourceType: q.Msg.SourceType, OriginalText: q.Msg.OriginalText, IdempotencyKey: q.Msg.IdempotencyKey, Methods: fromProtoMethods(q.Msg.Methods), Groups: q.Msg.Groups, RequiredAppliances: q.Msg.RequiredAppliances, AllergenEvidence: q.Msg.AllergenEvidence})
	if e != nil {
		var c internal.ErrConflict
		if errors.As(e, &c) {
			return nil, connect.NewError(connect.CodeAborted, e)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, e)
	}
	return connect.NewResponse(&v1.UpdateRecipeResponse{Recipe: toProto(r)}), nil
}

func toProto(r internal.Recipe) *v1.Recipe {
	return &v1.Recipe{Id: r.ID, Revision: r.Revision, Name: r.Name, Notes: r.Notes, SourceUrl: r.SourceURL, SourceType: r.SourceType, OriginalText: r.OriginalText, Status: r.Status, CreatedAt: timestamppb.New(r.CreatedAt), UpdatedAt: timestamppb.New(r.UpdatedAt), Methods: toProtoMethods(r.Methods), Groups: r.Groups, RequiredAppliances: r.RequiredAppliances, AllergenEvidence: r.AllergenEvidence}
}

func fromProtoMethods(methods []*v1.RecipeMethod) []internal.Method {
	out := make([]internal.Method, 0, len(methods))
	for _, method := range methods {
		if method == nil {
			continue
		}
		m := internal.Method{ID: method.Id, Name: method.Name, Steps: make([]internal.MethodStep, 0, len(method.Steps))}
		for _, step := range method.Steps {
			if step != nil {
				m.Steps = append(m.Steps, internal.MethodStep{ID: step.Id, Instruction: step.Instruction, DependsOn: step.DependsOn, Inputs: step.Inputs, Outputs: step.Outputs})
			}
		}
		out = append(out, m)
	}
	return out
}

func toProtoMethods(methods []internal.Method) []*v1.RecipeMethod {
	out := make([]*v1.RecipeMethod, 0, len(methods))
	for _, method := range methods {
		m := &v1.RecipeMethod{Id: method.ID, Name: method.Name, Steps: make([]*v1.RecipeStep, 0, len(method.Steps))}
		for _, step := range method.Steps {
			m.Steps = append(m.Steps, &v1.RecipeStep{Id: step.ID, Instruction: step.Instruction, DependsOn: step.DependsOn, Inputs: step.Inputs, Outputs: step.Outputs})
		}
		out = append(out, m)
	}
	return out
}
