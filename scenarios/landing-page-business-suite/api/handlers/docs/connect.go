package docs

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
)

// ConnectDependencies supplies the root-owned filesystem location and logging
// policy to the Docs Connect transport.
type ConnectDependencies struct {
	DocsRoot func() string
	Log      func(string, map[string]any)
}

// ConnectHandler implements the generated DocsService contract.
type ConnectHandler struct{ deps ConnectDependencies }

func NewConnectHandler(deps ConnectDependencies) *ConnectHandler { return &ConnectHandler{deps: deps} }

func (h *ConnectHandler) GetDocsTree(_ context.Context, _ *connect.Request[lpbsv1.GetDocsTreeRequest]) (*connect.Response[lpbsv1.GetDocsTreeResponse], error) {
	docsRoot := h.deps.DocsRoot()
	if _, err := os.Stat(docsRoot); os.IsNotExist(err) {
		h.log("docs_directory_not_found", map[string]any{"path": docsRoot, "error": err.Error()})
		return connect.NewResponse(&lpbsv1.GetDocsTreeResponse{}), nil
	}
	entries, err := BuildTree(docsRoot, "")
	if err != nil {
		h.log("docs_tree_build_failed", map[string]any{"path": docsRoot, "error": err.Error()})
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("read docs directory: %w", err))
	}
	h.log("docs_tree_success", map[string]any{"path": docsRoot, "entry_count": len(entries)})
	return connect.NewResponse(&lpbsv1.GetDocsTreeResponse{Entries: entriesProto(entries)}), nil
}

func (h *ConnectHandler) GetDocContent(_ context.Context, request *connect.Request[lpbsv1.GetDocContentRequest]) (*connect.Response[lpbsv1.GetDocContentResponse], error) {
	path, err := safeDocumentPath(h.deps.DocsRoot(), request.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	// #nosec G304 -- safeDocumentPath validates the resolved path against docsRoot.
	body, err := os.ReadFile(path.full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("file not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("read document: %w", err))
	}
	return connect.NewResponse(&lpbsv1.GetDocContentResponse{Path: path.clean, Content: string(body), Title: ExtractTitle(string(body), path.clean)}), nil
}

type documentPath struct{ clean, full string }

func safeDocumentPath(docsRoot, requested string) (documentPath, error) {
	clean := filepath.Clean(requested)
	if requested == "" || strings.Contains(clean, "..") || !strings.HasSuffix(strings.ToLower(clean), ".md") {
		return documentPath{}, errors.New("path must name a markdown file within the docs root")
	}
	full, err := filepath.Abs(filepath.Join(docsRoot, clean))
	if err != nil {
		return documentPath{}, errors.New("invalid document path")
	}
	root, err := filepath.Abs(docsRoot)
	if err != nil || !IsWithinDirectory(root, full) {
		return documentPath{}, errors.New("path must name a markdown file within the docs root")
	}
	return documentPath{clean: clean, full: full}, nil
}

func entriesProto(entries []Entry) []*lpbsv1.DocEntry {
	result := make([]*lpbsv1.DocEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, &lpbsv1.DocEntry{Name: entry.Name, Path: entry.Path, IsDir: entry.IsDir, Children: entriesProto(entry.Children)})
	}
	return result
}

func (h *ConnectHandler) log(event string, fields map[string]any) {
	if h.deps.Log != nil {
		h.deps.Log(event, fields)
	}
}

// RegisterConnectRoutes mounts the generated DocsService procedures behind
// the admin authorization policy supplied by API composition.
func RegisterConnectRoutes(router *mux.Router, deps ConnectDependencies, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	_, service := lpbsconnect.NewDocsServiceHandler(NewConnectHandler(deps))
	handler := requireAdmin(http.HandlerFunc(service.ServeHTTP))
	router.Handle(lpbsconnect.DocsServiceGetDocsTreeProcedure, handler).Methods(http.MethodPost)
	router.Handle(lpbsconnect.DocsServiceGetDocContentProcedure, handler).Methods(http.MethodPost)
}
