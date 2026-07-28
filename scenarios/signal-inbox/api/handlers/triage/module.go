package triage

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	triageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/triage/triage_v1connect"
	"signal-inbox/internal/module"
	internal "signal-inbox/internal/triage"
)

func Module(service *internal.Service) module.Module {
	path, handler := triageconnect.NewTriageServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "triage", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}

func Schema() string { return internal.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "triage_get", Path: triageconnect.TriageServiceGetTriageProcedure, Method: "POST", Summary: "Get triage record", Description: "Returns the mutable disposition and append-only annotation thread for a signal.", Category: "triage", Request: &module.Schema{Type: "GetTriageRequest"}, Response: &module.Schema{Type: "GetTriageResponse"}},
	{ID: "triage_set_disposition", Path: triageconnect.TriageServiceSetDispositionProcedure, Method: "POST", Summary: "Set disposition", Description: "Updates the signal's single disposition without changing its journal entry or search eligibility.", Category: "triage", Request: &module.Schema{Type: "SetDispositionRequest"}, Response: &module.Schema{Type: "SetDispositionResponse"}},
	{ID: "triage_add_annotation", Path: triageconnect.TriageServiceAddAnnotationProcedure, Method: "POST", Summary: "Add annotation", Description: "Appends an operator, agent, or system annotation and optional typed outcome link.", Category: "triage", Request: &module.Schema{Type: "AddAnnotationRequest"}, Response: &module.Schema{Type: "AddAnnotationResponse"}},
}
