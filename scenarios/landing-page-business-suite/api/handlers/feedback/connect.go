package feedback

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	metrics "landing-page-business-suite-api/internal/metrics"
)

type ConnectHandler struct {
	service  metrics.FeedbackServicer
	notifier Notifier
}

// seam: Notifier delivers feedback notifications after durable persistence. It
// is a narrow composition seam so transport tests do not need email
// infrastructure.
type Notifier interface {
	Notify(*metrics.FeedbackRequest)
}

func NewConnectHandler(service metrics.FeedbackServicer, notifier Notifier) *ConnectHandler {
	return &ConnectHandler{service: service, notifier: notifier}
}

func (h *ConnectHandler) CreateFeedback(ctx context.Context, request *connect.Request[lpbsv1.FeedbackCreateRequest]) (*connect.Response[lpbsv1.FeedbackCreateResponse], error) {
	input := request.Msg
	if strings.TrimSpace(input.GetEmail()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("email address is required"))
	}
	if strings.TrimSpace(input.GetSubject()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("subject is required"))
	}
	if strings.TrimSpace(input.GetMessage()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("message is required"))
	}
	feedbackType := input.GetType()
	if !validType(feedbackType) {
		feedbackType = "general"
	}
	var orderID *string
	if input.OrderId != nil {
		value := input.GetOrderId()
		orderID = &value
	}
	created, err := h.service.Create(ctx, &metrics.CreateFeedbackInput{Type: feedbackType, Email: input.GetEmail(), Subject: input.GetSubject(), Message: input.GetMessage(), OrderID: orderID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create feedback: %w", err))
	}
	if h.notifier != nil {
		h.notifier.Notify(created)
	}
	return connect.NewResponse(&lpbsv1.FeedbackCreateResponse{Success: true, Id: int64(created.ID)}), nil
}

func (h *ConnectHandler) ListFeedback(ctx context.Context, request *connect.Request[lpbsv1.ListFeedbackRequest]) (*connect.Response[lpbsv1.ListFeedbackResponse], error) {
	status, err := feedbackStatusFromProto(request.Msg.Status)
	if err != nil {
		return nil, err
	}
	items, err := h.service.List(ctx, status)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list feedback: %w", err))
	}
	response := &lpbsv1.ListFeedbackResponse{Feedback: make([]*lpbsv1.FeedbackRecord, 0, len(items))}
	for index := range items {
		response.Feedback = append(response.Feedback, feedbackRecordProto(&items[index]))
	}
	return connect.NewResponse(response), nil
}

func (h *ConnectHandler) GetFeedback(ctx context.Context, request *connect.Request[lpbsv1.GetFeedbackRequest]) (*connect.Response[lpbsv1.GetFeedbackResponse], error) {
	id, err := feedbackID(request.Msg.GetId())
	if err != nil {
		return nil, err
	}
	item, err := h.service.GetByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("feedback not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get feedback: %w", err))
	}
	return connect.NewResponse(&lpbsv1.GetFeedbackResponse{Feedback: feedbackRecordProto(item)}), nil
}

func (h *ConnectHandler) UpdateFeedbackStatus(ctx context.Context, request *connect.Request[lpbsv1.UpdateFeedbackStatusRequest]) (*connect.Response[lpbsv1.UpdateFeedbackStatusResponse], error) {
	id, err := feedbackID(request.Msg.GetId())
	if err != nil {
		return nil, err
	}
	status, err := feedbackStatusFromProto(&request.Msg.Status)
	if err != nil || status == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("valid feedback status is required"))
	}
	item, err := h.service.UpdateStatus(ctx, id, status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("feedback not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update feedback status: %w", err))
	}
	return connect.NewResponse(&lpbsv1.UpdateFeedbackStatusResponse{Feedback: feedbackRecordProto(item)}), nil
}

func (h *ConnectHandler) DeleteFeedback(ctx context.Context, request *connect.Request[lpbsv1.DeleteFeedbackRequest]) (*connect.Response[lpbsv1.DeleteFeedbackResponse], error) {
	id, err := feedbackID(request.Msg.GetId())
	if err != nil {
		return nil, err
	}
	if err := h.service.Delete(ctx, id); errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("feedback not found"))
	} else if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete feedback: %w", err))
	}
	return connect.NewResponse(&lpbsv1.DeleteFeedbackResponse{Deleted: true, Id: int64(id)}), nil
}

