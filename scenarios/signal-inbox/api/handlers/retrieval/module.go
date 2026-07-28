package retrieval

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	retrievalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/retrieval/retrieval_v1connect"
	"signal-inbox/internal/module"
	internal "signal-inbox/internal/retrieval"
)

func Module(service *internal.Service) module.Module {
	path, handler := retrievalconnect.NewRetrievalServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "retrieval", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}
func Schema() string { return internal.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "retrieval_search", Path: retrievalconnect.RetrievalServiceSearchProcedure, Method: "POST", Summary: "Search signals", Description: "Searches the full signal corpus; category and disposition only filter results and never index eligibility.", Category: "retrieval", Request: &module.Schema{Type: "SearchRequest"}, Response: &module.Schema{Type: "SearchResponse"}},
	{ID: "retrieval_ambient", Path: retrievalconnect.RetrievalServiceAmbientProcedure, Method: "POST", Summary: "Read ambient signals", Description: "Returns a budgeted unresolved view without changing corpus search eligibility.", Category: "retrieval", Request: &module.Schema{Type: "AmbientRequest"}, Response: &module.Schema{Type: "AmbientResponse"}},
}
