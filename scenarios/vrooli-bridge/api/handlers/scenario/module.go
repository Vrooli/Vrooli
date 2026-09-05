package scenario

import (
	"net/http"

	"vrooli-bridge/internal/module"
	internal "vrooli-bridge/internal/scenario"

	"github.com/gorilla/mux"
)

func Module(svc internal.Service) module.Module {
	h := NewHandler(Deps{Service: svc})
	return module.Module{
		Name: "scenario-proxy",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/targets/{node}/scenarios/{scenario}/{procedure:.*}", h.Call).Methods(http.MethodPost)
		},
	}
}
