package retrieval

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	internal "signal-inbox/internal/retrieval"
	"signal-inbox/internal/signals"

	retrievalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/retrieval"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/shared"
)

type connectHandler struct{ service *internal.Service }

func NewConnectHandler(service *internal.Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) Search(ctx context.Context, req *connect.Request[retrievalv1.SearchRequest]) (*connect.Response[retrievalv1.SearchResponse], error) {
	filter, err := filterFromProto(req.Msg.Filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	page, err := h.service.SearchPage(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&retrievalv1.SearchResponse{Results: resultsToProto(page.Results), NextPageAfter: page.NextPageAfter}), nil
}

func (h *connectHandler) Ambient(ctx context.Context, req *connect.Request[retrievalv1.AmbientRequest]) (*connect.Response[retrievalv1.AmbientResponse], error) {
	results, err := h.service.Ambient(ctx, req.Msg.CategoryId, int(req.Msg.Budget))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&retrievalv1.AmbientResponse{Results: resultsToProto(results)}), nil
}

func filterFromProto(filter *retrievalv1.SearchFilter) (internal.Filter, error) {
	if filter == nil {
		return internal.Filter{}, nil
	}
	result := internal.Filter{Text: filter.Text, CategoryID: filter.CategoryId, Disposition: filter.Disposition, Limit: int(filter.PageSize), Tags: filter.Tags, PageAfter: filter.PageAfter}
	switch filter.SourceKind {
	case "", "url", "text", "image":
		result.SourceKind = signals.SourceKind(filter.SourceKind)
	default:
		return internal.Filter{}, errors.New("source_kind must be url, text, or image")
	}
	if filter.CapturedAfter != nil {
		if err := filter.CapturedAfter.CheckValid(); err != nil {
			return internal.Filter{}, err
		}
		value := filter.CapturedAfter.AsTime().UTC()
		result.CapturedAfter = &value
	}
	if filter.CapturedBefore != nil {
		if err := filter.CapturedBefore.CheckValid(); err != nil {
			return internal.Filter{}, err
		}
		value := filter.CapturedBefore.AsTime().UTC()
		result.CapturedBefore = &value
	}
	return result, nil
}

func resultsToProto(results []internal.Result) []*retrievalv1.RetrievedSignal {
	response := make([]*retrievalv1.RetrievedSignal, 0, len(results))
	for _, result := range results {
		response = append(response, &retrievalv1.RetrievedSignal{Signal: signalToProto(result.Signal), CategoryId: result.CategoryID, Disposition: result.Disposition, Score: result.Score})
	}
	return response
}

func signalToProto(signal signals.Signal) *sharedv1.Signal {
	kind := sharedv1.SourceKind_SOURCE_KIND_UNSPECIFIED
	switch signal.SourceKind {
	case signals.SourceKindURL:
		kind = sharedv1.SourceKind_SOURCE_KIND_URL
	case signals.SourceKindText:
		kind = sharedv1.SourceKind_SOURCE_KIND_TEXT
	case signals.SourceKindImage:
		kind = sharedv1.SourceKind_SOURCE_KIND_IMAGE
	}
	return &sharedv1.Signal{Id: signal.ID, SourceKind: kind, SourceIdentity: signal.SourceIdentity, SourceUrl: signal.SourceURL, CapturedAt: timestamppb.New(signal.CapturedAt), RawPayloadRef: signal.RawPayloadRef, ExtractedContent: signal.ExtractedContent, ContentHash: signal.ContentHash, NeedsAttention: signal.NeedsAttention, CaptureNote: signal.CaptureNote, Tags: signal.Tags}
}
