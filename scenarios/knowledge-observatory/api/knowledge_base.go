package main

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	pkg "github.com/vrooli/ai-go/search"
	"github.com/vrooli/api-core/connectx"
	kov1 "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1"
	koconnect "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1/knowledgeobservatoryv1connect"
	"knowledge-observatory/internal/aisearch"
	"knowledge-observatory/internal/knowledgebase"
)

type knowledgeBaseHandler struct {
	server  *Server
	sources *knowledgebase.Service
}

func (s *Server) registerKnowledgeBase() {
	root := resolveScenariosRoot()
	if root == "" {
		return
	}
	handler := &knowledgeBaseHandler{server: s, sources: &knowledgebase.Service{RepoRoot: filepath.Dir(root)}}
	p, h := koconnect.NewKnowledgeBaseServiceHandler(handler)
	connectx.RegisterServices(s.router, connectx.ServiceMount{Path: p, Handler: h})
}

func sourceError(err error) error {
	code := connect.CodeInternal
	switch {
	case errors.Is(err, knowledgebase.ErrInvalid):
		code = connect.CodeInvalidArgument
	case errors.Is(err, knowledgebase.ErrRevision):
		code = connect.CodeFailedPrecondition
	case errors.Is(err, fs.ErrNotExist):
		code = connect.CodeNotFound
	case errors.Is(err, fs.ErrPermission):
		code = connect.CodePermissionDenied
	case errors.Is(err, context.Canceled):
		code = connect.CodeCanceled
	}
	return connect.NewError(code, err)
}

func (h *knowledgeBaseHandler) InspectDocument(ctx context.Context, req *connect.Request[kov1.InspectDocumentRequest]) (*connect.Response[kov1.InspectDocumentResponse], error) {
	out, err := h.sources.Inspect(ctx, req.Msg)
	if err != nil {
		return nil, sourceError(err)
	}
	return connect.NewResponse(out), nil
}

func (h *knowledgeBaseHandler) ReviewDocuments(ctx context.Context, req *connect.Request[kov1.ReviewDocumentsRequest]) (*connect.Response[kov1.ReviewDocumentsResponse], error) {
	out, err := h.sources.Review(ctx, req.Msg)
	if err != nil {
		return nil, sourceError(err)
	}
	return connect.NewResponse(out), nil
}

func (h *knowledgeBaseHandler) SearchDocuments(ctx context.Context, req *connect.Request[kov1.SearchDocumentsRequest]) (*connect.Response[kov1.SearchDocumentsResponse], error) {
	in := req.Msg
	if strings.TrimSpace(in.Query) == "" || len(in.Query) > 2000 || in.Limit < 0 || in.Limit > 20 {
		return nil, sourceError(knowledgebase.ErrInvalid)
	}
	switch in.Scope {
	case "", "global":
		if in.Target != "" {
			return nil, sourceError(knowledgebase.ErrInvalid)
		}
	case "scenario":
		if in.Target == "" || strings.ContainsAny(in.Target, "/\\:") || in.Target == ".." {
			return nil, sourceError(knowledgebase.ErrInvalid)
		}
	case "path":
		if _, err := knowledgebase.PortablePath(in.Target); err != nil {
			return nil, sourceError(err)
		}
	default:
		return nil, sourceError(knowledgebase.ErrInvalid)
	}
	switch in.Mode {
	case "", "auto", "hybrid", "dense", "text":
	default:
		return nil, sourceError(knowledgebase.ErrInvalid)
	}
	q := pkg.SearchQuery{Query: in.Query, Scope: parseScope(in.Scope, in.Target), Mode: parseSearchMode(in.Mode), Limit: int(in.Limit)}
	if q.Limit == 0 {
		q.Limit = 5
	}
	var resp pkg.SearchResponse
	var err error
	if h.server.docSearch != nil {
		resp, err = h.server.docSearch.Search(ctx, q)
	} else if h.server.docSearchService != nil && (in.Mode == "" || in.Mode == "auto" || in.Mode == "text") {
		resp.Results, err = aisearch.NewDocsearchFallback(h.server.docSearchService)(ctx, q)
		resp.Method = "text"
	} else {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("documentation search unavailable"))
	}
	if err != nil {
		return nil, sourceError(err)
	}
	out := &kov1.SearchDocumentsResponse{Method: resp.Method, Reranker: resp.Reranker}
	for _, hit := range h.server.currentDocHits(resp.Results) {
		out.Results = append(out.Results, &kov1.DocumentHit{Path: hit.RelativePath, Title: hit.Title, Snippet: hit.Snippet, Score: hit.Score, Metadata: knowledgebase.Metadata(hit.Metadata)})
	}
	return connect.NewResponse(out), nil
}

func (h *knowledgeBaseHandler) KnowledgeStatus(ctx context.Context, _ *connect.Request[kov1.KnowledgeStatusRequest]) (*connect.Response[kov1.KnowledgeStatusResponse], error) {
	out := &kov1.KnowledgeStatusResponse{}
	if h.server.docSearch != nil {
		st := h.server.docSearch.Status(ctx)
		out.Available = st.Available
		out.IndexedCount = int32(st.IndexedCount)
		out.LastReconcileAt = st.LastReconcileAt
		out.LastReconcileOutcome = st.LastReconcileOutcome
		out.Reranker = st.Reranker
	}
	return connect.NewResponse(out), nil
}
