package bindings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
	bindingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings/bindings_v1connect"
	"program-runtime/internal/actspace"
	"program-runtime/internal/bindings"
	"program-runtime/internal/module"
	"program-runtime/internal/programs"
	"program-runtime/internal/sessions"
)

type service struct {
	bindingsconnect.UnimplementedBindingRegistryServiceHandler
	registry *bindings.Registry
}

func Module(registry *bindings.Registry) module.Module {
	return module.Module{
		Name: "bindings",
		Mount: func(r *mux.Router) {
			path, handler := Handler(registry)
			r.PathPrefix(path).Handler(handler)
		},
		Endpoints: Endpoints,
	}
}

// Handler returns the generated Connect handler and mount path. The server
// package mounts it directly so the transport remains generated and typed.
func Handler(registry *bindings.Registry) (string, http.Handler) {
	return bindingsconnect.NewBindingRegistryServiceHandler(&service{registry: registry})
}

// Bridge is a private sidecar seam. It is not part of the public proto/CLI
// surface: the kernel can reach it only with a live session id, and the
// registry still performs all descriptor validation and governance checks.
func Bridge(registry *bindings.Registry, manager *sessions.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var request struct {
			SessionID string         `json:"session_id"`
			BindingID string         `json:"binding_id"`
			Args      map[string]any `json:"args"`
			Confirmed bool           `json:"confirmed"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeBridgeError(w, http.StatusBadRequest, fmt.Sprintf("decode binding request: %v", err))
			return
		}
		session, err := manager.Get(request.SessionID)
		if err != nil {
			writeBridgeError(w, http.StatusNotFound, err.Error())
			return
		}
		result, err := registry.Execute(r.Context(), request.BindingID, request.Args, mapKeys(session.Grants), request.Confirmed, &http.Client{Timeout: 10 * time.Second})
		if err != nil {
			writeBridgeError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	})
}

// AgentBridge is the private sidecar seam for delegated agent-manager work.
// It requires a live program session and returns only the explicit result
// projection collected by the typed delegator.
func AgentBridge(manager *sessions.Manager, delegator programs.Delegator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var request programs.DelegationRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeBridgeError(w, http.StatusBadRequest, fmt.Sprintf("decode delegation request: %v", err))
			return
		}
		if _, err := manager.Get(request.SessionID); err != nil {
			writeBridgeError(w, http.StatusNotFound, err.Error())
			return
		}
		if delegator == nil {
			writeBridgeError(w, http.StatusServiceUnavailable, "agent-manager delegation is not configured")
			return
		}
		result, err := delegator.Delegate(r.Context(), request)
		if err != nil {
			writeBridgeError(w, http.StatusBadGateway, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	})
}

func mapKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func writeBridgeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (s *service) ListBindings(_ context.Context, req *connect.Request[bindingsv1.ListBindingsRequest]) (*connect.Response[bindingsv1.ListBindingsResponse], error) {
	if s.registry == nil {
		return nil, connect.NewError(connect.CodeUnavailable, context.Canceled)
	}
	return connect.NewResponse(&bindingsv1.ListBindingsResponse{Bindings: s.registry.List(req.Msg.GetScenario(), req.Msg.GetGroup())}), nil
}

func (s *service) ListUnbound(_ context.Context, req *connect.Request[bindingsv1.ListUnboundRequest]) (*connect.Response[bindingsv1.ListUnboundResponse], error) {
	if s.registry == nil {
		return nil, connect.NewError(connect.CodeUnavailable, context.Canceled)
	}
	return connect.NewResponse(&bindingsv1.ListUnboundResponse{Capabilities: s.registry.Unbound(req.Msg.GetScenario())}), nil
}

func (s *service) ResolveActCells(ctx context.Context, req *connect.Request[bindingsv1.ResolveActCellsRequest]) (*connect.Response[bindingsv1.ResolveActCellsResponse], error) {
	if s.registry == nil {
		return nil, connect.NewError(connect.CodeUnavailable, context.Canceled)
	}
	verdicts := s.registry.ResolveActCells(ctx, req.Msg.GetCells())
	confidence := string(actspace.Confidence(verdicts))
	return connect.NewResponse(&bindingsv1.ResolveActCellsResponse{Cells: verdicts, AuditedCells: int32(len(verdicts)), TotalCells: int32(len(req.Msg.GetCells())), DenominatorConfidence: confidence}), nil
}

var _ bindingsconnect.BindingRegistryServiceHandler = (*service)(nil)
