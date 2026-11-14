package catalog

import (
	"context"
	"errors"
	"log"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	domain "token-economy/internal/catalog"

	accessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access"
)

type connectHandler struct {
	service domain.Service
	logger  *log.Logger
}

func NewConnectHandler(service domain.Service, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{service: service, logger: logger}
}

func (h *connectHandler) CreateCatalogEntry(ctx context.Context, req *connect.Request[accessv1.CreateCatalogEntryRequest]) (*connect.Response[accessv1.CreateCatalogEntryResponse], error) {
	if req.Msg.Entry == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("catalog entry is required"))
	}
	entry, err := h.service.Create(ctx, inputFromProto(req.Msg.Entry), req.Msg.IdempotencyKey)
	if err != nil {
		return nil, h.mapError("CreateCatalogEntry", err)
	}
	return connect.NewResponse(&accessv1.CreateCatalogEntryResponse{Entry: entryToProto(entry)}), nil
}

func (h *connectHandler) UpdateCatalogEntry(ctx context.Context, req *connect.Request[accessv1.UpdateCatalogEntryRequest]) (*connect.Response[accessv1.UpdateCatalogEntryResponse], error) {
	if req.Msg.Entry == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("catalog entry is required"))
	}
	entry, err := h.service.Update(ctx, inputFromProto(req.Msg.Entry), req.Msg.IdempotencyKey)
	if err != nil {
		return nil, h.mapError("UpdateCatalogEntry", err)
	}
	return connect.NewResponse(&accessv1.UpdateCatalogEntryResponse{Entry: entryToProto(entry)}), nil
}

func (h *connectHandler) GetCatalogEntry(ctx context.Context, req *connect.Request[accessv1.GetCatalogEntryRequest]) (*connect.Response[accessv1.GetCatalogEntryResponse], error) {
	entry, err := h.service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.mapError("GetCatalogEntry", err)
	}
	return connect.NewResponse(&accessv1.GetCatalogEntryResponse{Entry: entryToProto(entry)}), nil
}

func (h *connectHandler) ListCatalogEntries(ctx context.Context, req *connect.Request[accessv1.ListCatalogEntriesRequest]) (*connect.Response[accessv1.ListCatalogEntriesResponse], error) {
	entries, err := h.service.List(ctx, req.Msg.IncludeRetired)
	if err != nil {
		return nil, h.mapError("ListCatalogEntries", err)
	}
	return connect.NewResponse(&accessv1.ListCatalogEntriesResponse{Entries: entriesToProto(entries)}), nil
}

func (h *connectHandler) RetireCatalogEntry(ctx context.Context, req *connect.Request[accessv1.RetireCatalogEntryRequest]) (*connect.Response[accessv1.RetireCatalogEntryResponse], error) {
	entry, err := h.service.Retire(ctx, req.Msg.Id, req.Msg.IdempotencyKey)
	if err != nil {
		return nil, h.mapError("RetireCatalogEntry", err)
	}
	return connect.NewResponse(&accessv1.RetireCatalogEntryResponse{Entry: entryToProto(entry)}), nil
}

func (h *connectHandler) BrowseCatalog(ctx context.Context, _ *connect.Request[accessv1.BrowseCatalogRequest]) (*connect.Response[accessv1.BrowseCatalogResponse], error) {
	entries, err := h.service.ListAvailable(ctx)
	if err != nil {
		return nil, h.mapError("BrowseCatalog", err)
	}
	return connect.NewResponse(&accessv1.BrowseCatalogResponse{Entries: entriesToProto(entries)}), nil
}

func (h *connectHandler) RequireAvailable(ctx context.Context, catalogEntryID string) error {
	_, err := h.service.RequireAvailable(ctx, catalogEntryID)
	if err != nil {
		return h.mapError("RequireAvailable", err)
	}
	return nil
}

func (h *connectHandler) mapError(operation string, err error) error {
	var invalid *domain.InvalidCatalogError
	var unavailable *domain.UnavailableCatalogError
	switch {
	case errors.Is(err, domain.ErrEntryNotFound), errors.Is(err, domain.ErrTokenTypeNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, domain.ErrTokenTypeRetired), errors.As(err, &unavailable):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.As(err, &invalid):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		h.logger.Printf("catalog.%s: %v", operation, err)
		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
}

func inputFromProto(value *accessv1.CatalogEntry) domain.Input {
	return domain.Input{
		ID: value.Id, TokenTypeID: value.TokenTypeId, Title: value.Title,
		Description: value.Description, CostAmount: value.CostAmount,
		Availability:    availabilityFromProto(value.Availability),
		ApprovalPosture: approvalPostureFromProto(value.ApprovalPosture),
	}
}

func availabilityFromProto(value *accessv1.Availability) domain.Availability {
	if value == nil {
		return domain.Availability{}
	}
	return domain.Availability{
		AvailableFrom:     timestampPointer(value.AvailableFrom),
		AvailableUntil:    timestampPointer(value.AvailableUntil),
		RemainingQuantity: value.RemainingQuantity,
	}
}

func timestampPointer(value *timestamppb.Timestamp) *time.Time {
	if value == nil {
		return nil
	}
	parsed := value.AsTime()
	return &parsed
}

func approvalPostureFromProto(value accessv1.ApprovalPosture) domain.ApprovalPosture {
	switch value {
	case accessv1.ApprovalPosture_APPROVAL_POSTURE_IMMEDIATE:
		return domain.ApprovalPostureImmediate
	case accessv1.ApprovalPosture_APPROVAL_POSTURE_REQUIRES_APPROVAL:
		return domain.ApprovalPostureRequiresApproval
	default:
		return ""
	}
}

func approvalPostureToProto(value domain.ApprovalPosture) accessv1.ApprovalPosture {
	switch value {
	case domain.ApprovalPostureImmediate:
		return accessv1.ApprovalPosture_APPROVAL_POSTURE_IMMEDIATE
	case domain.ApprovalPostureRequiresApproval:
		return accessv1.ApprovalPosture_APPROVAL_POSTURE_REQUIRES_APPROVAL
	default:
		return accessv1.ApprovalPosture_APPROVAL_POSTURE_UNSPECIFIED
	}
}

func entryToProto(value domain.Entry) *accessv1.CatalogEntry {
	out := &accessv1.CatalogEntry{
		Id: value.ID, TokenTypeId: value.TokenTypeID, Title: value.Title,
		Description: value.Description, CostAmount: value.CostAmount,
		ApprovalPosture: approvalPostureToProto(value.ApprovalPosture),
		Retired:         value.Retired, CreatedAt: timestamppb.New(value.CreatedAt),
		UpdatedAt:    timestamppb.New(value.UpdatedAt),
		Availability: &accessv1.Availability{RemainingQuantity: value.Availability.RemainingQuantity},
	}
	if value.Availability.AvailableFrom != nil {
		out.Availability.AvailableFrom = timestamppb.New(*value.Availability.AvailableFrom)
	}
	if value.Availability.AvailableUntil != nil {
		out.Availability.AvailableUntil = timestamppb.New(*value.Availability.AvailableUntil)
	}
	if value.RetiredAt != nil {
		out.RetiredAt = timestamppb.New(*value.RetiredAt)
	}
	return out
}

func entriesToProto(values []domain.Entry) []*accessv1.CatalogEntry {
	out := make([]*accessv1.CatalogEntry, 0, len(values))
	for _, value := range values {
		out = append(out, entryToProto(value))
	}
	return out
}