func (h *ConnectHandler) DeleteFeedbackBulk(ctx context.Context, request *connect.Request[lpbsv1.DeleteFeedbackBulkRequest]) (*connect.Response[lpbsv1.DeleteFeedbackBulkResponse], error) {
	if len(request.Msg.GetIds()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("at least one feedback ID is required"))
	}
	ids := make([]int, 0, len(request.Msg.GetIds()))
	for _, value := range request.Msg.GetIds() {
		id, err := feedbackID(value)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	deleted, err := h.service.DeleteBulk(ctx, ids)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete feedback bulk: %w", err))
	}
	return connect.NewResponse(&lpbsv1.DeleteFeedbackBulkResponse{Deleted: deleted}), nil
}

func feedbackID(value int64) (int, error) {
	if value <= 0 || value > math.MaxInt {
		return 0, connect.NewError(connect.CodeInvalidArgument, errors.New("feedback ID must be positive"))
	}
	return int(value), nil
}

func feedbackStatusFromProto(value *lpbsv1.FeedbackStatus) (string, error) {
	if value == nil || *value == lpbsv1.FeedbackStatus_FEEDBACK_STATUS_UNSPECIFIED {
		return "", nil
	}
	switch *value {
	case lpbsv1.FeedbackStatus_FEEDBACK_STATUS_PENDING:
		return "pending", nil
	case lpbsv1.FeedbackStatus_FEEDBACK_STATUS_IN_PROGRESS:
		return "in_progress", nil
	case lpbsv1.FeedbackStatus_FEEDBACK_STATUS_RESOLVED:
		return "resolved", nil
	case lpbsv1.FeedbackStatus_FEEDBACK_STATUS_REJECTED:
		return "rejected", nil
	default:
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("invalid feedback status"))
	}
}

func feedbackRecordProto(value *metrics.FeedbackRequest) *lpbsv1.FeedbackRecord {
	if value == nil {
		return &lpbsv1.FeedbackRecord{}
	}
	result := &lpbsv1.FeedbackRecord{Id: int64(value.ID), Type: feedbackTypeProto(value.Type), Email: value.Email, Subject: value.Subject, Message: value.Message, Status: feedbackStatusProto(value.Status), CreatedAt: timestamppb.New(value.CreatedAt), UpdatedAt: timestamppb.New(value.UpdatedAt)}
	if value.OrderID != nil {
		result.OrderId = value.OrderID
	}
	return result
}

func feedbackTypeProto(value string) lpbsv1.FeedbackType {
	switch value {
	case "refund":
		return lpbsv1.FeedbackType_FEEDBACK_TYPE_REFUND
	case "bug":
		return lpbsv1.FeedbackType_FEEDBACK_TYPE_BUG
	case "feature":
		return lpbsv1.FeedbackType_FEEDBACK_TYPE_FEATURE
	default:
		return lpbsv1.FeedbackType_FEEDBACK_TYPE_GENERAL
	}
}

func validType(value string) bool {
	return value == "refund" || value == "bug" || value == "feature" || value == "general"
}

func feedbackStatusProto(value string) lpbsv1.FeedbackStatus {
	switch value {
	case "pending":
		return lpbsv1.FeedbackStatus_FEEDBACK_STATUS_PENDING
	case "in_progress":
		return lpbsv1.FeedbackStatus_FEEDBACK_STATUS_IN_PROGRESS
	case "resolved":
		return lpbsv1.FeedbackStatus_FEEDBACK_STATUS_RESOLVED
	case "rejected":
		return lpbsv1.FeedbackStatus_FEEDBACK_STATUS_REJECTED
	default:
		return lpbsv1.FeedbackStatus_FEEDBACK_STATUS_UNSPECIFIED
	}
}

func RegisterConnectRoutes(router *mux.Router, service metrics.FeedbackServicer, notifier Notifier, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	_, generated := lpbsconnect.NewFeedbackServiceHandler(NewConnectHandler(service, notifier))
	for _, procedure := range []string{lpbsconnect.FeedbackServiceCreateFeedbackProcedure} {
		router.Handle(procedure, generated).Methods(http.MethodPost)
	}
	for _, procedure := range []string{lpbsconnect.FeedbackServiceListFeedbackProcedure, lpbsconnect.FeedbackServiceGetFeedbackProcedure, lpbsconnect.FeedbackServiceUpdateFeedbackStatusProcedure, lpbsconnect.FeedbackServiceDeleteFeedbackProcedure, lpbsconnect.FeedbackServiceDeleteFeedbackBulkProcedure} {
		router.Handle(procedure, requireAdmin(generated.ServeHTTP)).Methods(http.MethodPost)
	}
}

var _ lpbsconnect.FeedbackServiceHandler = (*ConnectHandler)(nil)
