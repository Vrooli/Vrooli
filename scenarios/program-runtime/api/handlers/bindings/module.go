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

type conditionService struct {
	bindingsconnect.UnimplementedBindingConditionServiceHandler
	registry *bindings.Registry
}

func Module(registry *bindings.Registry) module.Module {
	return module.Module{
		Name: "bindings",
		Mount: func(r *mux.Router) {
			path, handler := Handler(registry)
			r.PathPrefix(path).Handler(handler)
			conditionPath, conditionHandler := ConditionHandler(registry)
			r.PathPrefix(conditionPath).Handler(conditionHandler)
		},
		Endpoints: Endpoints,
	}
}

// Handler returns the generated Connect handler and mount path. The server
// package mounts it directly so the transport remains generated and typed.
func Handler(registry *bindings.Registry) (string, http.Handler) {
	return bindingsconnect.NewBindingRegistryServiceHandler(&service{registry: registry})
}

func ConditionHandler(registry *bindings.Registry) (string, http.Handler) {
	return bindingsconnect.NewBindingConditionServiceHandler(&conditionService{registry: registry})
}

// Bridge is a private sidecar seam. It is not part of the public proto/CLI
// surface: the kernel can reach it only with a live session id, and the
// registry still performs all descriptor validation and governance checks.
func Bridge(registry *bindings.Registry, manager *sessions.Manager, refusalRecorders ...bindings.RefusalRecorder) http.Handler {
	var refusals bindings.RefusalRecorder
	if len(refusalRecorders) > 0 {
		refusals = refusalRecorders[0]
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var request struct {
			SessionID  string         `json:"session_id"`
			ProgramID  string         `json:"program_id"`
			Provenance string         `json:"provenance"`
			BindingID  string         `json:"binding_id"`
			Args       map[string]any `json:"args"`
			Confirmed  bool           `json:"confirmed"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeBridgeError(w, http.StatusBadRequest, fmt.Sprintf("decode binding request: %v", err))
			return
		}
		session, err := manager.Get(r.Context(), request.SessionID)
		if err != nil {
			writeBridgeError(w, http.StatusNotFound, err.Error())
			return
		}
		// A binding may be a bounded typed inference call. The registry's
		// provider role timeout is authoritative, so the bridge must exceed the
		// longest declared role rather than imposing a 10s transport ceiling.
		grants := mapKeys(session.Grants)
		if err := registry.Authorize(request.BindingID, grants, request.Confirmed); err != nil {
			registry.RecordInvocation(r.Context(), bindings.Invocation{BindingID: request.BindingID, SessionID: request.SessionID, ProgramID: request.ProgramID, Provenance: request.Provenance, Outcome: "refused", Reason: err.Error(), OccurredAt: time.Now().UTC()})
			if refusals != nil {
				_ = refusals.RecordRefusal(r.Context(), request.SessionID, request.BindingID, err.Error(), time.Now().UTC())
			}
			writeBridgeError(w, http.StatusBadRequest, err.Error())
			return
		}
		result, err := registry.Execute(r.Context(), request.BindingID, request.Args, grants, request.Confirmed, bindings.InvocationMetadata{SessionID: request.SessionID, ProgramID: request.ProgramID, Provenance: request.Provenance}, &http.Client{Timeout: 3 * time.Minute})
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
		if _, err := manager.Get(r.Context(), request.SessionID); err != nil {
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

func (s *service) DoctorBindings(_ context.Context, req *connect.Request[bindingsv1.DoctorBindingsRequest]) (*connect.Response[bindingsv1.DoctorBindingsResponse], error) {
	if s.registry == nil {
		return nil, connect.NewError(connect.CodeUnavailable, context.Canceled)
	}
	return connect.NewResponse(s.registry.Doctor(req.Msg.GetScenario())), nil
}

func (s *service) DescribeBinding(_ context.Context, req *connect.Request[bindingsv1.DescribeBindingRequest]) (*connect.Response[bindingsv1.DescribeBindingResponse], error) {
	if s.registry == nil {
		return nil, connect.NewError(connect.CodeUnavailable, context.Canceled)
	}
	response, err := s.registry.Describe(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(response), nil
}

var (
	_ bindingsconnect.BindingRegistryServiceHandler  = (*service)(nil)
	_ bindingsconnect.BindingConditionServiceHandler = (*conditionService)(nil)
)

func (s *conditionService) GetBindingCondition(ctx context.Context, req *connect.Request[bindingsv1.GetBindingConditionRequest]) (*connect.Response[bindingsv1.GetBindingConditionResponse], error) {
	window := time.Duration(req.Msg.GetWindowSeconds()) * time.Second
	response, err := s.registry.Conditions(ctx, req.Msg.GetBindingId(), req.Msg.GetScenario(), window)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(response), nil
}
