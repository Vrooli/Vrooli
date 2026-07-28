// Package pricing owns the public catalog Connect transport.
package pricing

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

type Handler struct {
	overview func() (*sharedv1.PricingOverview, error)
}

func NewHandler(overview func() (*sharedv1.PricingOverview, error)) *Handler {
	return &Handler{overview: overview}
}

func (h *Handler) GetPricing(_ context.Context, request *connect.Request[lpbsv1.GetPricingRequest]) (*connect.Response[lpbsv1.GetPricingResponse], error) {
	if request.Msg.GetIncludeHidden() {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("include_hidden is not available on the public pricing catalog"))
	}
	overview, err := h.overview()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load pricing overview: %w", err))
	}
	if requested := request.Msg.GetBundleKey(); requested != "" && requested != overview.GetBundle().GetBundleKey() {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("bundle %q is not available", requested))
	}
	return connect.NewResponse(&lpbsv1.GetPricingResponse{Pricing: overview}), nil
}

// RegisterRoutes mounts PricingService at its generated Connect service path.
func RegisterRoutes(router *mux.Router, overview func() (*sharedv1.PricingOverview, error)) {
	path, handler := lpbsconnect.NewPricingServiceHandler(NewHandler(overview))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}
