package contextcapture

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/api-core/targetmodel"
	wire "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/contextcapture"
	"google.golang.org/protobuf/types/known/timestamppb"
	domain "portal/internal/contextcapture"
)

type Handler struct {
	service  Service
	identity owneridentity.Validator
	now      func() time.Time
}

type Service interface {
	Import(context.Context, string, domain.Import, time.Duration) (domain.Document, error)
	Read(context.Context, string, string) (domain.Document, []byte, error)
	Render(context.Context, string, string) (domain.Document, []byte, string, error)
	ValidateReference(context.Context, string, string) error
	Delete(context.Context, string, string) error
	ReconcileImport(context.Context, string, string) (domain.ImportStatus, error)
	CancelImport(context.Context, string, string) error
}

func NewHandler(service Service, identity owneridentity.Validator, now func() time.Time) *Handler {
	if now == nil {
		now = time.Now
	}
	return &Handler{service: service, identity: identity, now: now}
}

func (h *Handler) owner(ctx context.Context, header http.Header) (string, error) {
	token, ok := strings.CutPrefix(header.Get("Authorization"), "Bearer ")
	if !ok || token == "" || len(token) > 16384 || h.identity == nil {
		return "", connect.NewError(connect.CodeUnauthenticated, errors.New("operator identity required"))
	}
	identity, err := h.identity.Validate(ctx, token)
	if err != nil || identity.Subject == "" || !h.now().Before(identity.ExpiresAt) {
		return "", connect.NewError(connect.CodeUnauthenticated, errors.New("operator identity unavailable"))
	}
	return identity.Subject, nil
}

func failure(err error) error {
	code := connect.CodeUnavailable
	if errors.Is(err, domain.ErrInvalid) {
		code = connect.CodeInvalidArgument
	}
	if errors.Is(err, domain.ErrConflict) {
		code = connect.CodeAlreadyExists
	}
	if errors.Is(err, domain.ErrQuota) {
		code = connect.CodeResourceExhausted
	}
	if errors.Is(err, domain.ErrUnavailable) {
		code = connect.CodeNotFound
	}
	return connect.NewError(code, errors.New("context operation unavailable"))
}

func (h *Handler) Import(ctx context.Context, r *connect.Request[wire.ImportRequest]) (*connect.Response[wire.Document], error) {
	owner, err := h.owner(ctx, r.Header())
	if err != nil {
		return nil, err
	}
	input, err := fromWire(r.Msg)
	if err != nil {
		return nil, failure(err)
	}
	doc, err := h.service.Import(ctx, owner, input, time.Duration(r.Msg.RetentionSeconds)*time.Second)
	if err != nil {
		return nil, failure(err)
	}
	return connect.NewResponse(toWire(doc)), nil
}

func (h *Handler) Read(ctx context.Context, r *connect.Request[wire.ReferenceRequest]) (*connect.Response[wire.ReadResponse], error) {
	owner, err := h.owner(ctx, r.Header())
	if err != nil {
		return nil, err
	}
	doc, pixels, err := h.service.Read(ctx, owner, r.Msg.Id)
	if err != nil {
		return nil, failure(err)
	}
	current, err := h.owner(ctx, r.Header())
	if err != nil {
		return nil, err
	}
	if current != owner || !h.now().Before(doc.ExpiresAt) {
		return nil, failure(domain.ErrUnavailable)
	}
	return connect.NewResponse(&wire.ReadResponse{Document: toWire(doc), Png: pixels}), nil
}

func (h *Handler) Delete(ctx context.Context, r *connect.Request[wire.ReferenceRequest]) (*connect.Response[wire.DeleteResponse], error) {
	owner, err := h.owner(ctx, r.Header())
	if err != nil {
		return nil, err
	}
	if err = h.service.Delete(ctx, owner, r.Msg.Id); err != nil {
		return nil, failure(err)
	}
	return connect.NewResponse(&wire.DeleteResponse{}), nil
}

