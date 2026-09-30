package portability

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/portability"
	internalPlanning "nutrition-planner/internal/planning"
	"nutrition-planner/internal/portability"
	internalPortability "nutrition-planner/internal/portability"
	internalRecipe "nutrition-planner/internal/recipe"
	internalShopping "nutrition-planner/internal/shopping"
	internalWorkspace "nutrition-planner/internal/workspace"
)

type connectHandler struct {
	recipes    internalRecipe.Service
	workspaces internalWorkspace.Service
	plans      internalPlanning.Repository
	shopping   internalShopping.Repository
	restorer   internalPortability.Restorer
	logger     *log.Logger
}

func NewConnectHandler(recipes internalRecipe.Service, workspaces internalWorkspace.Service, plans internalPlanning.Repository, shoppingRepo internalShopping.Repository, logger *log.Logger) *connectHandler {
	return NewConnectHandlerWithRestorer(recipes, workspaces, plans, shoppingRepo, nil, logger)
}

func NewConnectHandlerWithRestorer(recipes internalRecipe.Service, workspaces internalWorkspace.Service, plans internalPlanning.Repository, shoppingRepo internalShopping.Repository, restorer internalPortability.Restorer, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{recipes: recipes, workspaces: workspaces, plans: plans, shopping: shoppingRepo, restorer: restorer, logger: logger}
}

