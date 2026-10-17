package docs

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	internaldocs "landing-page-react-vite-api/internal/docs"
)

// Deps wires the docs Connect handler.
type Deps struct {
	Service *internaldocs.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the DocsService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetDocsTree(ctx context.Context, _ *connect.Request[landingv1.GetDocsTreeRequest]) (*connect.Response[landingv1.GetDocsTreeResponse], error) {
	entries, err := h.deps.Service.Tree()
	if err != nil {
		h.deps.Logger.Printf("docs.GetDocsTree: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to read docs directory"))
	}
	return connect.NewResponse(&landingv1.GetDocsTreeResponse{Entries: entriesToProto(entries)}), nil
}

func (h *connectHandler) GetDocContent(ctx context.Context, req *connect.Request[landingv1.GetDocContentRequest]) (*connect.Response[landingv1.GetDocContentResponse], error) {
	content, err := h.deps.Service.Read(req.Msg.Path)
	if err != nil {
		var invalid *internaldocs.InvalidPathError
		var notFound *internaldocs.NotFoundError
		switch {
		case errors.As(err, &invalid):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.As(err, &notFound):
			return nil, connect.NewError(connect.CodeNotFound, err)
		default:
			h.deps.Logger.Printf("docs.GetDocContent(%q): %v", req.Msg.Path, err)
			return nil, connect.NewError(connect.CodeInternal, errors.New("failed to read file"))
		}
	}
	return connect.NewResponse(&landingv1.GetDocContentResponse{
		Path:    content.Path,
		Content: content.Content,
		Title:   content.Title,
	}), nil
}

func entriesToProto(entries []internaldocs.Entry) []*landingv1.DocEntry {
	out := make([]*landingv1.DocEntry, 0, len(entries))
	for i := range entries {
		out = append(out, entryToProto(entries[i]))
	}
	return out
}

func entryToProto(e internaldocs.Entry) *landingv1.DocEntry {
	return &landingv1.DocEntry{
		Name:     e.Name,
		Path:     e.Path,
		IsDir:    e.IsDir,
		Children: entriesToProto(e.Children),
	}
}
