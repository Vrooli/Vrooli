package bundles

import (
	"context"
	"errors"
	"landing-page-react-vite-api/internal/plan"
	"log"

	"connectrpc.com/connect"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

// Deps wires the BundleAdmin Connect handler over the plan service.
type Deps struct {
	Plan   *plan.Service
	Logger *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the BundleAdminService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListBundleCatalog(ctx context.Context, _ *connect.Request[landingv1.ListBundleCatalogRequest]) (*connect.Response[landingv1.ListBundleCatalogResponse], error) {
	entries, err := h.deps.Plan.ListBundleCatalog(ctx)
	if err != nil {
		h.deps.Logger.Printf("bundles.ListBundleCatalog: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*landingv1.BundleCatalogEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, &landingv1.BundleCatalogEntry{Bundle: entry.Bundle, Prices: entry.Prices})
	}
	return connect.NewResponse(&landingv1.ListBundleCatalogResponse{Bundles: out}), nil
}

func (h *connectHandler) UpdateBundlePrice(ctx context.Context, req *connect.Request[landingv1.UpdateBundlePriceRequest]) (*connect.Response[landingv1.UpdateBundlePriceResponse], error) {
	m := req.Msg
	if m.BundleKey == "" || m.PriceId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("bundle_key and price_id are required"))
	}

	input := plan.UpdateBundlePriceInput{
		PlanName:       m.PlanName,
		DisplayEnabled: m.DisplayEnabled,
		Subtitle:       m.Subtitle,
		Badge:          m.Badge,
		CtaLabel:       m.CtaLabel,
		Highlight:      m.Highlight,
	}
	if m.DisplayWeight != nil {
		weight := int(*m.DisplayWeight)
		input.DisplayWeight = &weight
	}
	// The proto carries features as a repeated field (always present); apply it
	// as the authoritative desired set so an empty list clears the override.
	features := m.Features
	input.Features = &features

	price, err := h.deps.Plan.UpdateBundlePrice(ctx, m.BundleKey, m.PriceId, input)
	if err != nil {
		h.deps.Logger.Printf("bundles.UpdateBundlePrice(%q/%q): %v", m.BundleKey, m.PriceId, err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&landingv1.UpdateBundlePriceResponse{Price: price}), nil
}
