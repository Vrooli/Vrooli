package capabilities

import (
	"connectrpc.com/connect"
	capabilitiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/capabilities/capabilitiesv1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/authz"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/capabilities"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"
)

func Module(service capabilities.Service, targetInterceptors ...connect.Interceptor) module.Module {
	interceptors := []connect.Interceptor{authz.MutationInterceptor()}
	interceptors = append(interceptors, targetInterceptors...)
	path, handler := capabilitiesconnect.NewCapabilitiesServiceHandler(NewConnectHandler(service), connect.WithInterceptors(interceptors...))
	return module.Connect("capabilities", path, handler, Endpoints)
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "capabilities_list", Path: capabilitiesconnect.CapabilitiesServiceListCapabilitiesProcedure, Method: "POST", Summary: "List operator capabilities", Description: "Returns provider-owned capability descriptors, inputs, status, remediation, and secret-free evidence.", Category: "capabilities"},
	{ID: "capabilities_status", Path: capabilitiesconnect.CapabilitiesServiceGetCapabilityStatusProcedure, Method: "POST", Summary: "Read operator capability status", Description: "Returns the current provider capability statuses without action execution.", Category: "capabilities"},
	{ID: "capabilities_preview", Path: capabilitiesconnect.CapabilitiesServicePreviewCapabilityProcedure, Method: "POST", Summary: "Preview a capability action", Description: "Validates typed inputs and returns the provider's declared mutations without applying them.", Category: "capabilities"},
	{ID: "capabilities_apply", Path: capabilitiesconnect.CapabilitiesServiceApplyCapabilityProcedure, Method: "POST", Summary: "Apply a confirmed capability action", Description: "Applies a confirmed provider capability and returns secret-free evidence.", Category: "capabilities"},
	{ID: "capabilities_verify", Path: capabilitiesconnect.CapabilitiesServiceVerifyCapabilityProcedure, Method: "POST", Summary: "Verify a capability", Description: "Runs the owner verification adapter for one target-bound credential context and returns secret-free evidence.", Category: "capabilities"},
}
