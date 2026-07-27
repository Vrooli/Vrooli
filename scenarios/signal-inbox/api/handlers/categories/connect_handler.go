package categories

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	categoriesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/categories"
	internal "signal-inbox/internal/categories"
)

type connectHandler struct{ service *internal.Service }

func NewConnectHandler(service *internal.Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) CreateCategory(ctx context.Context, req *connect.Request[categoriesv1.CreateCategoryRequest]) (*connect.Response[categoriesv1.CreateCategoryResponse], error) {
	category, err := h.service.Create(ctx, req.Msg.Name, req.Msg.Description)
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&categoriesv1.CreateCategoryResponse{Category: categoryToProto(category)}), nil
}

func (h *connectHandler) ListCategories(ctx context.Context, req *connect.Request[categoriesv1.ListCategoriesRequest]) (*connect.Response[categoriesv1.ListCategoriesResponse], error) {
	categories, err := h.service.List(ctx, req.Msg.IncludeRetired)
	if err != nil {
		return nil, toConnectError(err)
	}
	response := &categoriesv1.ListCategoriesResponse{Categories: make([]*categoriesv1.Category, 0, len(categories))}
	for _, category := range categories {
		response.Categories = append(response.Categories, categoryToProto(category))
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) RenameCategory(ctx context.Context, req *connect.Request[categoriesv1.RenameCategoryRequest]) (*connect.Response[categoriesv1.RenameCategoryResponse], error) {
	category, err := h.service.Rename(ctx, req.Msg.Id, req.Msg.Name, req.Msg.Description)
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&categoriesv1.RenameCategoryResponse{Category: categoryToProto(category)}), nil
}

func (h *connectHandler) RetireCategory(ctx context.Context, req *connect.Request[categoriesv1.RetireCategoryRequest]) (*connect.Response[categoriesv1.RetireCategoryResponse], error) {
	category, err := h.service.Retire(ctx, req.Msg.Id)
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&categoriesv1.RetireCategoryResponse{Category: categoryToProto(category)}), nil
}

func (h *connectHandler) GetClassification(ctx context.Context, req *connect.Request[categoriesv1.GetClassificationRequest]) (*connect.Response[categoriesv1.GetClassificationResponse], error) {
	classification, found, err := h.service.GetClassification(ctx, req.Msg.SignalId)
	if err != nil {
		return nil, toConnectError(err)
	}
	if !found {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("classification not found"))
	}
	return connect.NewResponse(&categoriesv1.GetClassificationResponse{Classification: classificationToProto(classification)}), nil
}

func (h *connectHandler) ConfirmClassification(ctx context.Context, req *connect.Request[categoriesv1.ConfirmClassificationRequest]) (*connect.Response[categoriesv1.ConfirmClassificationResponse], error) {
	classification, err := h.service.Confirm(ctx, req.Msg.SignalId, req.Msg.CategoryId)
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&categoriesv1.ConfirmClassificationResponse{Classification: classificationToProto(classification)}), nil
}

func categoryToProto(category internal.Category) *categoriesv1.Category {
	result := &categoriesv1.Category{Id: category.ID, Name: category.Name, Description: category.Description, Reserved: category.Reserved, CreatedAt: timestamppb.New(category.CreatedAt)}
	if category.RetiredAt != nil {
		result.RetiredAt = timestamppb.New(*category.RetiredAt)
	}
	return result
}

func classificationToProto(classification internal.Classification) *categoriesv1.Classification {
	return &categoriesv1.Classification{Id: classification.ID, SignalId: classification.SignalID, ProposedCategoryId: classification.ProposedCategoryID, ProposedConfidence: classification.ProposedConfidence, Model: classification.Model, ConfirmedCategoryId: classification.ConfirmedCategoryID, State: stateToProto(classification.State), Reason: classification.Reason, CreatedAt: timestamppb.New(classification.CreatedAt)}
}

func stateToProto(state internal.ClassificationState) categoriesv1.ClassificationState {
	switch state {
	case internal.StateProposed:
		return categoriesv1.ClassificationState_CLASSIFICATION_STATE_PROPOSED
	case internal.StateConfirmed:
		return categoriesv1.ClassificationState_CLASSIFICATION_STATE_CONFIRMED
	case internal.StateOverridden:
		return categoriesv1.ClassificationState_CLASSIFICATION_STATE_OVERRIDDEN
	case internal.StateUncategorized:
		return categoriesv1.ClassificationState_CLASSIFICATION_STATE_UNCATEGORIZED
	default:
		return categoriesv1.ClassificationState_CLASSIFICATION_STATE_UNSPECIFIED
	}
}

func toConnectError(err error) error {
	var invalid internal.ErrInvalidCategory
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	var reserved internal.ErrReservedCategory
	if errors.As(err, &reserved) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	var missing internal.ErrCategoryNotFound
	if errors.As(err, &missing) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