func (h *connectHandler) ExportGroceriesCSV(ctx context.Context, req *connect.Request[v1.ExportGroceriesCSVRequest]) (*connect.Response[v1.ExportGroceriesCSVResponse], error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if req.Msg.WorkspaceId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id is required"))
	}
	if _, err := h.workspaces.Get(ctx, req.Msg.WorkspaceId, principal.Subject); err != nil {
		var nf internalWorkspace.ErrNotFound
		if errors.As(err, &nf) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		var forbidden internalWorkspace.ErrForbidden
		if errors.As(err, &forbidden) {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	revision, raw, err := h.plans.Get(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if req.Msg.ExpectedRevision >= 0 && req.Msg.ExpectedRevision != revision {
		return nil, connect.NewError(connect.CodeAborted, errors.New("plan revision is stale"))
	}
	var draft internalPlanning.Draft
	if err := json.Unmarshal([]byte(raw), &draft); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	recipes, err := h.recipes.List(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	checked, err := h.shopping.Checked(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	lines := internalShopping.Derive(draft, recipes, checked)
	rows := make([]internalPortability.GroceryRow, 0, len(lines))
	for _, line := range lines {
		rows = append(rows, internalPortability.GroceryRow{Key: line.Key, Label: line.Label, Need: line.Need, Stock: line.Stock, Missing: line.Missing, PackageCount: line.PackageCount, Price: line.Price, Checked: line.Checked})
	}
	content, err := internalPortability.GroceriesCSV(rows)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ExportGroceriesCSVResponse{Filename: "daily-groceries.csv", ContentCsv: string(content), Revision: revision}), nil
}

func (h *connectHandler) ExportRecipes(ctx context.Context, req *connect.Request[v1.ExportRecipesRequest]) (*connect.Response[v1.ExportRecipesResponse], error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if req.Msg.WorkspaceId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id is required"))
	}
	if _, err := h.workspaces.Get(ctx, req.Msg.WorkspaceId, principal.Subject); err != nil {
		var nf internalWorkspace.ErrNotFound
		if errors.As(err, &nf) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		var forbidden internalWorkspace.ErrForbidden
		if errors.As(err, &forbidden) {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items, err := h.recipes.List(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	content, err := portability.Recipes(items)
	if err != nil {
		h.logger.Printf("recipe export: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ExportRecipesResponse{Format: "daily.recipes", SchemaVersion: 2, ContentJson: string(content), Omissions: []string{"nutrition", "cost"}}), nil
}

func (h *connectHandler) ExportWorkspace(ctx context.Context, req *connect.Request[v1.ExportWorkspaceRequest]) (*connect.Response[v1.ExportWorkspaceResponse], error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if req.Msg.WorkspaceId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id is required"))
	}
	owned, err := h.workspaces.Get(ctx, req.Msg.WorkspaceId, principal.Subject)
	if err != nil {
		return nil, workspaceError(err)
	}
	items, err := h.recipes.List(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	planRevision, planJSON, err := h.plans.Get(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	content, err := internalPortability.Workspace(internalPortability.WorkspaceSnapshot{ID: owned.ID, Name: owned.Name, Revision: owned.Revision, Recipes: items, PlanRevision: planRevision, PlanJSON: planJSON})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var envelope internalPortability.Export
	if err := json.Unmarshal(content, &envelope); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ExportWorkspaceResponse{Format: envelope.Format, SchemaVersion: int32(envelope.SchemaVersion), ContentJson: string(content), Omissions: envelope.Manifest.Omissions, PlanRevision: planRevision}), nil
}

func (h *connectHandler) PreviewWorkspaceImport(ctx context.Context, req *connect.Request[v1.PreviewWorkspaceImportRequest]) (*connect.Response[v1.PreviewWorkspaceImportResponse], error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if req.Msg.WorkspaceId == "" || req.Msg.ContentJson == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id and content_json are required"))
	}
	if _, err := h.workspaces.Get(ctx, req.Msg.WorkspaceId, principal.Subject); err != nil {
		return nil, workspaceError(err)
	}
	envelope, err := internalPortability.ImportWorkspace([]byte(req.Msg.ContentJson))
	if err != nil {
		return connect.NewResponse(&v1.PreviewWorkspaceImportResponse{Valid: false, Errors: []string{err.Error()}}), nil
	}
	kinds := make([]string, 0, len(envelope.Manifest.RecordKinds))
	kinds = append(kinds, envelope.Manifest.RecordKinds...)
	return connect.NewResponse(&v1.PreviewWorkspaceImportResponse{
		Valid:         true,
		Format:        envelope.Format,
		SchemaVersion: int32(envelope.SchemaVersion),
		RecordCount:   int32(envelope.Manifest.RecordCount),
		RecordKinds:   kinds,
		Omissions:     envelope.Manifest.Omissions,
	}), nil
}

func (h *connectHandler) ApplyWorkspaceImport(ctx context.Context, req *connect.Request[v1.ApplyWorkspaceImportRequest]) (*connect.Response[v1.ApplyWorkspaceImportResponse], error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if req.Msg.WorkspaceId == "" || req.Msg.ContentJson == "" || req.Msg.IdempotencyKey == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id, content_json, and idempotency_key are required"))
	}
	if h.restorer == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("workspace restore is not configured"))
	}
	if _, err := h.workspaces.Get(ctx, req.Msg.WorkspaceId, principal.Subject); err != nil {
		return nil, workspaceError(err)
	}
	envelope, err := internalPortability.ImportWorkspace([]byte(req.Msg.ContentJson))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	result, err := h.restorer.ApplyWorkspace(ctx, req.Msg.WorkspaceId, req.Msg.ExpectedWorkspaceRevision, req.Msg.IdempotencyKey, envelope)
	if err != nil {
		var stale internalPortability.ErrRestoreRevision
		if errors.As(err, &stale) {
			return nil, connect.NewError(connect.CodeAborted, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ApplyWorkspaceImportResponse{WorkspaceRevision: result.WorkspaceRevision, RecipesApplied: int32(result.RecipesApplied), CheckpointId: result.CheckpointID}), nil
}

func (h *connectHandler) PreviewRecipesImport(ctx context.Context, req *connect.Request[v1.PreviewRecipesImportRequest]) (*connect.Response[v1.PreviewRecipesImportResponse], error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if req.Msg.WorkspaceId == "" || req.Msg.ContentJson == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id and content_json are required"))
	}
	if h.restorer == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("recipe import is not configured"))
	}
	if _, err := h.workspaces.Get(ctx, req.Msg.WorkspaceId, principal.Subject); err != nil {
		return nil, workspaceError(err)
	}
	envelope, err := internalPortability.ImportRecipes([]byte(req.Msg.ContentJson))
	if err != nil {
		return connect.NewResponse(&v1.PreviewRecipesImportResponse{Valid: false, Errors: []string{err.Error()}}), nil
	}
	preview, err := h.restorer.PreviewRecipes(ctx, req.Msg.WorkspaceId, envelope)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.PreviewRecipesImportResponse{Valid: len(preview.Errors) == 0, Format: envelope.Format, SchemaVersion: int32(envelope.SchemaVersion), RecipeCount: int32(preview.RecipeCount), DuplicateCount: int32(preview.DuplicateCount), ConflictCount: int32(preview.ConflictCount), Errors: preview.Errors}), nil
}

func (h *connectHandler) ApplyRecipesImport(ctx context.Context, req *connect.Request[v1.ApplyRecipesImportRequest]) (*connect.Response[v1.ApplyRecipesImportResponse], error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if req.Msg.WorkspaceId == "" || req.Msg.ContentJson == "" || req.Msg.IdempotencyKey == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id, content_json, and idempotency_key are required"))
	}
	if h.restorer == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("recipe import is not configured"))
	}
	if _, err := h.workspaces.Get(ctx, req.Msg.WorkspaceId, principal.Subject); err != nil {
		return nil, workspaceError(err)
	}
	envelope, err := internalPortability.ImportRecipes([]byte(req.Msg.ContentJson))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	result, err := h.restorer.ApplyRecipes(ctx, req.Msg.WorkspaceId, req.Msg.ExpectedWorkspaceRevision, req.Msg.IdempotencyKey, req.Msg.ConflictPolicy, envelope)
	if err != nil {
		var stale internalPortability.ErrRestoreRevision
		if errors.As(err, &stale) {
			return nil, connect.NewError(connect.CodeAborted, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ApplyRecipesImportResponse{WorkspaceRevision: result.WorkspaceRevision, RecipesApplied: int32(result.RecipesApplied), RecipesSkipped: int32(result.RecipesSkipped), RemappedIds: result.RemappedIDs}), nil
}

func (h *connectHandler) ExportRecipePDF(ctx context.Context, req *connect.Request[v1.ExportRecipePDFRequest]) (*connect.Response[v1.ExportPDFResponse], error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if req.Msg.WorkspaceId == "" || req.Msg.RecipeId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id and recipe_id are required"))
	}
	if _, err := h.workspaces.Get(ctx, req.Msg.WorkspaceId, principal.Subject); err != nil {
		return nil, workspaceError(err)
	}
	item, err := h.recipes.Get(ctx, req.Msg.WorkspaceId, req.Msg.RecipeId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	content, err := internalPortability.RecipePDFWithPageSize(item, req.Msg.PageSize)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ExportPDFResponse{Filename: "recipe-" + item.ID + ".pdf", Content: content}), nil
}

