package apihttp

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/module"
	"github.com/vrooli/api-core/schedule"
	servercore "github.com/vrooli/api-core/server"
)

// NewModuleRouter mounts API modules and applies the supplied middleware to
// the resulting router. It is the shared route-composition path for scenario
// APIs that publish module routes; callers wrap it in their chosen recovery
// boundary.
func NewModuleRouter(modules []module.Module, middleware ...func(http.Handler) http.Handler) *mux.Router {
	router := mux.NewRouter()
	for _, wrap := range middleware {
		router.Use(wrap)
	}
	for _, feature := range modules {
		feature.Mount(router)
	}
	return router
}

// NewScenarioModuleHandler owns the standard request boundary for APIs that
// are composed from modules: request logging, route mounting and panic
// recovery. Scenario-specific lifecycle and dependency wiring remains with
// the scenario.
func NewScenarioModuleHandler(clock schedule.Clock, logger *log.Logger, modules ...module.Module) http.Handler {
	if clock == nil {
		panic("NewScenarioModuleHandler requires a clock")
	}
	if logger == nil {
		logger = log.Default()
	}
	return servercore.RecoveryMiddleware(logger)(NewModuleRouter(modules, RequestLoggingMiddleware(clock, logger)))
}