func fromWire(r *wire.ImportRequest) (domain.Import, error) {
	if r.Source == nil || r.Source.Bounds == nil || r.Region == nil || r.Source.CapturedAt == nil || r.Source.CapturedAt.CheckValid() != nil {
		return domain.Import{}, domain.ErrInvalid
	}
	s := r.Source
	surface, err := targetmodel.SurfaceRefFromProto(s.Surface)
	if err != nil {
		return domain.Import{}, domain.ErrInvalid
	}
	input := domain.Import{RequestID: r.RequestId, Source: domain.Source{Surface: surface, CaptureID: s.CaptureId, DisplayID: s.DisplayId, GeometryRevision: s.GeometryRevision, CapturedAt: s.CapturedAt.AsTime(), Bounds: domain.Bounds{X: s.Bounds.X, Y: s.Bounds.Y, Width: s.Bounds.Width, Height: s.Bounds.Height}}, Region: domain.Region{X: r.Region.X, Y: r.Region.Y, Width: r.Region.Width, Height: r.Region.Height}, PNG: r.Png}
	if len(r.Strokes) > 32 {
		return domain.Import{}, domain.ErrInvalid
	}
	for _, stroke := range r.Strokes {
		if stroke == nil || len(stroke.Points) > 256 {
			return domain.Import{}, domain.ErrInvalid
		}
		points := []domain.Point{}
		for _, p := range stroke.Points {
			if p == nil {
				return domain.Import{}, domain.ErrInvalid
			}
			points = append(points, domain.Point{X: p.X, Y: p.Y})
		}
		input.Strokes = append(input.Strokes, points)
	}
	return input, nil
}

func toWire(doc domain.Document) *wire.Document {
	s := doc.Source
	r := doc.Region
	out := &wire.Document{RequestId: doc.RequestID, Id: doc.ID, Source: &wire.Source{Surface: s.Surface.Proto(), CaptureId: s.CaptureID, DisplayId: s.DisplayID, GeometryRevision: s.GeometryRevision, CapturedAt: timestamppb.New(s.CapturedAt), Bounds: &wire.Bounds{X: s.Bounds.X, Y: s.Bounds.Y, Width: s.Bounds.Width, Height: s.Bounds.Height}}, Region: &wire.Region{X: r.X, Y: r.Y, Width: r.Width, Height: r.Height}, OriginalSha256: doc.OriginalSHA256, CreatedAt: timestamppb.New(doc.CreatedAt), ExpiresAt: timestamppb.New(doc.ExpiresAt)}
	for _, stroke := range doc.Strokes {
		entry := &wire.Stroke{}
		for _, p := range stroke {
			entry.Points = append(entry.Points, &wire.Point{X: p.X, Y: p.Y})
		}
		out.Strokes = append(out.Strokes, entry)
	}
	return out
}

func (h *Handler) ReconcileImport(ctx context.Context, r *connect.Request[wire.ReconcileImportRequest]) (*connect.Response[wire.ReconcileImportResponse], error) {
	owner, err := h.owner(ctx, r.Header())
	if err != nil {
		return nil, err
	}
	status, err := h.service.ReconcileImport(ctx, owner, r.Msg.RequestId)
	if err != nil {
		return nil, failure(err)
	}
	current, err := h.owner(ctx, r.Header())
	if err != nil {
		return nil, err
	}
	if current != owner {
		return nil, failure(domain.ErrUnavailable)
	}
	response := &wire.ReconcileImportResponse{RequestId: r.Msg.RequestId}
	switch status.State {
	case "absent":
		response.State = wire.ImportState_IMPORT_STATE_ABSENT
	case "staging":
		response.State = wire.ImportState_IMPORT_STATE_STAGING
	case "ready":
		if status.Document == nil || !h.now().Before(status.Document.ExpiresAt) {
			response.State = wire.ImportState_IMPORT_STATE_UNAVAILABLE
		} else {
			response.State = wire.ImportState_IMPORT_STATE_READY
			response.Document = toWire(*status.Document)
		}
	default:
		response.State = wire.ImportState_IMPORT_STATE_UNAVAILABLE
	}
	return connect.NewResponse(response), nil
}

func (h *Handler) CancelImport(ctx context.Context, r *connect.Request[wire.ReconcileImportRequest]) (*connect.Response[wire.DeleteResponse], error) {
	owner, err := h.owner(ctx, r.Header())
	if err != nil {
		return nil, err
	}
	if err = h.service.CancelImport(ctx, owner, r.Msg.RequestId); err != nil {
		return nil, failure(err)
	}
	return connect.NewResponse(&wire.DeleteResponse{}), nil
}

func (h *Handler) Render(ctx context.Context, r *connect.Request[wire.ReferenceRequest]) (*connect.Response[wire.RenderResponse], error) {
	owner, err := h.owner(ctx, r.Header())
	if err != nil {
		return nil, err
	}
	doc, pixels, digest, err := h.service.Render(ctx, owner, r.Msg.Id)
	if err != nil {
		return nil, failure(err)
	}
	current, err := h.owner(ctx, r.Header())
	if err != nil {
		return nil, err
	}
	if current != owner || !h.now().Before(doc.ExpiresAt) {
		return nil, failure(domain.ErrUnavailable)
	}
	return connect.NewResponse(&wire.RenderResponse{Document: toWire(doc), Png: pixels, RenderedSha256: digest}), nil
}
