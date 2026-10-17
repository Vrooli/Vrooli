// Package file_preview is the HTTP-handler home for the file-preview domain.
// It exposes the generated Connect-RPC FilePreviewService (proto schema:
// packages/proto/schemas/web-console/v1/file_preview) for resolve + bounded
// text content. The opaque-id blob/range route is a sanctioned REST exception;
// its handler lives in api/file_preview_handlers.go and its descriptor in
// endpoints.go.
package file_preview

import (
	"context"
	"log"

	"web-console/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	filepreviewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/file_preview/file_preview_v1connect"
)

// Service is the seam the Connect handler depends on. The concrete
// implementation lives in package main (adapts the session store, the
// filepreview resolver, and the preview-id store to satisfy this interface).
type Service interface {
	Resolve(ctx context.Context, in ResolveInput) (ResolveResult, error)
	GetTextContent(ctx context.Context, sessionID, previewID string) (TextResult, error)
	ListDirectory(ctx context.Context, in ListInput) (ListResult, error)
}

// ResolveInput carries a resolve request.
type ResolveInput struct {
	SessionID     string
	Path          string
	SourceContext string // "message_link" | "inline_code" | "cli" | ""
}

// ResolveResult is the transport-neutral resolve result. Kind is the string
// form of filepreview.Kind; the Connect handler maps it onto the proto enum.
type ResolveResult struct {
	PreviewID            string
	InputPath            string
	ResolvedPath         string
	Basename             string
	Line                 int
	HasLine              bool
	ResolutionBasis      string
	Kind                 string
	MIMEType             string
	SizeBytes            int64
	ModTimeUnixNano      int64
	CanPreview           bool
	CanDownload          bool
	SupportsRange        bool
	TextContentAvailable bool
	ListingAvailable     bool
	BlobURL              string
	ExpiresUnixNano      int64
	Warnings             []string
}

// TextResult is the bounded text-content result.
type TextResult struct {
	ResolvedPath string
	Kind         string
	MIMEType     string
	Content      string
	Truncated    bool
	Line         int
	HasLine      bool
}

// ListInput carries a directory-listing request. Sort is the string form of
// filepreview.Sort; the Connect handler maps the proto enum onto it.
type ListInput struct {
	SessionID  string
	PreviewID  string
	Sort       string
	ShowHidden bool
	PageSize   int
	PageToken  string
}

// ListEntry is the transport-neutral form of one directory child. Kind is the
// string form of filepreview.Kind and is empty when the entry's kind is only
// determined on open.
type ListEntry struct {
	Name            string
	EntryType       string
	Kind            string
	SizeBytes       int64
	ModTimeUnixNano int64
	CanPreview      bool
	SymlinkTarget   string
	SymlinkBroken   bool
	Mode            string
	ChildCount      int64
}

// ListResult is one bounded page of a directory.
type ListResult struct {
	ResolvedPath  string
	ParentPath    string
	Entries       []ListEntry
	TotalEntries  int
	Truncated     bool
	NextPageToken string
	EffectiveSort string
	Warnings      []string
}

// Module wires the file-preview Connect service into the API server. The blob
// REST route is mounted separately by package main (it needs the Server's
// session lookup + preview store), so this Module only mounts the Connect
// handler. Endpoints still describes all three routes for gen-endpoints.
func Module(svc Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := filepreviewconnect.NewFilePreviewServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "file_preview",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
