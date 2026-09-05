package looks

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"

	internallooks "image-tools/internal/looks"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	looksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/looks"
)

// BlobStore is the storage surface RenderPreview needs to persist a rendered
// thumbnail (satisfied by *internal/storage.Store).
type BlobStore interface {
	Put(ctx context.Context, key string, r io.Reader, mime string) error
}

// Deps wires the seams the Connect looks handler needs.
type Deps struct {
	// Store is the Look library (built-ins + custom persistence).
	Store *internallooks.Store
	// Blobs persists rendered preview thumbnails. May be nil (RenderPreview then
	// refuses with an actionable error).
	Blobs  BlobStore
	Logger *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the LooksService handler. Store is required.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListLooks(ctx context.Context, req *connect.Request[looksv1.ListLooksRequest]) (*connect.Response[looksv1.ListLooksResponse], error) {
	list, err := h.deps.Store.List(ctx, req.Msg.GetKind())
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&looksv1.ListLooksResponse{Looks: list}), nil
}

func (h *connectHandler) GetLook(ctx context.Context, req *connect.Request[looksv1.GetLookRequest]) (*connect.Response[looksv1.GetLookResponse], error) {
	look, err := h.deps.Store.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&looksv1.GetLookResponse{Look: look}), nil
}

func (h *connectHandler) CreateLook(ctx context.Context, req *connect.Request[looksv1.CreateLookRequest]) (*connect.Response[looksv1.CreateLookResponse], error) {
	look, err := h.deps.Store.Create(ctx, req.Msg.GetLook())
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&looksv1.CreateLookResponse{Look: look}), nil
}

func (h *connectHandler) UpdateLook(ctx context.Context, req *connect.Request[looksv1.UpdateLookRequest]) (*connect.Response[looksv1.UpdateLookResponse], error) {
	look, err := h.deps.Store.Update(ctx, req.Msg.GetLook())
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&looksv1.UpdateLookResponse{Look: look}), nil
}

func (h *connectHandler) DeleteLook(ctx context.Context, req *connect.Request[looksv1.DeleteLookRequest]) (*connect.Response[looksv1.DeleteLookResponse], error) {
	if err := h.deps.Store.Delete(ctx, req.Msg.GetId()); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&looksv1.DeleteLookResponse{Deleted: true}), nil
}

func (h *connectHandler) CompileLook(ctx context.Context, req *connect.Request[looksv1.CompileLookRequest]) (*connect.Response[looksv1.CompileLookResponse], error) {
	look, err := h.deps.Store.Get(ctx, req.Msg.GetLookId())
	if err != nil {
		return nil, mapErr(err)
	}
	out := internallooks.Compile(look, req.Msg.GetSubject(), req.Msg.GetPrompt(), req.Msg.GetHasInput())
	return connect.NewResponse(out), nil
}

func (h *connectHandler) RenderPreview(ctx context.Context, req *connect.Request[looksv1.RenderPreviewRequest]) (*connect.Response[looksv1.RenderPreviewResponse], error) {
	id := req.Msg.GetLookId()
	look, err := h.deps.Store.Get(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	if h.deps.Blobs == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("preview storage is not configured"))
	}
	pngBytes, deferred, err := internallooks.RenderPreview(look)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	key := "thumb/" + uuid.NewString() + ".png"
	if err := h.deps.Blobs.Put(ctx, key, bytes.NewReader(pngBytes), "image/png"); err != nil {
		h.deps.Logger.Printf("looks.RenderPreview store: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to store preview"))
	}
	// Persist the ref on custom Looks only (built-ins are read-only — the caller
	// gets the rendered ref but the seed entry is untouched).
	if !h.deps.Store.IsBuiltin(id) {
		if err := h.deps.Store.SetThumbnail(ctx, id, key); err != nil {
			return nil, mapErr(err)
		}
	}
	return connect.NewResponse(&looksv1.RenderPreviewResponse{ThumbnailRef: key, DeferredSteps: deferred}), nil
}

// mapErr translates store errors into Connect codes with actionable messages.
func mapErr(err error) error {
	switch {
	case errors.Is(err, internallooks.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, internallooks.ErrBuiltinReadOnly):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, internallooks.ErrIDCollision):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, internallooks.ErrInvalid):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
