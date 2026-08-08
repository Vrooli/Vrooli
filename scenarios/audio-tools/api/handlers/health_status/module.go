// Package health_status hosts the HealthStatusService Connect-RPC
// handler: a typed view of the in-process capabilities.Registry rollup
// (per-capability × per-provider availability). The existing REST
// /health endpoint stays an ops probe; consumers wanting more than a
// single liveness boolean read through this Connect surface.
package health_status

import (
	"net/http"

	"audio-tools/internal/capabilities"
	"audio-tools/internal/clock"
	"audio-tools/internal/logx"
	"audio-tools/internal/modulekit"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	hsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/health_status/health_status_v1connect"
)

// Deps wires the handler. All fields are required; production wires
// `clock.System{}` + `logx.Std{...}` in main.go / bootstrap, tests wire
// mocks.FakeClock / mocks.FakeLogger.
type Deps struct {
	Registry *capabilities.Registry
	Logger   logx.Logger
	Clock    clock.Clock
}

// Module returns the health_status module contribution.
func Module(d Deps) modulekit.Module {
	connectPath, h := hsconnect.NewHealthStatusServiceHandler(NewConnectHandler(d))
	return modulekit.Module{
		Name: "health_status",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: h})
			r.HandleFunc("/api/v1/capabilities/describe", describe(d.Registry)).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

func describe(registry *capabilities.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if registry == nil {
			http.Error(w, "capabilities registry not configured", http.StatusServiceUnavailable)
			return
		}
		// The registry description is the machine-readable contract consumed by
		// dependency conformance. It is deliberately served beside the existing
		// typed health RPC so operators and validators read the same source.
		data, err := registry.Describe(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}
}

// Schema reports the health_status module's SQL contribution. None: the
// rollup is computed in-memory from the capabilities.Registry.
func Schema() string { return "" }

// Endpoints lists the Connect procedures for codegen / docs.
var Endpoints = []modulekit.EndpointDescriptor{
	healthEndpoint("get_provider_health", "GetProviderHealth", "Return the cached per-capability provider-health rollup"),
	healthEndpoint("refresh_provider_health", "RefreshProviderHealth", "Re-run every provider checker and return the fresh rollup"),
	healthEndpoint("stream_provider_health", "StreamProviderHealth", "Server-streaming feed of provider-health rollups (one event per registry tick)"),
}

func healthEndpoint(id, method, summary string) modulekit.EndpointDescriptor {
	return modulekit.EndpointDescriptor{
		ID:       "health_status." + id,
		Path:     "/vrooli.audio_tools.v1.health_status.HealthStatusService/" + method,
		Method:   http.MethodPost,
		Summary:  summary,
		Category: "health_status",
	}
}
