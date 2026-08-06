package capabilities

import (
	"net/http"
	"program-runtime/internal/capabilities"
	"program-runtime/internal/module"

	"github.com/gorilla/mux"
)

func Module(registry *capabilities.Registry) module.Module {
	return module.Module{
		Name: "capabilities",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/capabilities/describe", func(w http.ResponseWriter, req *http.Request) {
				if registry == nil {
					http.Error(w, "capabilities registry is not configured", http.StatusServiceUnavailable)
					return
				}
				data, err := registry.Describe(req.Context())
				if err != nil {
					http.Error(w, err.Error(), http.StatusServiceUnavailable)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(data)
			}).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}
