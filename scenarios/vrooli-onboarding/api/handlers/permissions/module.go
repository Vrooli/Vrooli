package permissions

import (
	"connectrpc.com/connect"
	permissionsv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/permissions/permissionsv1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/authz"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"
)

func Module(service RegistryProvider, targetInterceptors ...connect.Interceptor) module.Module {
	interceptors := []connect.Interceptor{authz.MutationInterceptor()}
	interceptors = append(interceptors, targetInterceptors...)
	path, handler := permissionsv1connect.NewPermissionRegistryServiceHandler(NewConnectHandler(service), connect.WithInterceptors(interceptors...))
	return module.Connect("permissions", path, handler, Endpoints)
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "permission_registry_list", Path: permissionsv1connect.PermissionRegistryServiceListPermissionsProcedure, Method: "POST", Summary: "List onboarding permissions", Description: "Enumerates owner-defined permissions through the shared typed registry for the authenticated principal."},
	{ID: "permission_registry_read", Path: permissionsv1connect.PermissionRegistryServiceReadPermissionProcedure, Method: "POST", Summary: "Read an onboarding permission", Description: "Reads one permission through its owner authorization and target binding."},
	{ID: "permission_registry_write", Path: permissionsv1connect.PermissionRegistryServiceWritePermissionProcedure, Method: "POST", Summary: "Write an onboarding permission", Description: "Writes one permission through its owner authorization, revision precondition and idempotent receipt."},
}
