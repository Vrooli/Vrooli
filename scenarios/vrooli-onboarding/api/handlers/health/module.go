package health

import (
	"net/http"

	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"

	"github.com/gorilla/mux"
	apihealth "github.com/vrooli/api-core/health"
)

// Module owns the two operational health probes. They remain plain REST so
// lifecycle managers, load balancers, and curl can use them without a client.
func Module() module.Module {
	return module.Module{
		Name: "health",
		Mount: func(r *mux.Router) {
			h := Handler()
			r.HandleFunc("/health", h).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/health", h).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

func Handler() http.HandlerFunc { return apihealth.New("vrooli-onboarding").Version("2.0.0").Handler() }

func Schema() string { return "" }
