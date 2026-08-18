package corpus

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	internal "document-manager/internal/corpus"
	corpusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/corpus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type connectHandler struct{ service internal.Service }

func NewConnectHandler(service internal.Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) CreateCollection(ctx context.Context, req *connect.Request[corpusv1.CreateCollectionRequest]) (*connect.Response[corpusv1.CreateCollectionResponse], error) {
	c, err := h.service.CreateCollection(ctx, req.Msg.Name, req.Msg.DefaultPrivacyClass, req.Msg.Federated)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&corpusv1.CreateCollectionResponse{Collection: collectionProto(c)}), nil
}

func (h *connectHandler) GetCollection(ctx context.Context, req *connect.Request[corpusv1.GetCollectionRequest]) (*connect.Response[corpusv1.GetCollectionResponse], error) {
	c, err := h.service.GetCollection(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&corpusv1.GetCollectionResponse{Collection: collectionProto(c)}), nil
}

func (h *connectHandler) ListCollections(ctx context.Context, req *connect.Request[corpusv1.ListCollectionsRequest]) (*connect.Response[corpusv1.ListCollectionsResponse], error) {
	collections, err := h.service.ListCollections(ctx, int(req.Msg.Limit))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &corpusv1.ListCollectionsResponse{Collections: make([]*corpusv1.Collection, 0, len(collections))}
	for _, c := range collections {
		out.Collections = append(out.Collections, collectionProto(c))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) AddDocument(ctx context.Context, req *connect.Request[corpusv1.AddDocumentRequest]) (*connect.Response[corpusv1.AddDocumentResponse], error) {
	d, err := h.service.AddDocument(ctx, req.Msg.CollectionId, req.Msg.DocumentHash, req.Msg.PrivacyClass)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&corpusv1.AddDocumentResponse{Document: documentProto(d)}), nil
}

func (h *connectHandler) ListDocuments(ctx context.Context, req *connect.Request[corpusv1.ListDocumentsRequest]) (*connect.Response[corpusv1.ListDocumentsResponse], error) {
	documents, err := h.service.ListDocuments(ctx, req.Msg.CollectionId, int(req.Msg.Limit))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &corpusv1.ListDocumentsResponse{Documents: make([]*corpusv1.CollectionDocument, 0, len(documents))}
	for _, d := range documents {
		out.Documents = append(out.Documents, documentProto(d))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) Export(ctx context.Context, req *connect.Request[corpusv1.ExportRequest]) (*connect.Response[corpusv1.ExportResponse], error) {
	archive, err := h.service.Export(ctx, req.Msg.CollectionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&corpusv1.ExportResponse{ArchiveJson: archive, Format: "vrooli-document-corpus+json;version=1"}), nil
}

func (h *connectHandler) Import(ctx context.Context, req *connect.Request[corpusv1.ImportRequest]) (*connect.Response[corpusv1.ImportResponse], error) {
	c, count, err := h.service.Import(ctx, req.Msg.ArchiveJson)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if count < 0 || count > 2_147_483_647 {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("imported document count exceeds protobuf limit: %d", count))
	}
	return connect.NewResponse(&corpusv1.ImportResponse{Collection: collectionProto(c), DocumentsImported: int32(count) /* #nosec G115 -- explicit protobuf int32 bounds check above */}), nil
}

func (h *connectHandler) Prune(ctx context.Context, req *connect.Request[corpusv1.PruneRequest]) (*connect.Response[corpusv1.PruneResponse], error) {
	dryRun, removed, bytes := h.service.Prune(ctx, req.Msg.DryRun)
	return connect.NewResponse(&corpusv1.PruneResponse{DryRun: dryRun, RemovedKinds: removed, ReclaimedBytes: bytes}), nil
}

func collectionProto(c internal.Collection) *corpusv1.Collection {
	return &corpusv1.Collection{Id: c.ID, Name: c.Name, DefaultPrivacyClass: c.DefaultPrivacyClass, Federated: c.Federated, CreatedAt: timestamppb.New(c.CreatedAt)}
}

func documentProto(d internal.Membership) *corpusv1.CollectionDocument {
	return &corpusv1.CollectionDocument{CollectionId: d.CollectionID, DocumentHash: d.DocumentHash, PrivacyClass: d.PrivacyClass, CreatedAt: timestamppb.New(d.CreatedAt)}
}
