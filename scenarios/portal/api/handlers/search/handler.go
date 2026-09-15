package search

import (
	"context"

	"connectrpc.com/connect"

	internalchat "portal/internal/chat"
	internalsearch "portal/internal/search"

	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/search"
)

type Handler struct {
	service *internalsearch.Service
}

func NewHandler(service *internalsearch.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Suggest(ctx context.Context, req *connect.Request[searchv1.SuggestRequest]) (*connect.Response[searchv1.SuggestResponse], error) {
	if h.service == nil {
		return connect.NewResponse(&searchv1.SuggestResponse{
			Degraded: true,
			Reason:   "search service is not configured",
		}), nil
	}
	result, err := h.service.Suggest(ctx, internalsearch.QueryInput{
		Query: req.Msg.GetQuery(),
		Types: req.Msg.GetTypes(),
		Limit: req.Msg.GetLimit(),
		Group: req.Msg.GetGroup(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&searchv1.SuggestResponse{
		Hits:      internalchat.ToProtoSearchHits(result.Hits),
		Degraded:  result.Degraded,
		Reason:    result.Reason,
		LatencyMs: result.LatencyMS,
	}), nil
}
