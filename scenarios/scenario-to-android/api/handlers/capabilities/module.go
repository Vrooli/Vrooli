package capabilities

import (
	"net/http"

	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"

	"scenario-to-android/internal/capabilities"
	"scenario-to-android/internal/module"

	"github.com/gorilla/mux"
)

func Module(registry *capabilities.Registry) module.Module {
	return module.Module{
		Name: "capabilities",
		Mount: func(r *mux.Router) {
			r.Handle("/api/v1/capabilities/describe", capabilityregistry.DescribeHandler(registry)).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}
