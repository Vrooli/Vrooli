// Package health_status hosts the HealthStatusService Connect-RPC
// handler: a typed view of the in-process capabilities.Registry rollup
// (per-capability × per-provider availability). The existing REST
// /health endpoint stays an ops probe; consumers wanting more than a
// single liveness boolean read through this Connect surface.
package health_status

import (
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
		},
		Endpoints: Endpoints,
	}
}

// Schema reports the health_status module's SQL contribution. None: the
// rollup is computed in-memory from the capabilities.Registry.
func Schema() string { return "" }

// Endpoints lists the Connect procedures for codegen / docs.
var Endpoints = []modulekit.EndpointDescriptor{
	{
		ID:       "health_status.get_provider_health",
		Path:     "/vrooli.audio_tools.v1.health_status.HealthStatusService/GetProviderHealth",
		Method:   "POST",
		Summary:  "Return the cached per-capability provider-health rollup",
		Category: "health_status",
		CLIMapping: &modulekit.CLIMapping{
			Command: "audio-tools health show",
			Args:    []string{"[--json]"},
		},
	},
	{
		ID:       "health_status.refresh_provider_health",
		Path:     "/vrooli.audio_tools.v1.health_status.HealthStatusService/RefreshProviderHealth",
		Method:   "POST",
		Summary:  "Re-run every provider checker and return the fresh rollup",
		Category: "health_status",
		CLIMapping: &modulekit.CLIMapping{
			Command: "audio-tools health show",
			Args:    []string{"--refresh", "[--json]"},
		},
	},
	{
		ID:       "health_status.stream_provider_health",
		Path:     "/vrooli.audio_tools.v1.health_status.HealthStatusService/StreamProviderHealth",
		Method:   "POST",
		Summary:  "Server-streaming feed of provider-health rollups (one event per registry tick)",
		Category: "health_status",
		CLIMapping: &modulekit.CLIMapping{
			Command: "audio-tools health watch",
		},
	},
}
