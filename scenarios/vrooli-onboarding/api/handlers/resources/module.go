package resources

import (
	"connectrpc.com/connect"
	resourcesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/resources/resourcesv1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"
	internalresources "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/resources"
)

func Module(service internalresources.Service, targetInterceptors ...connect.Interceptor) module.Module {
	path, handler := resourcesconnect.NewResourcesServiceHandler(NewConnectHandler(service), connect.WithInterceptors(targetInterceptors...))
	return module.Connect("resources", path, handler, Endpoints)
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "resources_list", Path: resourcesconnect.ResourcesServiceListResourcesProcedure, Method: "POST", Summary: "List onboarding resources", Category: "resources"},
	{ID: "resources_get", Path: resourcesconnect.ResourcesServiceGetResourceProcedure, Method: "POST", Summary: "Read one onboarding resource", Category: "resources"},
	{ID: "resources_health", Path: resourcesconnect.ResourcesServiceGetResourceHealthProcedure, Method: "POST", Summary: "Read onboarding resource health", Category: "resources"},
	{ID: "resources_derived", Path: resourcesconnect.ResourcesServiceListDerivedResourcesProcedure, Method: "POST", Summary: "List derived onboarding resources", Category: "resources"},
}
