package bundles

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

// UpdatePriceInput is the transport-neutral representation of a partial price
// update. Pointer fields retain the old JSON endpoint's distinction between an
// omitted field and a deliberate zero/empty value.
type UpdatePriceInput struct {
	StripePriceID  *string
	PlanName       *string
	DisplayWeight  *int
	DisplayEnabled *bool
	Subtitle       *string
	Badge          *string
	CtaLabel       *string
	Highlight      *bool
	Features       *[]string
}

type ConnectDependencies struct {
	ListCatalog func(context.Context) ([]*lpbsv1.BundleCatalogEntry, error)
	UpdatePrice func(context.Context, string, string, UpdatePriceInput) (*sharedv1.PlanOption, error)
	Classify    func(error) connect.Code
}

type ConnectHandler struct {
	deps ConnectDependencies
}

func NewConnectHandler(deps ConnectDependencies) *ConnectHandler {
	return &ConnectHandler{deps: deps}
}

func (h *ConnectHandler) ListBundleCatalog(ctx context.Context, _ *connect.Request[lpbsv1.ListBundleCatalogRequest]) (*connect.Response[lpbsv1.ListBundleCatalogResponse], error) {
	bundles, err := h.deps.ListCatalog(ctx)
	if err != nil {
		return nil, connect.NewError(h.classify(err), fmt.Errorf("list bundle catalog: %w", err))
	}
	return connect.NewResponse(&lpbsv1.ListBundleCatalogResponse{Bundles: bundles}), nil
}

func (h *ConnectHandler) UpdateBundlePrice(ctx context.Context, request *connect.Request[lpbsv1.UpdateBundlePriceRequest]) (*connect.Response[lpbsv1.UpdateBundlePriceResponse], error) {
	message := request.Msg
	bundleKey := strings.TrimSpace(message.GetBundleKey())
	if bundleKey == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("bundle_key is required"))
	}
	priceID := strings.TrimSpace(message.GetPriceId())
	if priceID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("price_id is required"))
	}

	price, err := h.deps.UpdatePrice(ctx, bundleKey, priceID, updatePriceInput(message))
	if err != nil {
		return nil, connect.NewError(h.classify(err), err)
	}
	return connect.NewResponse(&lpbsv1.UpdateBundlePriceResponse{Price: price}), nil
}

func (h *ConnectHandler) classify(err error) connect.Code {
	if h.deps.Classify == nil {
		return connect.CodeInvalidArgument
	}
	return h.deps.Classify(err)
}

func updatePriceInput(message *lpbsv1.UpdateBundlePriceRequest) UpdatePriceInput {
	input := UpdatePriceInput{
		StripePriceID:  message.StripePriceId,
		PlanName:       message.PlanName,
		DisplayEnabled: message.DisplayEnabled,
		Subtitle:       message.Subtitle,
		Badge:          message.Badge,
		CtaLabel:       message.CtaLabel,
		Highlight:      message.Highlight,
	}
	if message.DisplayWeight != nil {
		weight := int(*message.DisplayWeight)
		input.DisplayWeight = &weight
	}
	if message.FeaturesPresent != nil && *message.FeaturesPresent {
		features := append([]string(nil), message.Features...)
		input.Features = &features
	}
	return input
}

// RegisterConnectRoutes mounts each generated procedure with the existing
// administrator boundary. The service has no public methods.
func RegisterConnectRoutes(router *mux.Router, deps ConnectDependencies, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	_, generated := lpbsconnect.NewBundleAdminServiceHandler(NewConnectHandler(deps))
	for _, procedure := range []string{
		lpbsconnect.BundleAdminServiceListBundleCatalogProcedure,
		lpbsconnect.BundleAdminServiceUpdateBundlePriceProcedure,
	} {
		router.Handle(procedure, requireAdmin(generated.ServeHTTP)).Methods(http.MethodPost)
	}
}
