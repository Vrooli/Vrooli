package search

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/search/search_v1connect"

	"portal/internal/module"
	internalsearch "portal/internal/search"
)

func Module(service *internalsearch.Service) module.Module {
	connectPath, connectHandler := searchconnect.NewSearchServiceHandler(NewHandler(service))
	return module.Module{
		Name: "search",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
