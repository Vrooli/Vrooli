package enrichment

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	internal "document-manager/internal/enrichment"
	enrichmentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/enrichment"
)

type connectHandler struct{ service internal.Service }

func NewConnectHandler(service internal.Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) Enrich(ctx context.Context, req *connect.Request[enrichmentv1.EnrichRequest]) (*connect.Response[enrichmentv1.EnrichResponse], error) {
	record, err := h.service.Enrich(ctx, req.Msg.DocumentHash, req.Msg.Text, req.Msg.PrivacyClass)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&enrichmentv1.EnrichResponse{Enriched: record.Status == "enriched", Summary: record.Summary, SuggestedPrivacyClass: record.SuggestedPrivacyClass, Status: record.Status}), nil
}

func (h *connectHandler) Embed(ctx context.Context, req *connect.Request[enrichmentv1.EmbedRequest]) (*connect.Response[enrichmentv1.EmbedResponse], error) {
	embedding, err := h.service.Embed(ctx, req.Msg.DocumentHash, req.Msg.UnitId, req.Msg.Text, req.Msg.PrivacyClass, req.Msg.Role)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	if embedding.Dimension < 0 || embedding.Dimension > 2_147_483_647 {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("embedding dimension exceeds protobuf limit: %d", embedding.Dimension))
	}
	return connect.NewResponse(&enrichmentv1.EmbedResponse{EmbeddingId: embedding.ID, Enriched: true, Dimension: int32(embedding.Dimension) /* #nosec G115 -- explicit protobuf int32 bounds check above */, Status: "enriched"}), nil
}
