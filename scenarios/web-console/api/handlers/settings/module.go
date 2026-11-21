// Package settings is the HTTP-handler home for the settings domain.
// It exposes the generated Connect-RPC SettingsService (proto schema:
// packages/proto/schemas/web-console/v1/settings).
//
// RPCs (mounted at /vrooli.web_console.v1.settings.SettingsService/...):
//
//	GetSessionDefaults    — returns the current session defaults
//	UpdateSessionDefaults — sparse update of session defaults
//
// Today this surface covers session defaults only. Additional preference
// categories should be added as new RPCs on SettingsService (or as
// sibling proto files in the same package if the surface grows large).
package settings

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	settingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/settings/settings_v1connect"

	"web-console/internal/module"
)

// Service is the seam the Connect handler depends on. The concrete
// implementation lives in package main (the Server struct adapts its
// existing settings methods to satisfy this interface) — defining it
// here keeps the handler package importable from main without a
// circular dependency.
type Service interface {
	GetDefaults() Defaults
	UpdateDefaults(req UpdateDefaultsRequest) (Defaults, error)
}

// Defaults is the transport-neutral session-defaults shape. Mirrors
// the proto SessionDefaults message field-for-field.
type Defaults struct {
	DefaultBackend string
	DefaultPolicy  Policy
}

// Policy mirrors the proto ExpirationPolicy message. Validation rules
// (allowed modes, duration formats) live in the Service implementation.
type Policy struct {
	Mode     string
	Duration string
}

// UpdateDefaultsRequest is a sparse update: only fields the caller
// explicitly sets are applied. Pointer fields convey "not set" so the
// service distinguishes absent from zero-value.
type UpdateDefaultsRequest struct {
	DefaultBackend *string
	DefaultPolicy  *Policy
}

// Module wires the settings domain into the API server.
func Module(svc Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := settingsconnect.NewSettingsServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "settings",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
