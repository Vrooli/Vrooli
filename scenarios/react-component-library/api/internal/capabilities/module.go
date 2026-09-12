package capabilities

import (
	"net/http"

	"react-component-library/internal/module"

	"github.com/gorilla/mux"
)

// Module exposes the machine-readable capability registry used by dependency
// conformance and operator-facing integration surfaces.
func Module() module.Module {
	registry := NewRegistry()
	return module.Module{
		Name: "capabilities",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/capabilities/describe", func(w http.ResponseWriter, req *http.Request) {
				data, err := registry.Describe(req.Context())
				if err != nil {
					http.Error(w, err.Error(), http.StatusServiceUnavailable)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(data)
			}).Methods(http.MethodGet)
		},
	}
}
