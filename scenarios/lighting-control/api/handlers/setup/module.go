package setup

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/repo-contract-go/cliinvoke"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/setup"
	setupconnect "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/setup/setupconnect"
	"google.golang.org/protobuf/proto"
	"lighting-control/internal/apiauth"
	"lighting-control/internal/module"
	internalsetup "lighting-control/internal/setup"
)

type Operations interface {
	Devices(context.Context) (*setupv1.LightingDeviceDiscovery, error)
	PlanDevices(context.Context, []string) (*setupv1.LightingDevicePlan, error)
	ApplyDevices(context.Context, string) (*setupv1.LightingDeviceAccessResult, error)
	RollbackDevices(context.Context, string) (*setupv1.LightingDeviceAccessResult, error)
	Status(context.Context) (*setupv1.LightingRuntimeStatus, error)
	Setup(context.Context, *setupv1.LightingSelection) (*setupv1.LightingRuntimeStatus, error)
	Enable(context.Context, *setupv1.LightingSelection) (*setupv1.LightingRuntimeStatus, error)
	Disable(context.Context) (*setupv1.LightingRuntimeStatus, error)
}
type Handler struct{ Operations Operations }

func Module() module.Module { return ModuleWithOperations(internalsetup.CLI{}) }
func ModuleWithOperations(ops Operations) module.Module {
	path, handler := setupconnect.NewSetupServiceHandler(&Handler{Operations: ops})
	return module.Module{Name: "setup", Endpoints: Endpoints, Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }}
}
func authorize(ctx context.Context, action string) error {
	scope := "lighting-control:write"
	if action == "status" || action == "devices" {
		scope = "lighting-control:read"
	}
	return apiauth.RequireCapability(ctx, scope)
}
func response[T any](status *T, err error) (*connect.Response[T], error) {
	if err != nil {
		code := connect.CodeFailedPrecondition
		switch {
		case errors.Is(err, context.Canceled):
			code = connect.CodeCanceled
		case errors.Is(err, context.DeadlineExceeded):
			code = connect.CodeDeadlineExceeded
		case errors.Is(err, cliinvoke.ErrBinaryMissing):
			code = connect.CodeUnavailable
		}
		failure := connect.NewError(code, err)
		if detailMessage, ok := any(status).(proto.Message); status != nil && ok && detailMessage != nil {
			if detail, detailErr := connect.NewErrorDetail(detailMessage); detailErr == nil {
				failure.AddDetail(detail)
			}
		}
		return nil, failure
	}
	return connect.NewResponse(status), nil
}
func (h *Handler) GetSetupStatus(ctx context.Context, _ *connect.Request[setupv1.GetSetupStatusRequest]) (*connect.Response[setupv1.LightingRuntimeStatus], error) {
	if err := authorize(ctx, "status"); err != nil {
		return nil, err
	}
	return response(h.Operations.Status(ctx))
}
func (h *Handler) SetupLighting(ctx context.Context, r *connect.Request[setupv1.SetupLightingRequest]) (*connect.Response[setupv1.LightingRuntimeStatus], error) {
	if err := authorize(ctx, "setup"); err != nil {
		return nil, err
	}
	return response(h.Operations.Setup(ctx, r.Msg.Selection))
}
func (h *Handler) EnableLighting(ctx context.Context, r *connect.Request[setupv1.EnableLightingRequest]) (*connect.Response[setupv1.LightingRuntimeStatus], error) {
	if err := authorize(ctx, "enable"); err != nil {
		return nil, err
	}
	return response(h.Operations.Enable(ctx, r.Msg.Selection))
}
func (h *Handler) DisableLighting(ctx context.Context, _ *connect.Request[setupv1.DisableLightingRequest]) (*connect.Response[setupv1.LightingRuntimeStatus], error) {
	if err := authorize(ctx, "disable"); err != nil {
		return nil, err
	}
	return response(h.Operations.Disable(ctx))
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "setup_devices", Path: setupconnect.SetupServiceGetSetupDevicesProcedure, Method: "POST", Summary: "Discover selectable lighting devices", Description: "Explicit metadata discovery without permission changes.", Category: "setup"},
	{ID: "setup_devices_plan", Path: setupconnect.SetupServicePlanLightingDevicesProcedure, Method: "POST", Summary: "Plan selected device access", Description: "Reviews the exact maintained permission plan for selected opaque device identities.", Category: "setup"},
	{ID: "setup_devices_apply", Path: setupconnect.SetupServiceApplyLightingDevicesProcedure, Method: "POST", Summary: "Apply reviewed selected device access", Description: "Applies an exact canonical plan or reports required native privilege without retaining a credential.", Category: "setup"},
	{ID: "setup_devices_rollback", Path: setupconnect.SetupServiceRollbackLightingDevicesProcedure, Method: "POST", Summary: "Roll back selected device access", Description: "Restores the exact prior maintained rule from the canonical permission receipt.", Category: "setup"},
	{ID: "setup_status", Path: setupconnect.SetupServiceGetSetupStatusProcedure, Method: "POST", Summary: "Read lighting setup state", Description: "Reads desired state without disabled hardware probes.", Category: "setup"},
	{ID: "setup_install", Path: setupconnect.SetupServiceSetupLightingProcedure, Method: "POST", Summary: "Set up an exact installation", Description: "Explicitly stages a governed installation or adopts a compatible existing executable.", Category: "setup"},
	{ID: "setup_enable", Path: setupconnect.SetupServiceEnableLightingProcedure, Method: "POST", Summary: "Enable lighting", Description: "Persists exact selected desired state and reconciles native owners.", Category: "setup"},
	{ID: "setup_disable", Path: setupconnect.SetupServiceDisableLightingProcedure, Method: "POST", Summary: "Disable lighting", Description: "Quiesces lighting before disabling the backend.", Category: "setup"},
}

func (h *Handler) GetSetupDevices(ctx context.Context, _ *connect.Request[setupv1.GetSetupDevicesRequest]) (*connect.Response[setupv1.LightingDeviceDiscovery], error) {
	if err := authorize(ctx, "devices"); err != nil {
		return nil, err
	}
	return response(h.Operations.Devices(ctx))
}
func (h *Handler) PlanLightingDevices(ctx context.Context, r *connect.Request[setupv1.PlanLightingDevicesRequest]) (*connect.Response[setupv1.LightingDevicePlan], error) {
	if err := authorize(ctx, "permissions.plan"); err != nil {
		return nil, err
	}
	return response(h.Operations.PlanDevices(ctx, r.Msg.DeviceIds))
}
func (h *Handler) ApplyLightingDevices(ctx context.Context, r *connect.Request[setupv1.ApplyLightingDevicesRequest]) (*connect.Response[setupv1.LightingDeviceAccessResult], error) {
	if err := authorize(ctx, "permissions.apply"); err != nil {
		return nil, err
	}
	return response(h.Operations.ApplyDevices(ctx, r.Msg.PlanId))
}
func (h *Handler) RollbackLightingDevices(ctx context.Context, r *connect.Request[setupv1.RollbackLightingDevicesRequest]) (*connect.Response[setupv1.LightingDeviceAccessResult], error) {
	if err := authorize(ctx, "permissions.rollback"); err != nil {
		return nil, err
	}
	return response(h.Operations.RollbackDevices(ctx, r.Msg.PlanId))
}