func (h *connectHandler) ExportWeeklyPDF(ctx context.Context, req *connect.Request[v1.ExportWeeklyPDFRequest]) (*connect.Response[v1.ExportPDFResponse], error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if req.Msg.WorkspaceId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id is required"))
	}
	if _, err := h.workspaces.Get(ctx, req.Msg.WorkspaceId, principal.Subject); err != nil {
		return nil, workspaceError(err)
	}
	revision, raw, err := h.plans.Get(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if req.Msg.ExpectedRevision >= 0 && req.Msg.ExpectedRevision != revision {
		return nil, connect.NewError(connect.CodeAborted, errors.New("plan revision is stale"))
	}
	var draft internalPlanning.Draft
	if err := json.Unmarshal([]byte(raw), &draft); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items, err := h.recipes.List(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	checked, err := h.shopping.Checked(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	content, err := internalPortability.WeeklyPDFWithPageSize(draft, internalShopping.Derive(draft, items, checked), req.Msg.PageSize)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ExportPDFResponse{Filename: "weekly-nutrition-plan.pdf", Content: content, Revision: revision}), nil
}

func workspaceError(err error) error {
	var nf internalWorkspace.ErrNotFound
	if errors.As(err, &nf) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	var forbidden internalWorkspace.ErrForbidden
	if errors.As(err, &forbidden) {
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
