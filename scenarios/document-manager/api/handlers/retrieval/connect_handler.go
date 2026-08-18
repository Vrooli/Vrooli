package retrieval

import (
	"context"

	"connectrpc.com/connect"
	internal "document-manager/internal/retrieval"
	retrievalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/retrieval"
)

type connectHandler struct{ service internal.Service }

func NewConnectHandler(service internal.Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) Query(ctx context.Context, req *connect.Request[retrievalv1.QueryRequest]) (*connect.Response[retrievalv1.QueryResponse], error) {
	response, err := executeQuery(ctx, h.service, internal.Query{Text: req.Msg.Text, CollectionID: req.Msg.CollectionId, CallerMaxPrivacy: req.Msg.CallerMaxPrivacy, Federated: req.Msg.Federated, Limit: int(req.Msg.Limit)})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&retrievalv1.QueryResponse{Results: resultProto(response.Results), Partial: response.Partial}), nil
}

func (h *connectHandler) Similar(ctx context.Context, req *connect.Request[retrievalv1.SimilarRequest]) (*connect.Response[retrievalv1.SimilarResponse], error) {
	response, err := executeQuery(ctx, h.service, internal.Query{Text: req.Msg.DocumentHash, CallerMaxPrivacy: req.Msg.CallerMaxPrivacy, Limit: int(req.Msg.Limit)})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&retrievalv1.SimilarResponse{Results: resultProto(response.Results), Partial: response.Partial}), nil
}

func executeQuery(ctx context.Context, service internal.Service, query internal.Query) (internal.Response, error) {
	return service.Query(ctx, query)
}

func resultProto(results []internal.Result) []*retrievalv1.QueryResult {
	out := make([]*retrievalv1.QueryResult, 0, len(results))
	for _, r := range results {
		out = append(out, &retrievalv1.QueryResult{UnitId: r.UnitID, DocumentHash: r.DocumentHash, AnchorUri: r.AnchorURI, Score: r.Score})
	}
	return out
}
