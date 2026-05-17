// Package provider_lifecycle hosts the ProviderLifecycleService
// Connect-RPC handler. It wraps the in-process ResourceController seam
// (see internal/capabilities/lifecycle.go) so the UI and CLI can
// start/stop/restart local-tier providers and stream their logs without
// shelling out from the front end.
package provider_lifecycle

import (
	"log"

	"audio-tools/internal/capabilities"
	"audio-tools/internal/modulekit"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	plconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/provider_lifecycle/provider_lifecycle_v1connect"
)

// Deps wires the handler. Registry is required for the
// ListLocalProviders process_state derivation and for cache-busting
// after successful mutations. Controller is required for the actual
// lifecycle shell-outs; in production it is *capabilities.CLIController,
// in tests it is a recording fake.
type Deps struct {
	Registry   *capabilities.Registry
	Controller capabilities.ResourceController
	Logger     *log.Logger
}

// Module returns the provider_lifecycle module contribution.
func Module(d Deps) modulekit.Module {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	connectPath, h := plconnect.NewProviderLifecycleServiceHandler(NewConnectHandler(d))
	return modulekit.Module{
		Name: "provider_lifecycle",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: h})
		},
		Endpoints: Endpoints,
	}
}

// Schema reports the provider_lifecycle module's SQL contribution.
// None: every action is a passthrough to the local resource controller.
func Schema() string { return "" }

// Endpoints lists the Connect procedures for codegen / docs.
var Endpoints = []modulekit.EndpointDescriptor{
	{
		ID:       "provider_lifecycle.list_local_providers",
		Path:     "/vrooli.audio_tools.v1.provider_lifecycle.ProviderLifecycleService/ListLocalProviders",
		Method:   "POST",
		Summary:  "List local-tier providers, their process_state, and supported lifecycle actions",
		Category: "provider_lifecycle",
		CLIMapping: &modulekit.CLIMapping{
			Command: "audio-tools provider list",
		},
	},
	{
		ID:       "provider_lifecycle.start_provider",
		Path:     "/vrooli.audio_tools.v1.provider_lifecycle.ProviderLifecycleService/StartProvider",
		Method:   "POST",
		Summary:  "Start a local provider (wraps `vrooli resource start <slug>`); honors X-Dry-Run",
		Category: "provider_lifecycle",
		CLIMapping: &modulekit.CLIMapping{
			Command: "audio-tools provider start",
			Args:    []string{"<provider-id>", "[--dry-run]"},
		},
	},
	{
		ID:       "provider_lifecycle.stop_provider",
		Path:     "/vrooli.audio_tools.v1.provider_lifecycle.ProviderLifecycleService/StopProvider",
		Method:   "POST",
		Summary:  "Stop a local provider (wraps `vrooli resource stop <slug>`); honors X-Dry-Run",
		Category: "provider_lifecycle",
		CLIMapping: &modulekit.CLIMapping{
			Command: "audio-tools provider stop",
			Args:    []string{"<provider-id>", "[--dry-run]"},
		},
	},
	{
		ID:       "provider_lifecycle.restart_provider",
		Path:     "/vrooli.audio_tools.v1.provider_lifecycle.ProviderLifecycleService/RestartProvider",
		Method:   "POST",
		Summary:  "Restart a local provider (wraps `vrooli resource restart <slug>`); honors X-Dry-Run",
		Category: "provider_lifecycle",
		CLIMapping: &modulekit.CLIMapping{
			Command: "audio-tools provider restart",
			Args:    []string{"<provider-id>", "[--dry-run]"},
		},
	},
	{
		ID:       "provider_lifecycle.pull_model",
		Path:     "/vrooli.audio_tools.v1.provider_lifecycle.ProviderLifecycleService/PullModel",
		Method:   "POST",
		Summary:  "Pull a model on the ollama provider; honors X-Dry-Run",
		Category: "provider_lifecycle",
		CLIMapping: &modulekit.CLIMapping{
			Command: "audio-tools provider pull-model",
			Args:    []string{"<model-name>", "[--dry-run]"},
		},
	},
	{
		ID:       "provider_lifecycle.get_provider_logs",
		Path:     "/vrooli.audio_tools.v1.provider_lifecycle.ProviderLifecycleService/GetProviderLogs",
		Method:   "POST",
		Summary:  "Stream stdout/stderr lines from a local provider's backing resource",
		Category: "provider_lifecycle",
		CLIMapping: &modulekit.CLIMapping{
			Command: "audio-tools provider logs",
			Args:    []string{"<provider-id>", "[--follow]", "[--tail N]"},
		},
	},
}
