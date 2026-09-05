package snippets

import (
	"context"
	"errors"
	"log"
	"time"

	"connectrpc.com/connect"
	snippetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/snippets"

	snippetdomain "web-console/internal/snippets"
)

type Deps struct {
	Service Service
	Logger  *log.Logger
	Now     func() time.Time
	Runner  CommandRunner
}

type connectHandler struct{ deps Deps }

func NewConnectHandler(deps Deps) *connectHandler {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.Runner == nil {
		deps.Runner = execCommandRunner{}
	}
	return &connectHandler{deps: deps}
}

func (h *connectHandler) mapError(op string, err error) error {
	switch {
	case errors.Is(err, snippetdomain.ErrInvalidSnippet):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, snippetdomain.ErrSnippetNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ErrPromptManagerUnavailable):
		return connect.NewError(connect.CodeUnavailable, ErrPromptManagerUnavailable)
	case isCommandFailure(err):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		h.deps.Logger.Printf("snippets.%s: %v", op, err)
		return connect.NewError(connect.CodeInternal, err)
	}
}

func isCommandFailure(err error) bool {
	var failure *CommandFailure
	return errors.As(err, &failure)
}

func (h *connectHandler) ListSnippets(ctx context.Context, _ *connect.Request[snippetsv1.ListSnippetsRequest]) (*connect.Response[snippetsv1.ListSnippetsResponse], error) {
	items, err := h.deps.Service.List(ctx)
	if err != nil {
		return nil, h.mapError("ListSnippets", err)
	}
	out := make([]*snippetsv1.Snippet, 0, len(items))
	for _, item := range items {
		out = append(out, snippetToProto(item))
	}
	return connect.NewResponse(&snippetsv1.ListSnippetsResponse{Snippets: out}), nil
}

func (h *connectHandler) UpsertSnippet(ctx context.Context, req *connect.Request[snippetsv1.UpsertSnippetRequest]) (*connect.Response[snippetsv1.UpsertSnippetResponse], error) {
	item, err := h.deps.Service.Upsert(ctx, UpsertRequest{
		ID: req.Msg.GetId(), Name: req.Msg.GetName(), Body: req.Msg.GetBody(),
		Color: req.Msg.GetColor(), Pinned: req.Msg.GetPinned(),
		HasPinned: req.Msg.GetHasPinned(), SortOrder: int(req.Msg.GetSortOrder()),
	})
	if err != nil {
		return nil, h.mapError("UpsertSnippet", err)
	}
	return connect.NewResponse(&snippetsv1.UpsertSnippetResponse{Snippet: snippetToProto(item)}), nil
}

func (h *connectHandler) DeleteSnippet(ctx context.Context, req *connect.Request[snippetsv1.DeleteSnippetRequest]) (*connect.Response[snippetsv1.DeleteSnippetResponse], error) {
	deleted, err := h.deps.Service.Delete(ctx, req.Msg.GetId())
	if err != nil {
		return nil, h.mapError("DeleteSnippet", err)
	}
	return connect.NewResponse(&snippetsv1.DeleteSnippetResponse{Deleted: deleted}), nil
}

func (h *connectHandler) TouchSnippet(ctx context.Context, req *connect.Request[snippetsv1.TouchSnippetRequest]) (*connect.Response[snippetsv1.TouchSnippetResponse], error) {
	item, err := h.deps.Service.Touch(ctx, req.Msg.GetId(), h.deps.Now())
	if err != nil {
		return nil, h.mapError("TouchSnippet", err)
	}
	return connect.NewResponse(&snippetsv1.TouchSnippetResponse{Snippet: snippetToProto(item)}), nil
}

func (h *connectHandler) PromoteSnippet(ctx context.Context, req *connect.Request[snippetsv1.PromoteSnippetRequest]) (*connect.Response[snippetsv1.PromoteSnippetResponse], error) {
	items, err := h.deps.Service.List(ctx)
	if err != nil {
		return nil, h.mapError("PromoteSnippet", err)
	}
	var selected *Snippet
	for index := range items {
		if items[index].ID == req.Msg.GetId() {
			selected = &items[index]
			break
		}
	}
	if selected == nil {
		return nil, h.mapError("PromoteSnippet", snippetdomain.ErrSnippetNotFound)
	}

	identifier, err := promoteSnippet(ctx, h.deps.Runner, selected.Name, selected.Body)
	if err != nil {
		return nil, h.mapError("PromoteSnippet", err)
	}
	return connect.NewResponse(&snippetsv1.PromoteSnippetResponse{Identifier: identifier}), nil
}

func snippetToProto(item Snippet) *snippetsv1.Snippet {
	return &snippetsv1.Snippet{
		Id: item.ID, Name: item.Name, Body: item.Body, Color: item.Color,
		Pinned: item.Pinned, UseCount: int32(item.UseCount), LastUsedAt: item.LastUsedAt,
		SortOrder: int32(item.SortOrder), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}
