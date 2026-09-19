package catalog

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/catalog"
	internal "nutrition-planner/internal/catalog"
	"nutrition-planner/internal/decimalx"
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

func (h *connectHandler) scope(ctx context.Context, workspaceID string) error {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if workspaceID == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id is required"))
	}
	if _, err := h.ws.Get(ctx, workspaceID, p.Subject); err != nil {
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

func (h *connectHandler) ListCatalog(ctx context.Context, req *connect.Request[v1.ListCatalogRequest]) (*connect.Response[v1.ListCatalogResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	items, err := h.repo.List(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListCatalogResponse{Revisions: make([]*v1.CatalogRevision, 0, len(items))}
	for _, item := range items {
		out.Revisions = append(out.Revisions, toProto(item))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CreateRevision(ctx context.Context, req *connect.Request[v1.CreateRevisionRequest]) (*connect.Response[v1.CreateRevisionResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	v, err := fromProto(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	created, err := h.repo.Create(ctx, v)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.CreateRevisionResponse{Revision: toProto(created)}), nil
}

func (h *connectHandler) GetRevision(ctx context.Context, req *connect.Request[v1.GetRevisionRequest]) (*connect.Response[v1.GetRevisionResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	item, err := h.repo.Get(ctx, req.Msg.Id, req.Msg.WorkspaceId, req.Msg.Revision)
	if err != nil {
		var nf internal.ErrNotFound
		if errors.As(err, &nf) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetRevisionResponse{Revision: toProto(item)}), nil
}

func fromProto(req *v1.CreateRevisionRequest) (internal.Revision, error) {
	serving, err := decimalx.Parse(req.ServingQuantity)
	if err != nil {
		return internal.Revision{}, errors.New("serving_quantity: invalid decimal")
	}
	nutrients := make([]internal.NutrientValue, 0, len(req.Nutrients))
	for _, n := range req.Nutrients {
		if n == nil {
			continue
		}
		amount, err := decimalx.Parse(n.Amount)
		if err != nil {
			return internal.Revision{}, errors.New("nutrient amount: invalid decimal")
		}
		basis, err := decimalx.Parse(n.Basis)
		if err != nil {
			return internal.Revision{}, errors.New("nutrient basis: invalid decimal")
		}
		nutrients = append(nutrients, internal.NutrientValue{NutrientID: n.NutrientId, Amount: amount, Unit: n.Unit, Basis: basis, BasisUnit: n.BasisUnit, Evidence: internal.EvidenceKind(n.Evidence), SourceRef: n.SourceRef})
	}
	return internal.Revision{WorkspaceID: req.WorkspaceId, ConceptID: req.ConceptId, Name: req.Name, ProductName: req.ProductName, Preparation: req.Preparation, ServingQuantity: serving, ServingUnit: req.ServingUnit, Nutrients: nutrients, AllergenEvidence: req.AllergenEvidence, SourceType: req.SourceType, SourceRef: req.SourceRef}, nil
}

func toProto(v internal.Revision) *v1.CatalogRevision {
	out := &v1.CatalogRevision{Id: v.ID, Revision: v.Revision, ConceptId: v.ConceptID, Name: v.Name, ProductName: v.ProductName, Preparation: v.Preparation, ServingQuantity: v.ServingQuantity.String(), ServingUnit: v.ServingUnit, AllergenEvidence: v.AllergenEvidence, SourceType: v.SourceType, SourceRef: v.SourceRef, CreatedAt: v.CreatedAt.Format("2006-01-02T15:04:05.999999999Z07:00")}
	out.Nutrients = make([]*v1.NutrientValue, 0, len(v.Nutrients))
	for _, n := range v.Nutrients {
		out.Nutrients = append(out.Nutrients, &v1.NutrientValue{NutrientId: n.NutrientID, Amount: n.Amount.String(), Unit: n.Unit, Basis: n.Basis.String(), BasisUnit: n.BasisUnit, Evidence: string(n.Evidence), SourceRef: n.SourceRef})
	}
	return out
}
