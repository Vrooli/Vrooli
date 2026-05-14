// Package capabilities is the HTTP-handler home for the capabilities
// domain. It exposes the generated Connect-RPC CapabilitiesService
// (proto schema: packages/proto/schemas/web-console/v1/capabilities).
//
// RPCs (mounted at /vrooli.web_console.v1.capabilities.CapabilitiesService/...):
//
//	Get      — full capability snapshot. Includes session backends and
//	           the active default backend.
//	Liveness — fast probe. Uses cached full-check results when fresh,
//	           otherwise lightweight health checkers only.
package capabilities

import (
	"context"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	capabilitiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/capabilities/capabilities_v1connect"

	"web-console/internal/module"
)

// Service is the seam the Connect handler depends on. Implemented in
// package main by capabilitiesAdapter, which bridges to the existing
// CapabilityRegistry and BackendRegistry.
type Service interface {
	Resolve(ctx context.Context) Snapshot
	Liveness(ctx context.Context) Snapshot
}

// Snapshot is the transport-neutral capabilities view. BackendOptions
// and DefaultBackend are zero-valued for liveness probes.
type Snapshot struct {
	Capabilities   []CapabilityState
	Timestamp      string
	BackendOptions []BackendOption
	DefaultBackend string
}

// CapabilityState mirrors the proto CapabilityState message.
type CapabilityState struct {
	ID             string
	Name           string
	Description    string
	DependencyKind string
	DependencySlug string
	Features       []string
	Status         string
	Message        string
	CheckedAt      string
}

// BackendOption mirrors the proto BackendOption message and the
// api.BackendDescriptor struct.
type BackendOption struct {
	ID              string
	DisplayName     string
	Description     string
	SurvivesRestart bool
	Available       bool
	Reason          string
}

// Module wires the capabilities domain into the API server.
func Module(svc Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := capabilitiesconnect.NewCapabilitiesServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "capabilities",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
