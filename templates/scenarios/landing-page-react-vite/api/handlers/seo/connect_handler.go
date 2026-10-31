package seo

import (
	"context"
	"log"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	internalseo "landing-page-react-vite-api/internal/seo"
	internalvariant "landing-page-react-vite-api/internal/variant"
)

// Deps wires the seo Connect handler.
type Deps struct {
	Service *internalseo.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the SeoService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetVariantSEO(ctx context.Context, req *connect.Request[landingv1.GetVariantSEORequest]) (*connect.Response[landingv1.SEOResponse], error) {
	if req.Msg.Slug == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errSlugRequired)
	}
	res, err := h.deps.Service.ResolveHead(ctx, req.Msg.Slug)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	out := &landingv1.SEOResponse{
		SiteName:          res.SiteName,
		Title:             res.Title,
		Description:       res.Description,
		OgTitle:           res.OGTitle,
		OgDescription:     res.OGDescription,
		OgImageUrl:        res.OGImageURL,
		TwitterCard:       res.TwitterCard,
		CanonicalUrl:      res.CanonicalURL,
		FaviconUrl:        res.FaviconURL,
		AppleTouchIconUrl: res.AppleTouchIconURL,
		ThemePrimaryColor: res.ThemePrimaryColor,
		Noindex:           res.NoIndex,
	}
	if len(res.StructuredData) > 0 {
		if s, err := structpb.NewStruct(res.StructuredData); err == nil {
			out.StructuredData = s
		}
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) UpdateVariantSEO(ctx context.Context, req *connect.Request[landingv1.UpdateVariantSEORequest]) (*connect.Response[landingv1.UpdateVariantSEOResponse], error) {
	if req.Msg.Slug == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errSlugRequired)
	}
	updatedAt, err := h.deps.Service.UpdateOverride(req.Msg.Slug, configFromProto(req.Msg.Config))
	if err != nil {
		h.deps.Logger.Printf("seo.UpdateVariantSEO(%q): %v", req.Msg.Slug, err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.UpdateVariantSEOResponse{
		Success:   true,
		UpdatedAt: updatedAt.Format(time.RFC3339),
	}), nil
}

func configFromProto(p *landingv1.VariantSEOConfig) internalvariant.SEOConfigJSON {
	if p == nil {
		return internalvariant.SEOConfigJSON{}
	}
	cfg := internalvariant.SEOConfigJSON{
		Title:         p.Title,
		Description:   p.Description,
		OGTitle:       p.OgTitle,
		OGDescription: p.OgDescription,
		OGImageURL:    p.OgImageUrl,
		TwitterCard:   p.TwitterCard,
		CanonicalPath: p.CanonicalPath,
		NoIndex:       p.Noindex,
	}
	if p.StructuredData != nil {
		cfg.StructuredData = p.StructuredData.AsMap()
	}
	return cfg
}
