// Package landing owns the generated public landing-configuration transport.
package landing

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"landing-page-business-suite-api/internal/landing"
)

// LandingConfigConnectHandler exposes the config.proto public landing payload via
// its generated contract without depending on API-root composition.
type LandingConfigConnectHandler struct{ service *landing.LandingConfigService }

func NewLandingConfigConnectHandler(service *landing.LandingConfigService) LandingConfigConnectHandler {
	return LandingConfigConnectHandler{service: service}
}

func (h LandingConfigConnectHandler) GetLandingConfig(ctx context.Context, request *connect.Request[lpbsv1.GetLandingConfigRequest]) (*connect.Response[lpbsv1.LandingConfigResponse], error) {
	response, err := h.service.GetLandingConfigForRequest(ctx, request.Msg.GetVariantSlug(), request.Msg.GetRoute(), request.Msg.GetLocale(), request.Msg.GetVisitorId())
	if err != nil {
		return nil, presentationConnectError(fmt.Errorf("load landing configuration: %w", err))
	}
	message, err := LandingConfigProto(response)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode landing configuration: %w", err))
	}
	result := connect.NewResponse(message)
	if response.Presentation != nil {
		result.Header().Set("Cache-Control", "no-store")
	}
	return result, nil
}

func (h LandingConfigConnectHandler) RecordPresentationExposure(ctx context.Context, request *connect.Request[lpbsv1.RecordPresentationExposureRequest]) (*connect.Response[lpbsv1.RecordPresentationExposureResponse], error) {
	source, err := presentationAssignmentSource(request.Msg.GetSource())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	recorded, err := h.service.RecordPresentationExposure(ctx, landing.PresentationExposureRequest{
		VisitorID: request.Msg.GetVisitorId(), VariantSlug: request.Msg.GetVariantSlug(), Revision: request.Msg.GetRevision(),
		Route: request.Msg.GetRoute(), Locale: request.Msg.GetLocale(), BlockDigest: request.Msg.GetBlockDigest(),
		WeightFingerprint: request.Msg.GetWeightFingerprint(), Source: source,
	})
	if err != nil {
		return nil, presentationConnectError(fmt.Errorf("record presentation exposure: %w", err))
	}
	result := connect.NewResponse(&lpbsv1.RecordPresentationExposureResponse{Recorded: recorded})
	result.Header().Set("Cache-Control", "no-store")
	return result, nil
}

func presentationAssignmentSource(source lpbsv1.PresentationAssignmentSource) (landing.PresentationAssignmentSource, error) {
	switch source {
	case lpbsv1.PresentationAssignmentSource_PRESENTATION_ASSIGNMENT_SOURCE_WEIGHTED_VISITOR:
		return landing.PresentationAssignmentWeightedVisitor, nil
	case lpbsv1.PresentationAssignmentSource_PRESENTATION_ASSIGNMENT_SOURCE_EXPLICIT_URL:
		return landing.PresentationAssignmentExplicitURL, nil
	default:
		return "", fmt.Errorf("assignment source is required")
	}
}

// LandingConfigProto converts domain payloads to their generated public wire contract.
func LandingConfigProto(response *landing.LandingConfigResponse) (*lpbsv1.LandingConfigResponse, error) {
	if response == nil || response.Presentation == nil {
		return nil, fmt.Errorf("typed presentation is required")
	}
	downloads, err := landing.ProtoPresentationDownloads(response.Downloads)
	if err != nil {
		return nil, err
	}
	presentationWire, err := ResolvedPresentationProto(*response.Presentation)
	if err != nil {
		return nil, fmt.Errorf("presentation: %w", err)
	}
	return &lpbsv1.LandingConfigResponse{
		Pricing:      response.Pricing,
		Downloads:    downloads,
		Fallback:     response.Fallback,
		Presentation: presentationWire,
	}, nil
}

func RegisterLandingConfigConnectRoutes(router *mux.Router, service *landing.LandingConfigService) {
	path, handler := lpbsconnect.NewLandingConfigServiceHandler(NewLandingConfigConnectHandler(service))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}

var _ lpbsconnect.LandingConfigServiceHandler = LandingConfigConnectHandler{}
