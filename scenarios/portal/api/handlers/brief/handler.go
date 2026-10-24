package brief

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"connectrpc.com/connect"
	briefv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/brief"
	"portal/internal/brief"
)

type Handler struct{ service *brief.Service }

func NewHandler(service *brief.Service) *Handler { return &Handler{service: service} }

func (h *Handler) Build(ctx context.Context, req *connect.Request[briefv1.BuildBriefRequest]) (*connect.Response[briefv1.BuildBriefResponse], error) {
	if h.service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("brief service is not configured"))
	}
	record, err := h.service.Build(ctx, brief.BuildInput{Prompt: req.Msg.GetPrompt(), Consumer: brief.ConsumerFromProto(req.Msg.GetConsumer()), ChatID: req.Msg.GetChatId(), MessageID: req.Msg.GetMessageId(), Harness: req.Msg.GetHarness().String(), SessionRef: req.Msg.GetSessionRef(), BudgetMS: int(req.Msg.GetBudgetMs())})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&briefv1.BuildBriefResponse{Brief: brief.ToProto(record)}), nil
}

func (h *Handler) Get(ctx context.Context, req *connect.Request[briefv1.GetBriefRequest]) (*connect.Response[briefv1.GetBriefResponse], error) {
	record, err := h.service.Get(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&briefv1.GetBriefResponse{Brief: brief.ToProto(record)}), nil
}
func (h *Handler) List(ctx context.Context, req *connect.Request[briefv1.ListBriefsRequest]) (*connect.Response[briefv1.ListBriefsResponse], error) {
	records, err := h.service.List(ctx, brief.ListInput{Consumer: brief.ConsumerFilterFromProto(req.Msg.GetConsumer()), ChatID: req.Msg.GetChatId(), SessionRef: req.Msg.GetSessionRef(), Limit: int(req.Msg.GetLimit())})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &briefv1.ListBriefsResponse{Briefs: make([]*briefv1.Brief, 0, len(records))}
	for _, record := range records {
		response.Briefs = append(response.Briefs, brief.ToProto(record))
	}
	return connect.NewResponse(response), nil
}
func (h *Handler) RecordUse(ctx context.Context, req *connect.Request[briefv1.RecordBriefUseRequest]) (*connect.Response[briefv1.RecordBriefUseResponse], error) {
	if strings.TrimSpace(req.Msg.GetBriefId()) == "" || (req.Msg.GetItemIndex() < 0 && req.Msg.GetKind() != briefv1.BriefUseKind_BRIEF_USE_KIND_OPENED) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("brief id and valid item index are required"))
	}
	recorded, err := h.service.RecordUse(ctx, brief.UseInput{BriefID: req.Msg.GetBriefId(), ItemIndex: int(req.Msg.GetItemIndex()), Kind: briefUseName(req.Msg.GetKind())})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&briefv1.RecordBriefUseResponse{Recorded: recorded}), nil
}

func (h *Handler) Stats(ctx context.Context, req *connect.Request[briefv1.BriefStatsRequest]) (*connect.Response[briefv1.BriefStatsResponse], error) {
	rows, err := h.service.Stats(ctx, brief.StatsInput{WindowDays: int(req.Msg.GetWindowDays()), Consumer: brief.ConsumerFilterFromProto(req.Msg.GetConsumer())})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&briefv1.BriefStatsResponse{Rows: brief.StatsToProto(rows)}), nil
}
func briefUseName(value briefv1.BriefUseKind) string {
	switch value {
	case briefv1.BriefUseKind_BRIEF_USE_KIND_OPENED:
		return "OPENED"
	case briefv1.BriefUseKind_BRIEF_USE_KIND_COPIED:
		return "COPIED"
	case briefv1.BriefUseKind_BRIEF_USE_KIND_REFERENCED:
		return "REFERENCED"
	case briefv1.BriefUseKind_BRIEF_USE_KIND_REJECTED:
		return "REJECTED"
	default:
		return "UNSPECIFIED"
	}
}
