package cost

import (
	"context"
	"errors"
	"log"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/cost"
	internal "nutrition-planner/internal/cost"
	"nutrition-planner/internal/decimalx"
	"nutrition-planner/internal/money"
	"nutrition-planner/internal/workspace"
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

func (h *connectHandler) ListPriceObservations(ctx context.Context, req *connect.Request[v1.ListPriceObservationsRequest]) (*connect.Response[v1.ListPriceObservationsResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	items, err := h.repo.List(ctx, req.Msg.WorkspaceId, req.Msg.ItemId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListPriceObservationsResponse{Observations: make([]*v1.PriceObservation, 0, len(items))}
	for _, item := range items {
		out.Observations = append(out.Observations, toProto(item))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CreatePriceObservation(ctx context.Context, req *connect.Request[v1.CreatePriceObservationRequest]) (*connect.Response[v1.CreatePriceObservationResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	amount, err := decimalx.Parse(req.Msg.PackageAmount)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	observed, err := time.Parse(time.RFC3339Nano, req.Msg.ObservedAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("observed_at must be RFC3339"))
	}
	v := internal.Observation{WorkspaceID: req.Msg.WorkspaceId, ItemID: req.Msg.ItemId, ProductID: req.Msg.ProductId, PackageLabel: req.Msg.PackageLabel, PackageAmount: amount, PackageUnit: req.Msg.PackageUnit, Retailer: req.Msg.Retailer, ObservedAt: observed, Available: req.Msg.Available, MembershipRequired: req.Msg.MembershipRequired, CouponRequired: req.Msg.CouponRequired, MinimumBuy: req.Msg.MinimumBuy, Source: req.Msg.Source}
	v.Price, err = money.New(req.Msg.PriceMinor, req.Msg.Currency, int(req.Msg.CurrencyExponent))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if req.Msg.ValidThrough != "" {
		v.ValidThrough, err = time.Parse(time.RFC3339Nano, req.Msg.ValidThrough)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("valid_through must be RFC3339"))
		}
	}
	created, err := h.repo.Create(ctx, v)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.CreatePriceObservationResponse{Observation: toProto(created)}), nil
}

func toProto(v internal.Observation) *v1.PriceObservation {
	out := &v1.PriceObservation{Id: v.ID, WorkspaceId: v.WorkspaceID, ItemId: v.ItemID, ProductId: v.ProductID, PackageLabel: v.PackageLabel, PackageAmount: v.PackageAmount.String(), PackageUnit: v.PackageUnit, PriceMinor: v.Price.Minor, Currency: v.Price.Currency, CurrencyExponent: int32(v.Price.Exponent), Retailer: v.Retailer, ObservedAt: v.ObservedAt.UTC().Format(time.RFC3339Nano), Available: v.Available, MembershipRequired: v.MembershipRequired, CouponRequired: v.CouponRequired, MinimumBuy: v.MinimumBuy, Source: v.Source}
	if !v.ValidThrough.IsZero() {
		out.ValidThrough = v.ValidThrough.UTC().Format(time.RFC3339Nano)
	}
	return out
}
