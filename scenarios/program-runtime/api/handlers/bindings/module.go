package bindings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/discovery"
	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
	bindingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings/bindings_v1connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing/routing_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
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

// IntentBridge is the private kernel seam for semantic discovery. Search-hub
// remains optional, while the response is always joined against the local
// manifest-backed registry before it reaches a program.
func IntentBridge(registry *bindings.Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var request struct {
			Intent string `json:"intent"`
			Limit  int32  `json:"limit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeBridgeError(w, http.StatusBadRequest, fmt.Sprintf("decode intent request: %v", err))
			return
		}
		response, err := (&service{registry: registry}).resolveIntent(r.Context(), request.Intent, request.Limit)
		if err != nil {
			writeBridgeError(w, http.StatusBadRequest, err.Error())
			return
		}
		payload, err := protojson.Marshal(response)
		if err != nil {
			writeBridgeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	})
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
		if registry.IsInferenceBinding(request.BindingID) {
			if err := manager.EnsureInferenceAvailable(r.Context(), request.SessionID); err != nil {
				var exceeded *sessions.SpendExceededError
				if errors.As(err, &exceeded) {
					writeBridgeError(w, http.StatusTooManyRequests, err.Error())
					return
				}
				writeBridgeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		if _, governed := registry.Binding(request.BindingID); !governed {
			if unresolved, ok := refusals.(bindings.UnresolvedRecorder); ok {
				_ = unresolved.RecordUnresolved(r.Context(), request.SessionID, request.BindingID, time.Now().UTC())
			}
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
		if registry.IsInferenceBinding(request.BindingID) {
			input, output, cost, present := bindings.InferenceUsage(result)
			if present {
				if err := manager.RecordInferenceUsage(r.Context(), request.SessionID, cost, input+output); err != nil {
					var exceeded *sessions.SpendExceededError
					if errors.As(err, &exceeded) {
						writeBridgeError(w, http.StatusTooManyRequests, err.Error())
						return
					}
					writeBridgeError(w, http.StatusInternalServerError, err.Error())
					return
				}
			}
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
		cost, measured, note := programs.DelegationCharge(result)
		if err := manager.RecordDelegationUsage(r.Context(), request.SessionID, cost, measured, note); err != nil {
			var exceeded *sessions.SpendExceededError
			if errors.As(err, &exceeded) {
				writeBridgeError(w, http.StatusTooManyRequests, err.Error())
				return
			}
			writeBridgeError(w, http.StatusInternalServerError, err.Error())
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

func (s *service) ResolveIntent(ctx context.Context, req *connect.Request[bindingsv1.ResolveIntentRequest]) (*connect.Response[bindingsv1.ResolveIntentResponse], error) {
	response, err := s.resolveIntent(ctx, req.Msg.GetIntent(), req.Msg.GetLimit())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(response), nil
}

func (s *service) resolveIntent(ctx context.Context, intent string, limit int32) (*bindingsv1.ResolveIntentResponse, error) {
	if strings.TrimSpace(intent) == "" {
		return nil, fmt.Errorf("intent is required")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	base, discoveryErr := discovery.ResolveScenarioURLDefault(ctx, "search-hub")
	if discoveryErr == nil {
		client := routingconnect.NewRoutingServiceClient(http.DefaultClient, strings.TrimRight(base, "/"))
		search, err := client.Query(ctx, connect.NewRequest(&routingv1.QueryRequest{Query: intent, All: true, Limit: limit}))
		if err == nil {
			joined := joinSearchHits(s.registry, search.Msg.GetRanked(), search.Msg.GetGroups(), int(limit))
			if len(joined) > 0 {
				return &bindingsv1.ResolveIntentResponse{Bindings: joined, Reason: "search-hub semantic discovery joined to governed local bindings"}, nil
			}
			discoveryErr = fmt.Errorf("search-hub returned no governed binding matches")
		} else {
			discoveryErr = err
		}
	}
	local, reason := s.registry.ResolveByIntent(intent)
	if len(local) > int(limit) {
		local = local[:limit]
	}
	if discoveryErr != nil {
		reason = fmt.Sprintf("local registry fallback: %v; %s", discoveryErr, reason)
	}
	return &bindingsv1.ResolveIntentResponse{Bindings: local, Reason: reason, Fallback: true}, nil
}

func joinSearchHits(registry *bindings.Registry, ranked []*routingv1.SearchHit, groups []*routingv1.ProviderResultGroup, limit int) []*bindingsv1.Binding {
	seen := make(map[string]struct{})
	out := make([]*bindingsv1.Binding, 0, limit)
	consume := func(hit *routingv1.SearchHit) {
		if hit == nil || len(out) >= limit {
			return
		}
		path := strings.Trim(strings.TrimSpace(hit.GetPath()), "/")
		parts := strings.FieldsFunc(path, func(r rune) bool { return r == '/' || r == ' ' })
		if len(parts) < 3 {
			return
		}
		id := strings.Join(parts[len(parts)-3:], "/")
		if _, ok := seen[id]; ok {
			return
		}
		binding, ok := registry.Binding(id)
		if !ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, binding)
	}
	for _, hit := range ranked {
		consume(hit)
	}
	for _, group := range groups {
		for _, hit := range group.GetHits() {
			consume(hit)
		}
	}
	return out
}

func (s *service) ResolveActCells(ctx context.Context, req *connect.Request[bindingsv1.ResolveActCellsRequest]) (*connect.Response[bindingsv1.ResolveActCellsResponse], error) {
	if s.registry == nil {
		return nil, connect.NewError(connect.CodeUnavailable, context.Canceled)
	}
	verdicts := s.registry.ResolveActCells(ctx, req.Msg.GetCells())
	confidence := string(actspace.Confidence(verdicts))
	return connect.NewResponse(&bindingsv1.ResolveActCellsResponse{Cells: verdicts, AuditedCells: int32(len(verdicts)), TotalCells: int32(len(req.Msg.GetCells())), DenominatorConfidence: confidence}), nil
}

func (s *service) DoctorBindings(ctx context.Context, req *connect.Request[bindingsv1.DoctorBindingsRequest]) (*connect.Response[bindingsv1.DoctorBindingsResponse], error) {
	if s.registry == nil {
		return nil, connect.NewError(connect.CodeUnavailable, context.Canceled)
	}
	return connect.NewResponse(s.registry.DoctorContext(ctx, req.Msg.GetScenario())), nil
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
