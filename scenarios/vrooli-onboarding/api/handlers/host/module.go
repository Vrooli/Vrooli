package host

import (
	"connectrpc.com/connect"
	hostv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/host/hostv1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/authz"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/host"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"
)

func Module(service host.Service, targetInterceptors ...connect.Interceptor) module.Module {
	interceptors := []connect.Interceptor{authz.MutationInterceptor()}
	interceptors = append(interceptors, targetInterceptors...)
	path, handler := hostv1connect.NewHostServiceHandler(NewConnectHandler(service), connect.WithInterceptors(interceptors...))
	return module.Connect("host", path, handler, Endpoints)
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "host_requirements", Path: hostv1connect.HostServiceListHostRequirementsProcedure, Method: "POST", Summary: "List host requirements", Description: "Returns typed host tools and safeguards for a selected target.", Category: "host"},
	{ID: "host_facts", Path: hostv1connect.HostServiceGetHostFactsProcedure, Method: "POST", Summary: "Read host facts", Description: "Returns bounded, secret-free inventory facts for a selected target.", Category: "host"},
	{ID: "host_targets", Path: hostv1connect.HostServiceListTargetsProcedure, Method: "POST", Summary: "List onboarding targets", Description: "Returns the fleet target inventory and shared readiness checks.", Category: "host"},
	{ID: "host_patch_safeguard_config", Path: hostv1connect.HostServicePatchHostSafeguardConfigProcedure, Method: "POST", Summary: "Patch safeguard configuration", Description: "Commits one typed safeguard configuration value.", Category: "host"},
	{ID: "host_set_recipient", Path: hostv1connect.HostServiceSetNotificationRecipientProcedure, Method: "POST", Summary: "Set notification recipient", Description: "Commits the notification recipient for the selected host.", Category: "host"},
}
