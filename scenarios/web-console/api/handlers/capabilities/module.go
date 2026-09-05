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
//	RunAction — explicit user-initiated scenario lifecycle action for
//	            declared dependencies only.
package capabilities

import (
	"context"
	"log"
	"net/http"

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
	RunAction(ctx context.Context, req ActionRequest) (ActionResult, error)
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
	ID                     string
	Name                   string
	Description            string
	DependencyKind         string
	DependencySlug         string
	Features               []string
	Status                 string
	Message                string
	CheckedAt              string
	ReasonCode             string
	ActionKind             string
	ActionLabel            string
	OperatorCommand        string
	FeatureStatus          map[string]string
	FeatureReason          map[string]string
	FeatureOperatorCommand map[string]string
	ProviderStatus         map[string]string
	ProviderFeatures       map[string]string
}

type ActionRequest struct {
	CapabilityID string
	ActionKind   string
	TargetID     string
}

type ActionResult struct {
	Success      bool
	Status       string
	Message      string
	OperationID  string
	CapabilityID string
	ActionKind   string
	Snapshot     Snapshot
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
			r.HandleFunc("/api/v1/capabilities/describe", describe(svc)).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

type registryDescriber interface {
	Describe(context.Context) ([]byte, error)
}

func describe(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		describer, ok := svc.(registryDescriber)
		if !ok {
			http.Error(w, "capabilities registry description is unavailable", http.StatusServiceUnavailable)
			return
		}
		data, err := describer.Describe(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}
}
