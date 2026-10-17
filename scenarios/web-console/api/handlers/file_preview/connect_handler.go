package file_preview

import (
	"context"
	"errors"
	"log"
	"math"
	"strings"

	"connectrpc.com/connect"

	filepreviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/file_preview"
)

// Deps wires the seams the Connect file-preview handler needs.
type Deps struct {
	Service Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC handler implementing
// FilePreviewServiceHandler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// Sentinel errors the Service implementation returns; the handler maps them to
// Connect codes.
var (
	// ErrSessionNotFound — unknown session id. Mapped to CodeNotFound.
	ErrSessionNotFound = errors.New("session not found")
	// ErrNotFound — unknown/expired preview id or missing file. CodeNotFound.
	ErrNotFound = errors.New("not found")
	// ErrInvalidArgument — malformed/missing fields. CodeInvalidArgument.
	ErrInvalidArgument = errors.New("invalid argument")
	// ErrPermissionDenied — path not readable / not allowed. CodePermissionDenied.
	ErrPermissionDenied = errors.New("permission denied")
	// ErrPreviewUnavailable — non-previewable category or too large.
	// CodeFailedPrecondition.
	ErrPreviewUnavailable = errors.New("preview unavailable")
	// ErrStale — the target changed underneath a multi-step read (a directory
	// mutated between listing pages). CodeAborted, because retrying from the
	// start is the correct client response.
	ErrStale = errors.New("target changed")
)

func (h *connectHandler) Resolve(ctx context.Context, req *connect.Request[filepreviewv1.ResolveRequest]) (*connect.Response[filepreviewv1.ResolveResponse], error) {
	sessionID := strings.TrimSpace(req.Msg.GetSessionId())
	if sessionID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id is required"))
	}
	if strings.TrimSpace(req.Msg.GetPath()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("path is required"))
	}
	res, err := h.deps.Service.Resolve(ctx, ResolveInput{
		SessionID:     sessionID,
		Path:          req.Msg.GetPath(),
		SourceContext: sourceContextToString(req.Msg.GetSourceContext()),
	})
	if err != nil {
		return nil, h.classify(err, "file_preview.Resolve")
	}
	return connect.NewResponse(&filepreviewv1.ResolveResponse{
		PreviewId:            res.PreviewID,
		InputPath:            res.InputPath,
		ResolvedPath:         res.ResolvedPath,
		Basename:             res.Basename,
		Line:                 int32(res.Line),
		HasLine:              res.HasLine,
		ResolutionBasis:      res.ResolutionBasis,
		PreviewKind:          kindToProto(res.Kind),
		MimeType:             res.MIMEType,
		SizeBytes:            res.SizeBytes,
		MtimeUnixNano:        res.ModTimeUnixNano,
		CanPreview:           res.CanPreview,
		CanDownload:          res.CanDownload,
		SupportsRange:        res.SupportsRange,
		TextContentAvailable: res.TextContentAvailable,
		ListingAvailable:     res.ListingAvailable,
		BlobUrl:              res.BlobURL,
		ExpiresUnixNano:      res.ExpiresUnixNano,
		Warnings:             append([]string(nil), res.Warnings...),
	}), nil
}

func (h *connectHandler) GetTextContent(ctx context.Context, req *connect.Request[filepreviewv1.GetTextContentRequest]) (*connect.Response[filepreviewv1.GetTextContentResponse], error) {
	sessionID := strings.TrimSpace(req.Msg.GetSessionId())
	previewID := strings.TrimSpace(req.Msg.GetPreviewId())
	if sessionID == "" || previewID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id and preview_id are required"))
	}
	res, err := h.deps.Service.GetTextContent(ctx, sessionID, previewID)
	if err != nil {
		return nil, h.classify(err, "file_preview.GetTextContent")
	}
	return connect.NewResponse(&filepreviewv1.GetTextContentResponse{
		ResolvedPath: res.ResolvedPath,
		PreviewKind:  kindToProto(res.Kind),
		MimeType:     res.MIMEType,
		Content:      res.Content,
		Truncated:    res.Truncated,
		Line:         int32(res.Line),
		HasLine:      res.HasLine,
	}), nil
}

func (h *connectHandler) ListDirectory(ctx context.Context, req *connect.Request[filepreviewv1.ListDirectoryRequest]) (*connect.Response[filepreviewv1.ListDirectoryResponse], error) {
	sessionID := strings.TrimSpace(req.Msg.GetSessionId())
	previewID := strings.TrimSpace(req.Msg.GetPreviewId())
	if sessionID == "" || previewID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id and preview_id are required"))
	}
	res, err := h.deps.Service.ListDirectory(ctx, ListInput{
		SessionID:  sessionID,
		PreviewID:  previewID,
		Sort:       sortToString(req.Msg.GetSort()),
		ShowHidden: req.Msg.GetShowHidden(),
		PageSize:   int(req.Msg.GetPageSize()),
		PageToken:  req.Msg.GetPageToken(),
	})
	if err != nil {
		return nil, h.classify(err, "file_preview.ListDirectory")
	}

	entries := make([]*filepreviewv1.DirectoryEntry, 0, len(res.Entries))
	for _, e := range res.Entries {
		entries = append(entries, &filepreviewv1.DirectoryEntry{
			Name:          e.Name,
			EntryType:     entryTypeToProto(e.EntryType),
			PreviewKind:   entryKindToProto(e.Kind),
			SizeBytes:     e.SizeBytes,
			MtimeUnixNano: e.ModTimeUnixNano,
			CanPreview:    e.CanPreview,
			SymlinkTarget: e.SymlinkTarget,
			SymlinkBroken: e.SymlinkBroken,
			Mode:          e.Mode,
			ChildCount:    e.ChildCount,
		})
	}

	// The listing engine caps entries far below this, but narrowing to the
	// wire's int32 should be lossless by construction, not by assumption.
	totalEntries := res.TotalEntries
	if totalEntries > math.MaxInt32 {
		totalEntries = math.MaxInt32
	}

	return connect.NewResponse(&filepreviewv1.ListDirectoryResponse{
		ResolvedPath:  res.ResolvedPath,
		ParentPath:    res.ParentPath,
		Entries:       entries,
		TotalEntries:  int32(totalEntries),
		Truncated:     res.Truncated,
		NextPageToken: res.NextPageToken,
		EffectiveSort: sortToProto(res.EffectiveSort),
		Warnings:      append([]string(nil), res.Warnings...),
	}), nil
}

