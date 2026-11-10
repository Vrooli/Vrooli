// Package scenarios wires the Connect-RPC ScenariosService: scenario
// index, per-scenario detail, plus the server-streaming
// GenerateScenarioArtifacts and unary ClearScenarioArtifacts that
// thread through *artifacts.Service.
package scenarios

import (
	"log"
	"net/http"
	"time"

	"flow-verifier/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/server"

	scenariosconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/scenarios/scenarios_v1connect"
)

// Module returns the scenarios domain's Connect-RPC contribution. The
// artifacts service is required for the scenario-level Generate/Clear
// RPCs; passing nil makes those RPCs return Internal at runtime.
func Module(scenariosSvc Service, artifactsSvc ArtifactsService) module.Module {
	return ModuleWithLogger(scenariosSvc, artifactsSvc, log.Default())
}

// ModuleWithLogger is the test-friendly variant.
func ModuleWithLogger(scenariosSvc Service, artifactsSvc ArtifactsService, logger *log.Logger) module.Module {
	handler := NewStreamHandler(StreamDeps{
		Scenarios: scenariosSvc,
		Artifacts: artifactsSvc,
		Logger:    logger,
	})
	path, h := scenariosconnect.NewScenariosServiceHandler(handler)
	operationHandler, operationErr := server.WithOperationBudget(h, 2*time.Minute)
	if operationErr != nil {
		operationHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "lifecycle inventory operation unavailable", http.StatusInternalServerError)
		})
	}
	return module.Module{
		Name: "scenarios",
		Mount: func(r *mux.Router) {
			serviceHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path == scenariosconnect.ScenariosServiceListLifecyclesProcedure {
					operationHandler.ServeHTTP(w, req)
					return
				}
				h.ServeHTTP(w, req)
			})
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: serviceHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — scenarios are filesystem-truth, no tables.
func Schema() string { return "" }
