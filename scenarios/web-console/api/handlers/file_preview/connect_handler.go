package file_preview

import (
	"context"
	"errors"
	"log"
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
	default:
		return filepreviewv1.PreviewKind_PREVIEW_KIND_UNSUPPORTED
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
