package credentials

import (
	"connectrpc.com/connect"
	credentialconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/credentials/credentialsv1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/authz"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/credentials"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"
)

func Module(service credentials.Service, targetInterceptors ...connect.Interceptor) module.Module {
	interceptors := []connect.Interceptor{authz.MutationInterceptor()}
	interceptors = append(interceptors, targetInterceptors...)
	path, handler := credentialconnect.NewCredentialsServiceHandler(NewConnectHandler(service), connect.WithInterceptors(interceptors...))
	return module.Connect("credentials", path, handler, Endpoints)
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "credentials_list", Path: credentialconnect.CredentialsServiceListCredentialsProcedure, Method: "POST", Summary: "List credential descriptors", Description: "Returns scoped credential metadata immediately without credential values; readiness supplies bounded storage status.", Category: "credentials"},
	{ID: "credentials_provision", Path: credentialconnect.CredentialsServiceProvisionCredentialProcedure, Method: "POST", Summary: "Provision one credential", Description: "Stores a credential value supplied in the write-only request; the value is never returned.", Category: "credentials", Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Credential identity or value is missing"}, {Status: 401, Code: "unauthenticated", Description: "Verified operator authorization is required"}, {Status: 403, Code: "permission_denied", Description: "Onboarding write capability is required"}, {Status: 503, Code: "unavailable", Description: "Native credential authority could not provision the credential"}}},
	{ID: "credentials_diagnose", Path: credentialconnect.CredentialsServiceDiagnoseCredentialsProcedure, Method: "POST", Summary: "Diagnose the credential authority", Description: "Returns typed provider, inventory, and recovery metadata without credential values.", Category: "credentials"},
}
