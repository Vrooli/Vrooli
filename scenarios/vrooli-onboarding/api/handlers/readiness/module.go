package readiness

import (
	"connectrpc.com/connect"
	readinessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/readiness/readinessv1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/authz"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/readiness"
)

func Module(service readiness.Service, targetInterceptors ...connect.Interceptor) module.Module {
	interceptors := []connect.Interceptor{authz.MutationInterceptor()}
	interceptors = append(interceptors, targetInterceptors...)
	path, handler := readinessconnect.NewReadinessServiceHandler(NewConnectHandler(service), connect.WithInterceptors(interceptors...))
	return module.Connect("readiness", path, handler, Endpoints)
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "readiness_get", Path: readinessconnect.ReadinessServiceGetReadinessProcedure, Method: "POST", Summary: "Read onboarding readiness", Description: "Returns the typed metadata-safe readiness verdict.", Category: "readiness"},
	{ID: "readiness_acknowledge", Path: readinessconnect.ReadinessServiceAcknowledgeDegradedReadinessProcedure, Method: "POST", Summary: "Acknowledge degraded readiness", Description: "Records acknowledgement for the exact current degraded readiness set.", Category: "readiness", Request: &module.Schema{Type: "object", Properties: map[string]string{"readiness_digest": "string (required)"}}, Errors: []module.ErrorDesc{{Status: 409, Code: "aborted", Description: "The digest does not identify the current degraded set"}}},
}
