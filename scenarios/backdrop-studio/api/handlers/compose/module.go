package compose

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"image"
	"net/http"
	"net/url"
	"os"

	"backdrop-studio/internal/brandpalette"
	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/compose"
	"backdrop-studio/internal/module"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	composev1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/compose"
	composeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/compose/compose_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/shared"
)

func Module(_ *sql.DB) module.Module {
	h := &handler{}
	return module.Module{Name: "compose", Mount: func(r *mux.Router) {
		path, svc := composeconnect.NewComposeServiceHandler(h)
		r.PathPrefix(path).Handler(svc)
	}, Endpoints: Endpoints}
}

type handler struct{}

func (*handler) ResolvePlan(ctx context.Context, req *connect.Request[composev1.ResolvePlanRequest]) (*connect.Response[composev1.ResolvedPlan], error) {
	if req.Msg.GetStyle() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errStyleRequired)
	}
	s := req.Msg.GetStyle()
	style := catalog.Style{ID: s.GetId(), Strategy: s.GetStrategy(), Treatments: s.GetTreatments(), Placements: s.GetPlacements()}
	brief := compose.Brief{}
	if req.Msg.GetBrief() != nil {
		brief.BrandID = req.Msg.GetBrief().GetBrandId()
		brief.Placement = req.Msg.GetBrief().GetPlacement()
		brief.Prompt = req.Msg.GetBrief().GetPrompt()
		brief.Seed = req.Msg.GetBrief().GetSeed()
	}
	tokens := req.Msg.GetBrandTokens()
	if len(tokens) == 0 && brief.BrandID != "" {
		baseURL := os.Getenv("BRAND_MANAGER_URL")
		if baseURL != "" {
			parsed, parseErr := url.Parse(baseURL)
			if parseErr != nil || parsed.Scheme == "" {
				return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("compose: invalid BRAND_MANAGER_URL"))
			}
			fetched, fetchErr := brandpalette.Fetch(ctx, http.DefaultClient, baseURL, brief.BrandID)
			if fetchErr != nil {
				return nil, connect.NewError(connect.CodeFailedPrecondition, fetchErr)
			}
			tokens = fetched
		}
	}
	plan, err := compose.Resolve(style, brief, compose.MapPalette(tokens), req.Msg.GetAdapter(), req.Msg.GetAdapterCommercialUse())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := &composev1.ResolvedPlan{StyleId: plan.StyleID, Strategy: plan.Strategy, ExpectedExecutionPath: plan.ExpectedExecutionPath, Executable: plan.Executable, ResolvedSlots: plan.ResolvedSlots}
	for _, op := range plan.Operations {
		out.Operations = append(out.Operations, &composev1.Operation{Name: op.Name, ParamsJson: op.ParamsJSON})
	}
	return connect.NewResponse(out), nil
}

func (*handler) ComposeDeviceFrame(_ context.Context, req *connect.Request[composev1.ComposeDeviceFrameRequest]) (*connect.Response[composev1.ComposeDeviceFrameResponse], error) {
	if len(req.Msg.GetBackdropPng()) == 0 || len(req.Msg.GetScreenshotPng()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("compose: backdrop_png and screenshot_png are required"))
	}
	if req.Msg.GetWidth() <= 0 || req.Msg.GetHeight() <= 0 || req.Msg.GetSurfaceId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("compose: surface_id and positive target width and height are required"))
	}
	backdrop, _, err := image.Decode(bytes.NewReader(req.Msg.GetBackdropPng()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("compose: decode backdrop: %w", err))
	}
	if backdrop.Bounds().Dx() != int(req.Msg.GetWidth()) || backdrop.Bounds().Dy() != int(req.Msg.GetHeight()) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("compose: backdrop geometry %dx%d does not match target %dx%d", backdrop.Bounds().Dx(), backdrop.Bounds().Dy(), req.Msg.GetWidth(), req.Msg.GetHeight()))
	}
	png, region, err := compose.ComposeDeviceFrame(req.Msg.GetBackdropPng(), req.Msg.GetScreenshotPng(), compose.DeviceArrangement(req.Msg.GetArrangement()), req.Msg.GetCaption())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&composev1.ComposeDeviceFrameResponse{ImagePng: png, Width: req.Msg.GetWidth(), Height: req.Msg.GetHeight(), OcclusionRegion: &sharedv1.ReservedRegion{X: region.X, Y: region.Y, Width: region.Width, Height: region.Height, Kind: region.Kind}}), nil
}

var errStyleRequired = &styleRequiredError{}

type styleRequiredError struct{}

func (*styleRequiredError) Error() string { return "compose: style is required" }

var Endpoints = []module.EndpointDescriptor{
	{ID: "compose_plan_resolve", Path: composeconnect.ComposeServiceResolvePlanProcedure, Method: http.MethodPost, Summary: "Resolve an inspectable composition plan", Category: "compose"},
	{ID: "compose_device_frame", Path: composeconnect.ComposeServiceComposeDeviceFrameProcedure, Method: http.MethodPost, Summary: "Compose a supplied screenshot into a device frame", Category: "compose"},
}