// classify maps the package's sentinel errors to Connect codes.
func (h *connectHandler) classify(err error, op string) error {
	switch {
	case errors.Is(err, ErrSessionNotFound), errors.Is(err, ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ErrInvalidArgument):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, ErrPermissionDenied):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, ErrPreviewUnavailable):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, ErrStale):
		return connect.NewError(connect.CodeAborted, err)
	default:
		h.deps.Logger.Printf("%s: %v", op, err)
		return connect.NewError(connect.CodeInternal, err)
	}
}

// kindToProto maps the filepreview.Kind string form onto the proto enum.
func kindToProto(kind string) filepreviewv1.PreviewKind {
	switch kind {
	case "markdown":
		return filepreviewv1.PreviewKind_PREVIEW_KIND_MARKDOWN
	case "code":
		return filepreviewv1.PreviewKind_PREVIEW_KIND_CODE
	case "text":
		return filepreviewv1.PreviewKind_PREVIEW_KIND_TEXT
	case "svg":
		return filepreviewv1.PreviewKind_PREVIEW_KIND_SVG
	case "image":
		return filepreviewv1.PreviewKind_PREVIEW_KIND_IMAGE
	case "pdf":
		return filepreviewv1.PreviewKind_PREVIEW_KIND_PDF
	case "audio":
		return filepreviewv1.PreviewKind_PREVIEW_KIND_AUDIO
	case "video":
		return filepreviewv1.PreviewKind_PREVIEW_KIND_VIDEO
	case "csv":
		return filepreviewv1.PreviewKind_PREVIEW_KIND_CSV
	case "diff":
		return filepreviewv1.PreviewKind_PREVIEW_KIND_DIFF
	case "directory":
		return filepreviewv1.PreviewKind_PREVIEW_KIND_DIRECTORY
	default:
		return filepreviewv1.PreviewKind_PREVIEW_KIND_UNSUPPORTED
	}
}

// entryKindToProto maps a listing entry's kind. Unlike a resolved target, an
// entry may legitimately have no kind yet: listings classify by extension
// alone, so an unmapped extension stays UNSPECIFIED ("determined on open")
// rather than being asserted as UNSUPPORTED.
func entryKindToProto(kind string) filepreviewv1.PreviewKind {
	if kind == "" {
		return filepreviewv1.PreviewKind_PREVIEW_KIND_UNSPECIFIED
	}
	return kindToProto(kind)
}

// entryTypeToProto maps the filepreview.EntryType string form onto the enum.
func entryTypeToProto(entryType string) filepreviewv1.EntryType {
	switch entryType {
	case "file":
		return filepreviewv1.EntryType_ENTRY_TYPE_FILE
	case "directory":
		return filepreviewv1.EntryType_ENTRY_TYPE_DIRECTORY
	case "symlink":
		return filepreviewv1.EntryType_ENTRY_TYPE_SYMLINK
	case "other":
		return filepreviewv1.EntryType_ENTRY_TYPE_OTHER
	default:
		return filepreviewv1.EntryType_ENTRY_TYPE_UNSPECIFIED
	}
}

// sortToString maps the proto sort enum onto the filepreview.Sort string form.
// UNSPECIFIED yields "", which the listing engine normalizes to its default.
func sortToString(s filepreviewv1.DirectorySort) string {
	switch s {
	case filepreviewv1.DirectorySort_DIRECTORY_SORT_DIRS_FIRST_NAME:
		return "dirs_first_name"
	case filepreviewv1.DirectorySort_DIRECTORY_SORT_NAME:
		return "name"
	case filepreviewv1.DirectorySort_DIRECTORY_SORT_SIZE_DESC:
		return "size_desc"
	case filepreviewv1.DirectorySort_DIRECTORY_SORT_MTIME_DESC:
		return "mtime_desc"
	default:
		return ""
	}
}

// sortToProto maps the applied sort back onto the enum so the client can see
// when an expensive sort was downgraded.
func sortToProto(s string) filepreviewv1.DirectorySort {
	switch s {
	case "dirs_first_name":
		return filepreviewv1.DirectorySort_DIRECTORY_SORT_DIRS_FIRST_NAME
	case "name":
		return filepreviewv1.DirectorySort_DIRECTORY_SORT_NAME
	case "size_desc":
		return filepreviewv1.DirectorySort_DIRECTORY_SORT_SIZE_DESC
	case "mtime_desc":
		return filepreviewv1.DirectorySort_DIRECTORY_SORT_MTIME_DESC
	default:
		return filepreviewv1.DirectorySort_DIRECTORY_SORT_UNSPECIFIED
	}
}

func sourceContextToString(sc filepreviewv1.SourceContext) string {
	switch sc {
	case filepreviewv1.SourceContext_SOURCE_CONTEXT_MESSAGE_LINK:
		return "message_link"
	case filepreviewv1.SourceContext_SOURCE_CONTEXT_INLINE_CODE:
		return "inline_code"
	case filepreviewv1.SourceContext_SOURCE_CONTEXT_CLI:
		return "cli"
	default:
		return ""
	}
}
