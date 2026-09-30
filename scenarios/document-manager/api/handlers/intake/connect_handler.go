package intake

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"document-manager/internal/corpus"
	internal "document-manager/internal/intake"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	intakev1 "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/intake"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

type connectHandler struct {
	service      internal.Service
	watchPath    string
	watchEnabled bool
	corpus       *corpus.Service
}

func NewConnectHandler(service internal.Service, corpusServices ...corpus.Service) *connectHandler {
	h := &connectHandler{service: service}
	if len(corpusServices) > 0 {
		h.corpus = &corpusServices[0]
	}
	return h
}

func (h *connectHandler) Ingest(ctx context.Context, req *connect.Request[intakev1.IngestRequest]) (*connect.Response[intakev1.IngestResponse], error) {
	d, duplicate, err := h.service.Ingest(internal.IngestInput{Content: req.Msg.Content, SourceName: req.Msg.SourceName, PrivacyClass: privacyName(req.Msg.PrivacyClass)})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if req.Msg.CollectionId != "" && h.corpus != nil {
		if _, err := h.corpus.AddDocument(ctx, req.Msg.CollectionId, d.ContentSHA256, req.Msg.PrivacyClass); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("collection membership: %w", err))
		}
	}
	return connect.NewResponse(&intakev1.IngestResponse{Document: toProto(d), Duplicate: duplicate}), nil
}

func (h *connectHandler) GetDocument(ctx context.Context, req *connect.Request[intakev1.GetDocumentRequest]) (*connect.Response[intakev1.GetDocumentResponse], error) {
	d, err := h.service.Get(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&intakev1.GetDocumentResponse{Document: toProto(d)}), nil
}

func (h *connectHandler) ListDocuments(ctx context.Context, req *connect.Request[intakev1.ListDocumentsRequest]) (*connect.Response[intakev1.ListDocumentsResponse], error) {
	docs, err := h.service.List(int(req.Msg.Limit))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &intakev1.ListDocumentsResponse{Documents: make([]*intakev1.Document, 0, len(docs))}
	for _, d := range docs {
		out.Documents = append(out.Documents, toProto(d))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ListSources(ctx context.Context, req *connect.Request[intakev1.ListSourcesRequest]) (*connect.Response[intakev1.ListSourcesResponse], error) {
	sources, err := h.service.Sources()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&intakev1.ListSourcesResponse{Sources: sources}), nil
}

func (h *connectHandler) ConfigureWatch(ctx context.Context, req *connect.Request[intakev1.ConfigureWatchRequest]) (*connect.Response[intakev1.ConfigureWatchResponse], error) {
	h.watchPath = req.Msg.Path
	h.watchEnabled = req.Msg.Enabled
	if h.watchEnabled {
		entries, err := os.ReadDir(h.watchPath)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			content, err := os.ReadFile(filepath.Join(h.watchPath, entry.Name()))
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}
			if _, _, err := h.service.Ingest(internal.IngestInput{Content: content, SourceName: entry.Name()}); err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		}
	}
	return connect.NewResponse(&intakev1.ConfigureWatchResponse{Path: h.watchPath, Enabled: h.watchEnabled}), nil
}

func (h *connectHandler) GetTypeVerdict(ctx context.Context, req *connect.Request[intakev1.GetTypeVerdictRequest]) (*connect.Response[intakev1.GetTypeVerdictResponse], error) {
	d, err := h.service.Get(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&intakev1.GetTypeVerdictResponse{DetectedMime: d.DetectedMIME, PdfType: d.PDFType, Confidence: d.PDFConfidence}), nil
}

func toProto(d internal.Document) *intakev1.Document {
	return &intakev1.Document{Id: d.ID, ContentSha256: d.ContentSHA256, SourceName: d.SourceName, DetectedMime: d.DetectedMIME, PdfType: d.PDFType, PdfConfidence: d.PDFConfidence, PrivacyClass: privacyEnum(d.PrivacyClass), CreatedAt: timestamppb.New(d.CreatedAt)}
}

func privacyName(p sharedv1.PrivacyClass) string {
	switch p {
	case sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC:
		return "public"
	case sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL:
		return "confidential"
	case sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET:
		return "secret"
	default:
		return "internal"
	}
}

func privacyEnum(raw string) sharedv1.PrivacyClass {
	switch raw {
	case "public":
		return sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC
	case "confidential":
		return sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL
	case "secret":
		return sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET
	default:
		return sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL
	}
}
