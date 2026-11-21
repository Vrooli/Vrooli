package debt

import (
	"context"
	"errors"
	"log"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"

	"connectrpc.com/connect"
	debtv1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/debt"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Deps struct {
	Repository catalog.Repository
	Logger     *log.Logger
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListDebt(ctx context.Context, req *connect.Request[debtv1.ListDebtRequest]) (*connect.Response[debtv1.ListDebtResponse], error) {
	entries, err := h.deps.Repository.ListDebt(ctx, req.Msg.TemplateId, req.Msg.Status)
	if err != nil {
		h.deps.Logger.Printf("debt.ListDebt: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("list debt"))
	}
	resp := &debtv1.ListDebtResponse{Entries: make([]*debtv1.DebtEntry, 0, len(entries))}
	for _, entry := range entries {
		resp.Entries = append(resp.Entries, debtToProto(entry))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetDebt(ctx context.Context, req *connect.Request[debtv1.GetDebtRequest]) (*connect.Response[debtv1.GetDebtResponse], error) {
	entry, err := h.deps.Repository.GetDebt(ctx, req.Msg.Key)
	if err != nil {
		var notFound catalog.ErrNotFound
		if errors.As(err, &notFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		h.deps.Logger.Printf("debt.GetDebt(%q): %v", req.Msg.Key, err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("get debt"))
	}
	return connect.NewResponse(&debtv1.GetDebtResponse{Entry: debtToProto(entry)}), nil
}

func debtToProto(entry catalog.DebtEntry) *debtv1.DebtEntry {
	return &debtv1.DebtEntry{
		Key:         entry.Key,
		TemplateId:  entry.TemplateID,
		Source:      entry.Source,
		Severity:    entry.Severity,
		Status:      entry.Status,
		Title:       entry.Title,
		Detail:      entry.Detail,
		FirstSeenAt: timestamppb.New(entry.FirstSeenAt),
		LastSeenAt:  timestamppb.New(entry.LastSeenAt),
	}
}
