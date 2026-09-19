package variant

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	internalcontent "landing-page-react-vite-api/internal/content"
	internalvariant "landing-page-react-vite-api/internal/variant"
)

// Deps wires the variant Connect handler.
type Deps struct {
	Service *internalvariant.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the VariantService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) SelectVariant(ctx context.Context, _ *connect.Request[landingv1.SelectVariantRequest]) (*connect.Response[landingv1.VariantResponse], error) {
	v, err := h.deps.Service.SelectVariant()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.VariantResponse{Variant: variantToProto(v, false)}), nil
}

func (h *connectHandler) GetPublicVariant(ctx context.Context, req *connect.Request[landingv1.GetPublicVariantRequest]) (*connect.Response[landingv1.VariantResponse], error) {
	if req.Msg.Slug == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("variant slug required"))
	}
	v, err := h.deps.Service.GetVariantBySlug(req.Msg.Slug)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if v.Status != "active" {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("variant not available"))
	}
	return connect.NewResponse(&landingv1.VariantResponse{Variant: variantToProto(v, false)}), nil
}

func (h *connectHandler) GetVariant(ctx context.Context, req *connect.Request[landingv1.GetVariantRequest]) (*connect.Response[landingv1.VariantResponse], error) {
	if req.Msg.Slug == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("variant slug required"))
	}
	v, err := h.deps.Service.GetVariantBySlug(req.Msg.Slug)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&landingv1.VariantResponse{Variant: variantToProto(v, true)}), nil
}

func (h *connectHandler) ListVariants(ctx context.Context, req *connect.Request[landingv1.ListVariantsRequest]) (*connect.Response[landingv1.ListVariantsResponse], error) {
	variants, err := h.deps.Service.ListVariants(req.Msg.StatusFilter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*landingv1.Variant, 0, len(variants))
	for i := range variants {
		out = append(out, variantToProto(&variants[i], false))
	}
	return connect.NewResponse(&landingv1.ListVariantsResponse{Variants: out}), nil
}

func (h *connectHandler) CreateVariant(ctx context.Context, req *connect.Request[landingv1.CreateVariantRequest]) (*connect.Response[landingv1.VariantResponse], error) {
	m := req.Msg
	if m.Slug == "" || m.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("slug and name are required"))
	}
	if len(m.Axes) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("axes selection is required"))
	}
	v, err := h.deps.Service.CreateVariant(m.Slug, m.Name, m.Description, int(m.Weight), m.Axes)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	// Seed the new variant with the control's sections (non-fatal on failure).
	h.deps.Service.CopyControlSections(int64(v.ID))
	return connect.NewResponse(&landingv1.VariantResponse{Variant: variantToProto(v, false)}), nil
}

func (h *connectHandler) UpdateVariant(ctx context.Context, req *connect.Request[landingv1.UpdateVariantRequest]) (*connect.Response[landingv1.VariantResponse], error) {
	m := req.Msg
	if m.Slug == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("variant slug required"))
	}
	var weight *int
	if m.Weight != nil {
		w := int(*m.Weight)
		weight = &w
	}
	var axes map[string]string
	if m.Axes != nil {
		axes = m.Axes.Values
		if axes == nil {
			axes = map[string]string{}
		}
	}
	v, err := h.deps.Service.UpdateVariant(m.Slug, m.Name, m.Description, weight, axes, headerFromProto(m.HeaderConfig))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&landingv1.VariantResponse{Variant: variantToProto(v, false)}), nil
}

func (h *connectHandler) ArchiveVariant(ctx context.Context, req *connect.Request[landingv1.ArchiveVariantRequest]) (*connect.Response[landingv1.VariantResponse], error) {
	if req.Msg.Slug == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("variant slug required"))
	}
	if err := h.deps.Service.ArchiveVariant(req.Msg.Slug); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	v, err := h.deps.Service.GetVariantBySlug(req.Msg.Slug)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.VariantResponse{Variant: variantToProto(v, false)}), nil
}

func (h *connectHandler) DeleteVariant(ctx context.Context, req *connect.Request[landingv1.DeleteVariantRequest]) (*connect.Response[landingv1.DeleteVariantResponse], error) {
	if req.Msg.Slug == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("variant slug required"))
	}
	if err := h.deps.Service.DeleteVariant(req.Msg.Slug); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&landingv1.DeleteVariantResponse{Deleted: true}), nil
}

func (h *connectHandler) ExportVariantSnapshot(ctx context.Context, req *connect.Request[landingv1.ExportVariantSnapshotRequest]) (*connect.Response[landingv1.ExportVariantSnapshotResponse], error) {
	if req.Msg.Slug == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("variant slug required"))
	}
	snap, err := h.deps.Service.ExportSnapshot(req.Msg.Slug)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&landingv1.ExportVariantSnapshotResponse{Snapshot: snapshotToProto(snap)}), nil
}

func (h *connectHandler) ImportVariantSnapshot(ctx context.Context, req *connect.Request[landingv1.ImportVariantSnapshotRequest]) (*connect.Response[landingv1.VariantResponse], error) {
	m := req.Msg
	if m.Slug == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("variant slug required"))
	}
	if m.Snapshot == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("snapshot required"))
	}
	input, err := snapshotInputFromProto(m.Snapshot)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if _, err := h.deps.Service.ImportSnapshot(m.Slug, input); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	v, err := h.deps.Service.GetVariantBySlug(m.Slug)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.VariantResponse{Variant: variantToProto(v, true)}), nil
}

func snapshotToProto(s *internalvariant.Snapshot) *landingv1.VariantSnapshot {
	p := &landingv1.VariantSnapshot{
		Slug:         s.Slug,
		Name:         s.Name,
		Description:  s.Description,
		Weight:       int32(s.Weight),
		Status:       s.Status,
		Axes:         s.Axes,
		HeaderConfig: headerToProto(s.HeaderConfig),
	}
	if len(s.SEOConfig) > 0 {
		p.SeoConfig = seoRawToProto(s.SEOConfig)
	}
	for i := range s.Sections {
		p.Sections = append(p.Sections, sectionToProto(s.Sections[i]))
	}
	return p
}

func snapshotInputFromProto(p *landingv1.VariantSnapshot) (internalvariant.SnapshotInput, error) {
	input := internalvariant.SnapshotInput{
		Slug:         p.Slug,
		Name:         p.Name,
		Description:  p.Description,
		Weight:       int(p.Weight),
		Status:       p.Status,
		Axes:         p.Axes,
		HeaderConfig: headerFromProto(p.HeaderConfig),
	}
	if p.SeoConfig != nil {
		raw, err := seoProtoToRaw(p.SeoConfig)
		if err != nil {
			return input, err
		}
		input.SEOConfig = raw
	}
	input.Sections = make([]internalcontent.SectionInput, 0, len(p.Sections))
	for _, s := range p.Sections {
		input.Sections = append(input.Sections, sectionInputFromProto(s))
	}
	return input, nil
}
