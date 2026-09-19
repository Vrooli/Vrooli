package styles

import (
	"context"
	"errors"
	"log"

	"brand-manager/internal/styles"

	"connectrpc.com/connect"

	stylesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/styles"
	stylesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/styles/styles_v1connect"
)

// Deps wires the handler's seams.
type Deps struct {
	Service *styles.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the StylesService handler.
func NewConnectHandler(d Deps) *connectHandler { return &connectHandler{deps: d} }

var _ stylesconnect.StylesServiceHandler = (*connectHandler)(nil)

func (h *connectHandler) ListContainerStyles(ctx context.Context, _ *connect.Request[stylesv1.ListContainerStylesRequest]) (*connect.Response[stylesv1.ListContainerStylesResponse], error) {
	list, err := h.deps.Service.ListStyles(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &stylesv1.ListContainerStylesResponse{}
	for _, s := range list {
		resp.Styles = append(resp.Styles, styleToProto(s))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetContainerStyle(ctx context.Context, req *connect.Request[stylesv1.GetContainerStyleRequest]) (*connect.Response[stylesv1.GetContainerStyleResponse], error) {
	s, err := h.deps.Service.GetStyle(ctx, req.Msg.GetId())
	if err != nil {
		return nil, translate(err)
	}
	return connect.NewResponse(&stylesv1.GetContainerStyleResponse{Style: styleToProto(s)}), nil
}

func (h *connectHandler) CreateContainerStyle(ctx context.Context, req *connect.Request[stylesv1.CreateContainerStyleRequest]) (*connect.Response[stylesv1.CreateContainerStyleResponse], error) {
	m := req.Msg
	s, err := h.deps.Service.CreateStyle(ctx, styles.ContainerStyle{
		Name:                 m.GetName(),
		Shape:                m.GetShape(),
		CornerRatio:          m.GetCornerRatio(),
		BackgroundKind:       m.GetBackgroundKind(),
		BackgroundTop:        m.GetBackgroundTop(),
		BackgroundBottom:     m.GetBackgroundBottom(),
		MarkScale:            m.GetMarkScale(),
		MaskableScale:        m.GetMaskableScale(),
		AccentColor:          m.GetAccentColor(),
		Glow:                 glowFromProto(m.GetGlow()),
		SmallMarkThresholdPx: int(m.GetSmallMarkThresholdPx()),
	})
	if err != nil {
		return nil, translate(err)
	}
	return connect.NewResponse(&stylesv1.CreateContainerStyleResponse{Style: styleToProto(s)}), nil
}

func (h *connectHandler) UpdateContainerStyle(ctx context.Context, req *connect.Request[stylesv1.UpdateContainerStyleRequest]) (*connect.Response[stylesv1.UpdateContainerStyleResponse], error) {
	m := req.Msg
	// Load, then merge non-zero fields: a partial update keeps stored values.
	current, err := h.deps.Service.GetStyle(ctx, m.GetId())
	if err != nil {
		return nil, translate(err)
	}
	if m.GetName() != "" {
		current.Name = m.GetName()
	}
	if m.GetShape() != "" {
		current.Shape = m.GetShape()
	}
	if m.GetCornerRatio() != 0 {
		current.CornerRatio = m.GetCornerRatio()
	}
	if m.GetBackgroundKind() != "" {
		current.BackgroundKind = m.GetBackgroundKind()
	}
	if m.GetBackgroundTop() != "" {
		current.BackgroundTop = m.GetBackgroundTop()
	}
	if m.GetBackgroundBottom() != "" {
		current.BackgroundBottom = m.GetBackgroundBottom()
	}
	if m.GetMarkScale() != 0 {
		current.MarkScale = m.GetMarkScale()
	}
	if m.GetMaskableScale() != 0 {
		current.MaskableScale = m.GetMaskableScale()
	}
	if m.GetAccentColor() != "" {
		current.AccentColor = m.GetAccentColor()
	}
	if len(m.GetGlow()) > 0 {
		current.Glow = glowFromProto(m.GetGlow())
	}
	if m.GetSmallMarkThresholdPx() != 0 {
		current.SmallMarkThresholdPx = int(m.GetSmallMarkThresholdPx())
	}
	updated, err := h.deps.Service.UpdateStyle(ctx, current)
	if err != nil {
		return nil, translate(err)
	}
	return connect.NewResponse(&stylesv1.UpdateContainerStyleResponse{Style: styleToProto(updated)}), nil
}

func (h *connectHandler) ListProductLines(ctx context.Context, _ *connect.Request[stylesv1.ListProductLinesRequest]) (*connect.Response[stylesv1.ListProductLinesResponse], error) {
	lines, err := h.deps.Service.ListProductLines(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &stylesv1.ListProductLinesResponse{}
	for _, l := range lines {
		resp.Lines = append(resp.Lines, lineToProto(l))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) CreateProductLine(ctx context.Context, req *connect.Request[stylesv1.CreateProductLineRequest]) (*connect.Response[stylesv1.CreateProductLineResponse], error) {
	line, err := h.deps.Service.CreateProductLine(ctx, styles.ProductLine{
		Name:             req.Msg.GetName(),
		ContainerStyleID: req.Msg.GetContainerStyleId(),
		Products:         req.Msg.GetProducts(),
	})
	if err != nil {
		return nil, translate(err)
	}
	return connect.NewResponse(&stylesv1.CreateProductLineResponse{Line: lineToProto(line)}), nil
}

func translate(err error) error {
	var nf styles.ErrNotFound
	if errors.As(err, &nf) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
