package models

import (
	"audio-tools/internal/modulekit"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	modelsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/models/models_v1connect"
)

// Module mounts the model registry service. It has no database schema: the
// registry is a validated embedded seed.
func Module(d Deps) modulekit.Module {
	connectPath, h := modelsconnect.NewModelRegistryServiceHandler(NewConnectHandler(d))
	return modulekit.Module{
		Name: "models",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: h})
		},
		Endpoints: Endpoints,
	}
}

// Schema is empty: the registry is static data, not durable state.
func Schema() string { return "" }

var Endpoints = []modulekit.EndpointDescriptor{
	{
		ID: "models.list", Path: "/vrooli.audio_tools.v1.models.ModelRegistryService/ListModels",
		Method: "POST", Summary: "List the validated model/voice registry with readiness, licence, and hardware facts",
		Category: "models",
	},
	{
		ID: "models.resolve", Path: "/vrooli.audio_tools.v1.models.ModelRegistryService/ResolveModel",
		Method: "POST", Summary: "Resolve the deterministic default model for a capability and language (licence-fail-closed)",
		Category: "models",
	},
}
