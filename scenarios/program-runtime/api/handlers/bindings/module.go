package bindings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/spacedoc"
	"github.com/vrooli/repo-contract-go"
	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
	bindingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings/bindings_v1connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing/routing_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"program-runtime/internal/actspace"
	"program-runtime/internal/bindings"
	"program-runtime/internal/budgets"
	"program-runtime/internal/library"
	"program-runtime/internal/module"
	"program-runtime/internal/programs"
	"program-runtime/internal/sessions"
)

const (
	maxDeepParaphrases = 3
	maxJudgeCandidates = 8
)

type service struct {
	bindingsconnect.UnimplementedBindingRegistryServiceHandler
	registry     *bindings.Registry
	library      *library.Repository
	actSpacePath string
	metrics      *DiscoveryMetrics
}

// DiscoveryMetrics is the process-local owner for the two measures exposed by
// this scenario. It deliberately counts only typed discovery outcomes; raw
// Search Hub queries are not library usage.
type DiscoveryMetrics struct {
	discoveryCalls atomic.Int64
	nullVerdicts   atomic.Int64
	libraryHits    atomic.Int64
}

func (m *DiscoveryMetrics) record(response *bindingsv1.ResolveIntentResponse) {
	if m == nil || response == nil {
		return
	}
	m.discoveryCalls.Add(1)
	if result := response.GetResult(); result != nil && result.GetLibrary() != nil {
		m.libraryHits.Add(1)
	}
	if result := response.GetResult(); result == nil || (result.GetBindingId() == "" && result.GetLibrary() == nil) {
		m.nullVerdicts.Add(1)
	}
}

func (m *DiscoveryMetrics) LibraryUsage() int { return int(m.libraryHits.Load()) }

func (m *DiscoveryMetrics) NullVerdictRatePercent() int {
	calls := m.discoveryCalls.Load()
	if calls == 0 {
		return 0
	}
	return int((m.nullVerdicts.Load() * 100) / calls)
}

func LibraryDiscoveryUsage() int { return discoveryMetrics.LibraryUsage() }

func DiscoveryNullVerdictRatePercent() int { return discoveryMetrics.NullVerdictRatePercent() }

var discoveryMetrics DiscoveryMetrics

type conditionService struct {
	bindingsconnect.UnimplementedBindingConditionServiceHandler
	registry *bindings.Registry
}

func Module(registry *bindings.Registry, repositories ...*library.Repository) module.Module {
	var libraryRepository *library.Repository
	if len(repositories) > 0 {
		libraryRepository = repositories[0]
	}
	return module.Module{
		Name: "bindings",
		Mount: func(r *mux.Router) {
			path, handler := Handler(registry, libraryRepository)
			r.PathPrefix(path).Handler(handler)
			conditionPath, conditionHandler := ConditionHandler(registry)
			r.PathPrefix(conditionPath).Handler(conditionHandler)
		},
		Endpoints: Endpoints,
	}
}

// Handler returns the generated Connect handler and mount path. The server
// package mounts it directly so the transport remains generated and typed.
func Handler(registry *bindings.Registry, libraryRepository ...*library.Repository) (string, http.Handler) {
	var repository *library.Repository
	if len(libraryRepository) > 0 {
		repository = libraryRepository[0]
	}
	return bindingsconnect.NewBindingRegistryServiceHandler(&service{registry: registry, library: repository, metrics: &discoveryMetrics})
}

func ConditionHandler(registry *bindings.Registry) (string, http.Handler) {
	return bindingsconnect.NewBindingConditionServiceHandler(&conditionService{registry: registry})
}

// IntentBridge is the private kernel seam for semantic discovery. Search-hub
// remains optional, while the response is always joined against the local
// manifest-backed registry before it reaches a program.
func IntentBridge(registry *bindings.Registry, libraryRepository ...*library.Repository) http.Handler {
	var repository *library.Repository
	if len(libraryRepository) > 0 {
		repository = libraryRepository[0]
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var request struct {
			Intent string `json:"intent"`
			Limit  int32  `json:"limit"`
			Mode   string `json:"mode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeBridgeError(w, http.StatusBadRequest, fmt.Sprintf("decode intent request: %v", err))
			return
		}
		owner := &service{registry: registry, library: repository, metrics: &discoveryMetrics}
		response, err := owner.resolveIntent(r.Context(), request.Intent, request.Limit, request.Mode)
		owner.metrics.record(response)
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

// DescribeBridge is the private kernel seam for live binding contracts. The
// registry remains the only owner of descriptor resolution and validation;
// this handler only authenticates the live session and marshals its response.
func DescribeBridge(registry *bindings.Registry, manager *sessions.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var request struct {
			SessionID string `json:"session_id"`
			Binding   string `json:"binding"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeBridgeError(w, http.StatusBadRequest, fmt.Sprintf("decode binding description request: %v", err))
			return
		}
		if _, err := manager.Get(r.Context(), request.SessionID); err != nil {
			writeBridgeError(w, http.StatusNotFound, err.Error())
			return
		}
		id, err := resolveDescriptionID(registry, request.Binding)
		if err != nil {
			writeBridgeError(w, http.StatusBadRequest, err.Error())
			return
		}
		response, err := registry.Describe(id)
		if err != nil {
			writeBridgeError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			return
		}
	})
}

// ReachabilityBridge exposes the same TTL-backed scenario census used by
// governed calls. It is private and session-authenticated because it is a
// kernel control surface, not a public fleet health endpoint.
func ReachabilityBridge(registry *bindings.Registry, manager *sessions.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var request struct {
			SessionID string `json:"session_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeBridgeError(w, http.StatusBadRequest, fmt.Sprintf("decode reachability request: %v", err))
			return
		}
		if _, err := manager.Get(r.Context(), request.SessionID); err != nil {
			writeBridgeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(registry.ReachabilitySnapshot(r.Context()))
	})
}

// UnresolvedBridge records runtime name misses produced by dynamic Python
// lookups (the normal static path is handled by program preflight). It is a
// best-effort telemetry seam: the kernel still owns the user-visible
// NameError, while this handler owns durable unresolved-name evidence.
func UnresolvedBridge(manager *sessions.Manager, recorder bindings.UnresolvedRecorder) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var request struct {
			SessionID     string `json:"session_id"`
			Provenance    string `json:"provenance"`
			AttemptedName string `json:"attempted_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeBridgeError(w, http.StatusBadRequest, fmt.Sprintf("decode unresolved-name request: %v", err))
			return
		}
		if strings.TrimSpace(request.AttemptedName) == "" {
			writeBridgeError(w, http.StatusBadRequest, "attempted_name is required")
			return
		}
		if _, err := manager.Get(r.Context(), request.SessionID); err != nil {
			writeBridgeError(w, http.StatusNotFound, err.Error())
			return
		}
		if recorder != nil {
			if err := recorder.RecordUnresolved(r.Context(), request.SessionID, request.Provenance, request.AttemptedName, time.Now().UTC()); err != nil {
				writeBridgeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func resolveDescriptionID(registry *bindings.Registry, reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	if strings.HasPrefix(reference, "vrooli.") {
		reference = strings.TrimPrefix(reference, "vrooli.")
	}
	pathReference := strings.ReplaceAll(reference, ".", "/")
	for _, candidate := range registry.List("", "") {
		id := candidate.GetId()
		if reference == id || pathReference == id || strings.ReplaceAll(pathReference, "-", "_") == strings.ReplaceAll(id, "-", "_") {
			return id, nil
		}
	}
	return "", fmt.Errorf("binding %q is not governed; use a binding id or qualified path", reference)
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
			Rows       string         `json:"rows"`
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
				_ = unresolved.RecordUnresolved(r.Context(), request.SessionID, request.Provenance, request.BindingID, time.Now().UTC())
			}
		}
		// A binding may be a bounded typed inference call. The registry's
		// provider role timeout is authoritative, so the bridge must exceed the
		// longest declared role rather than imposing a 10s transport ceiling.
		grants := mapKeys(session.Grants)
		if err := registry.Authorize(request.BindingID, grants, request.Confirmed); err != nil {
			registry.RecordInvocation(r.Context(), bindings.Invocation{BindingID: request.BindingID, SessionID: request.SessionID, ProgramID: request.ProgramID, Provenance: request.Provenance, Outcome: "refused", Reason: err.Error(), OccurredAt: time.Now().UTC()})
			if refusals != nil {
				_ = refusals.RecordRefusal(r.Context(), request.SessionID, request.Provenance, request.BindingID, err.Error(), time.Now().UTC())
			}
			writeBridgeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if request.Rows != "" {
			binding, _ := registry.Binding(request.BindingID)
			valid := false
			if binding != nil {
				for _, candidate := range binding.GetRowFieldCandidates() {
					if candidate == request.Rows {
						valid = true
						break
					}
				}
			}
			if !valid {
				writeBridgeError(w, http.StatusBadRequest, fmt.Sprintf("binding %s rows must name one of its repeated response fields", request.BindingID))
				return
			}
		}
		result, err := registry.Execute(r.Context(), request.BindingID, request.Args, grants, request.Confirmed, bindings.InvocationMetadata{SessionID: request.SessionID, ProgramID: request.ProgramID, Provenance: request.Provenance}, &http.Client{Timeout: budgets.BridgeCall})
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

// projectionVerbs maps each runtime-owned verb to the governed binding that
// serves it. The verbs are compositions over the same registry a program calls
// directly — not a second outbound path — so they inherit governance, argument
// validation, reachability, and invocation recording rather than re-declaring
// them. A verb whose owner has no governed binding fails closed and names the
// owner, because inventing an ungoverned client for it would put an
// unvalidated call behind a surface that advertises the opposite.
var projectionVerbs = map[string]projectionVerb{
	"recall": {
		binding: "search-hub/query/query",
		owner:   "search-hub",
		rows:    "ranked",
		build: func(request projectionRequest) (map[string]any, error) {
			intent := request.first("intent", "query", "text")
			if intent == "" {
				return nil, errors.New("recall requires a non-empty intent")
			}
			args := map[string]any{"query": intent}
			if request.first("depth") == "deep" {
				args["limit"] = 30
			} else {
				args["limit"] = 10
			}
			return args, nil
		},
	},
	// `validate` reads the latest recorded verdicts rather than starting a run.
	// Starting a run is a write that test-genie exposes as run-ineligible, so no
	// governed binding exists for it and inventing one here would put an
	// unvalidated mutation behind a read-shaped verb. Programs start runs
	// through the lifecycle (`vrooli scenario test`); this verb answers "what
	// does validation currently say".
	"validate": {
		binding: "test-genie/runs/list",
		owner:   "test-genie",
		rows:    "runs",
		build: func(request projectionRequest) (map[string]any, error) {
			scenario := request.first("scenario", "target", "name")
			if scenario == "" {
				return nil, errors.New("validate requires a scenario name")
			}
			args := map[string]any{"scenario": scenario}
			// An absent `status` must not become a filter. Reading it with
			// fmt.Sprint on a missing map key yielded the literal string
			// "<nil>", which test-genie matched against zero runs — so this
			// verb reported SUCCEEDED with an empty result for every scenario.
			// projectionRequest.first cannot produce that value.
			if status := request.first("status"); status != "" {
				args["status"] = status
			}
			if request.first("depth") == "deep" {
				args["limit"] = 20
			} else {
				args["limit"] = 5
			}
			return args, nil
		},
	},
	"capture": {
		binding: "vrooli-memory/journal/note",
		owner:   "vrooli-memory",
		rows:    projectionWholeResponse,
		build: func(request projectionRequest) (map[string]any, error) {
			body := request.first("text", "note", "body")
			if body == "" {
				return nil, errors.New("capture requires non-empty text")
			}
			// Same defect, opposite symptom: an absent `kind` stringified to
			// "<nil>", which is non-empty, so the documented "note" default was
			// unreachable and every caller that omitted the field was refused.
			kind := request.first("kind")
			if kind == "" {
				kind = "note"
			}
			if kind != "note" && kind != "work-record" {
				return nil, fmt.Errorf("capture kind %q is not accepted; use \"note\" or \"work-record\"", kind)
			}
			args := map[string]any{"body": body, "kind": kind}
			// A work record carries the four fields the learning loop expects.
			// They are optional on the binding, so a caller that omits them
			// still writes a valid record rather than being refused.
			for _, field := range []string{"scope", "trigger", "approach", "evidence", "outcome"} {
				if value := request.first(field); value != "" {
					args[field] = value
				}
			}
			return args, nil
		},
	},
	// `guide` has no governed binding: prompt-manager ships no resolved
	// manifest binding in the live registry, so there is nothing typed to
	// compose. It stays declared and fails closed with that reason rather than
	// disappearing, so the gap is visible to the agent and to the unbound
	// census instead of silently absent.
	"guide": {owner: "prompt-manager", rows: "results"},
}

// projectionWholeResponse is the explicit row projection for operations whose
// response has no repeated field. Such operations still return a one-row
// Handle; the marker keeps that decision declared beside every other verb.
const projectionWholeResponse = "$response"

type projectionVerb struct {
	binding string
	owner   string
	rows    string
	build   func(projectionRequest) (map[string]any, error)
}

func projectionRowFields(binding *bindingsv1.Binding) []string {
	if binding == nil {
		return nil
	}
	fields := append([]string(nil), binding.GetRowFieldCandidates()...)
	if selected := strings.TrimSpace(binding.GetRowsField()); selected != "" {
		fields = append(fields, selected)
	}
	sort.Strings(fields)
	return slices.Compact(fields)
}

func validateProjectionRows(binding *bindingsv1.Binding, rows string) error {
	if rows == "" || rows == projectionWholeResponse {
		return nil
	}
	fields := projectionRowFields(binding)
	if slices.Contains(fields, rows) {
		return nil
	}
	available := strings.Join(fields, ", ")
	if available == "" {
		available = "<none>"
	}
	return fmt.Errorf("projection rows %q is not a repeated response field; available fields: %s", rows, available)
}

// projectionRequest is the decoded verb payload. The control fields are typed;
// everything else stays in Extra because the verb vocabulary is open.
//
// It exists because the previous shape — a bare map[string]any read with
// fmt.Sprint — silently converted every *absent* key into the four-character
// string "<nil>". That is a non-empty value, so it defeated every `!= ""`
// guard in this file: `validate` sent it as a status filter and matched zero
// runs, and `capture` sent it as a kind and refused its own default. Reading a
// missing key must produce the empty string, and the only reliable way to
// guarantee that is to stop calling fmt.Sprint on map lookups.
type projectionRequest struct {
	SessionID  string         `json:"session_id"`
	ProgramID  string         `json:"program_id"`
	Provenance string         `json:"provenance"`
	Extra      map[string]any `json:"-"`
}

// first returns the first key that carries a non-empty value, or "" when none
// does. A JSON null, a missing key, and an empty string are all "absent"; no
// input can make this return a rendering of nil.
func (p projectionRequest) first(keys ...string) string {
	for _, key := range keys {
		value, present := p.Extra[key]
		if !present || value == nil {
			continue
		}
		if text, ok := value.(string); ok {
			if trimmed := strings.TrimSpace(text); trimmed != "" {
				return trimmed
			}
			continue
		}
		if trimmed := strings.TrimSpace(fmt.Sprint(value)); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// decodeProjectionRequest reads the control fields into typed columns and
// retains the remainder for the verb builders.
func decodeProjectionRequest(body io.Reader) (projectionRequest, error) {
	var raw map[string]any
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return projectionRequest{}, err
	}
	request := projectionRequest{Extra: raw}
	request.SessionID = request.first("session_id")
	request.ProgramID = request.first("program_id")
	request.Provenance = request.first("provenance")
	return request, nil
}

// ProjectionBridge exposes runtime-owned projection verbs as typed, immutable
// Go endpoints rather than mutable library rows.
func ProjectionBridge(manager *sessions.Manager, registry *bindings.Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// The handler is mounted under a scenario-internal prefix, so the verb
		// is the final path segment. Trimming a bare "/projection/" prefix left
		// the whole path in `verb` and made every call a 404.
		verb := path.Base(r.URL.Path)
		spec, known := projectionVerbs[verb]
		if !known {
			writeBridgeError(w, http.StatusNotFound, fmt.Sprintf("unknown projection verb %q", verb))
			return
		}
		request, err := decodeProjectionRequest(r.Body)
		if err != nil {
			writeBridgeError(w, http.StatusBadRequest, fmt.Sprintf("decode projection request: %v", err))
			return
		}
		if manager != nil {
			if _, err := manager.Get(r.Context(), request.SessionID); err != nil {
				writeBridgeError(w, http.StatusNotFound, err.Error())
				return
			}
		}
		if spec.binding == "" {
			writeBridgeError(w, http.StatusServiceUnavailable, fmt.Sprintf("projection %q is unavailable: %s exposes no governed binding in the live registry", verb, spec.owner))
			return
		}
		if registry == nil {
			writeBridgeError(w, http.StatusServiceUnavailable, fmt.Sprintf("projection %q is unavailable: binding registry is not configured", verb))
			return
		}
		binding, governed := registry.Binding(spec.binding)
		if !governed {
			writeBridgeError(w, http.StatusServiceUnavailable, fmt.Sprintf("projection %q is unavailable: binding %q is not governed in the live registry", verb, spec.binding))
			return
		}
		rowsField := spec.rows
		if override := request.first("rows"); override != "" {
			rowsField = override
		}
		if err := validateProjectionRows(binding, rowsField); err != nil {
			writeBridgeError(w, http.StatusBadRequest, err.Error())
			return
		}
		args, err := spec.build(request)
		if err != nil {
			writeBridgeError(w, http.StatusBadRequest, err.Error())
			return
		}
		result, err := registry.Execute(r.Context(), spec.binding, args, nil, false,
			bindings.InvocationMetadata{SessionID: request.SessionID, ProgramID: request.ProgramID, Provenance: request.Provenance},
			&http.Client{Timeout: budgets.BridgeCall})
		if err != nil {
			writeBridgeError(w, http.StatusBadGateway, fmt.Sprintf("projection %q is unavailable: %v", verb, err))
			return
		}
		response := map[string]any{"verb": verb, "binding_id": spec.binding, "owner": spec.owner, "result": result}
		if rowsField != "" && rowsField != projectionWholeResponse {
			response["rows_field"] = rowsField
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
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

// AgentStartBridge launches delegated work without waiting for terminal state.
// The execution identity is persisted before it is returned to the kernel.
func AgentStartBridge(manager *sessions.Manager, delegator programs.Delegator) http.Handler {
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
		asyncDelegator, ok := delegator.(programs.AsyncDelegator)
		if !ok {
			writeBridgeError(w, http.StatusServiceUnavailable, "asynchronous agent-manager delegation is not configured")
			return
		}
		result, err := asyncDelegator.Start(r.Context(), request)
		if err != nil {
			writeBridgeError(w, http.StatusBadGateway, err.Error())
			return
		}
		executionID, _ := result["execution_id"].(string)
		if err := manager.SaveDelegation(r.Context(), &sessions.Delegation{SessionID: request.SessionID, ExecutionID: executionID, Owner: request.Owner, WorkflowKey: request.WorkflowKey, CreatedAt: time.Now().UTC(), LastStatus: fmt.Sprint(result["status"])}); err != nil {
			writeBridgeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	})
}

// AgentCollectBridge enforces session ownership before collecting a bounded
// delegated result. A wait of zero is a non-blocking status/result read.
func AgentCollectBridge(manager *sessions.Manager, delegator programs.Delegator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var request struct {
			SessionID   string `json:"session_id"`
			ExecutionID string `json:"execution_id"`
			WaitSeconds int    `json:"wait_seconds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeBridgeError(w, http.StatusBadRequest, fmt.Sprintf("decode collect request: %v", err))
			return
		}
		if _, err := manager.GetDelegation(r.Context(), request.SessionID, request.ExecutionID); err != nil {
			status := http.StatusNotFound
			if errors.Is(err, sessions.ErrDelegationNotOwned) {
				status = http.StatusForbidden
			}
			writeBridgeError(w, status, err.Error())
			return
		}
		asyncDelegator, ok := delegator.(programs.AsyncDelegator)
		if !ok {
			writeBridgeError(w, http.StatusServiceUnavailable, "asynchronous agent-manager delegation is not configured")
			return
		}
		result, err := asyncDelegator.Collect(r.Context(), request.SessionID, request.ExecutionID, request.WaitSeconds)
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

func (s *service) ListBindings(ctx context.Context, req *connect.Request[bindingsv1.ListBindingsRequest]) (*connect.Response[bindingsv1.ListBindingsResponse], error) {
	if s.registry == nil {
		return nil, connect.NewError(connect.CodeUnavailable, context.Canceled)
	}
	bindings := s.registry.ListContext(ctx, req.Msg.GetScenario(), req.Msg.GetGroup(), req.Msg.GetReachableOnly())
	checkedAt := s.registry.ReachabilityCheckedAt(ctx, req.Msg.GetScenario())
	return connect.NewResponse(&bindingsv1.ListBindingsResponse{Bindings: bindings, ReachabilityCheckedAt: checkedAt.UTC().Format(time.RFC3339Nano)}), nil
}

func (s *service) ListUnbound(_ context.Context, req *connect.Request[bindingsv1.ListUnboundRequest]) (*connect.Response[bindingsv1.ListUnboundResponse], error) {
	if s.registry == nil {
		return nil, connect.NewError(connect.CodeUnavailable, context.Canceled)
	}
	return connect.NewResponse(&bindingsv1.ListUnboundResponse{Capabilities: s.registry.Unbound(req.Msg.GetScenario())}), nil
}

func (s *service) SweepBindings(ctx context.Context, req *connect.Request[bindingsv1.SweepBindingsRequest]) (*connect.Response[bindingsv1.SweepBindingsResponse], error) {
	if s.registry == nil {
		return nil, connect.NewError(connect.CodeUnavailable, context.Canceled)
	}
	return connect.NewResponse(s.registry.Sweep(ctx, req.Msg.GetScenario(), req.Msg.GetEffect(), req.Msg.GetDryRun())), nil
}

func (s *service) ResolveIntent(ctx context.Context, req *connect.Request[bindingsv1.ResolveIntentRequest]) (*connect.Response[bindingsv1.ResolveIntentResponse], error) {
	response, err := s.resolveIntent(ctx, req.Msg.GetIntent(), req.Msg.GetLimit(), req.Msg.GetMode())
	s.metrics.record(response)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(response), nil
}

func (s *service) resolveIntent(ctx context.Context, intent string, limit int32, mode string) (*bindingsv1.ResolveIntentResponse, error) {
	if strings.TrimSpace(intent) == "" {
		return nil, fmt.Errorf("intent is required")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "judged"
	}
	if mode != "fast" && mode != "judged" && mode != "deep" {
		return nil, fmt.Errorf("mode must be fast, judged, or deep")
	}
	if mode != "fast" && unboundedDestructiveIntent(intent) {
		return nullVerdictDiscoveryResponse(&bindingsv1.ResolveIntentResponse{Mode: mode}, "intent is unbounded or unauthorized; no governed binding is safe to select"), nil
	}
	response := &bindingsv1.ResolveIntentResponse{Mode: mode}
	ranked, err := s.retrieveIntentCandidates(ctx, intent, limit, mode)
	if err != nil {
		return unavailableDiscoveryResponse(response, err.Error()), nil
	}
	if mode == "fast" && len(ranked) > 0 && ranked[0].GetScore() < minimumFastDiscoveryScore {
		return nullVerdictDiscoveryResponse(response, fmt.Sprintf("best binding match scored %.2f, below the fast discovery confidence floor", ranked[0].GetScore())), nil
	}
	candidates := joinSearchHits(s.registry, ranked, nil, int(limit))
	response.Bindings = candidates
	response.Reason = "provider-direct binding discovery"
	if s.library != nil {
		libraryHits, libraryErr := s.retrieveLibraryCandidates(ctx, intent, limit)
		if libraryErr != nil {
			return unavailableDiscoveryResponse(response, libraryErr.Error()), nil
		}
		if len(libraryHits) > 0 && (len(ranked) == 0 || libraryHits[0].GetScore() >= ranked[0].GetScore()) {
			program, getErr := s.library.Get(ctx, libraryHits[0].GetId(), 0)
			if getErr == nil && program != nil {
				response.Result = &bindingsv1.DiscoverResult{Confidence: "high", Method: mode + ".library-verified", Reason: "verified library program outranked raw bindings", Library: program}
				return response, nil
			}
		}
	}
	if len(candidates) == 0 {
		return nullVerdictDiscoveryResponse(response, "no governed binding or library program matched the intent"), nil
	}
	selected := candidates[0]
	confidence := "low"
	method := mode + ".provider-direct"
	if mode != "fast" {
		var judgeErr error
		selected, confidence, judgeErr = s.judgeIntent(ctx, intent, candidates)
		if judgeErr != nil {
			return unavailableDiscoveryResponse(response, judgeErr.Error()), nil
		}
		if selected == nil {
			return nullVerdictDiscoveryResponse(response, "judge returned a null verdict"), nil
		}
		method = mode + ".judge.default"
	}
	result := &bindingsv1.DiscoverResult{BindingId: selected.GetId(), Confidence: confidence, Method: method, Reason: response.Reason, Binding: selected}
	for _, candidate := range candidates[1:] {
		result.Alternatives = append(result.Alternatives, candidate.GetId())
	}
	if described, describeErr := s.registry.Describe(selected.GetId()); describeErr == nil {
		result.Arguments = described.GetArguments()
	}
	response.Result = result
	return response, nil
}

// Search Hub's binding provider emits a normalized score. Fast discovery has
// no judge to recover from a weak lexical match, so returning its arbitrary
// top hit is more dangerous than an explicit null verdict. The floor is below
// the reviewed exact/alias matches (1.0 and 0.75 in the live corpus) while
// rejecting the 0.25 catch-all result returned for unrelated intents.
const minimumFastDiscoveryScore = 0.5

func (s *service) retrieveIntentCandidates(ctx context.Context, intent string, limit int32, mode string) ([]*routingv1.SearchHit, error) {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "search-hub")
	if err != nil {
		return nil, fmt.Errorf("search-hub unavailable: %v", err)
	}
	client := routingconnect.NewRoutingServiceClient(http.DefaultClient, strings.TrimRight(base, "/"))
	variants := []string{intent}
	if mode == "deep" {
		variants = []string{intent, "how do I " + intent, "operation for " + intent}
		if len(variants) > maxDeepParaphrases {
			variants = variants[:maxDeepParaphrases]
		}
	}
	byID := make(map[string]*routingv1.SearchHit)
	for _, variant := range variants {
		search, queryErr := client.Query(ctx, connect.NewRequest(&routingv1.QueryRequest{Query: variant, Types: []string{"binding"}, Limit: limit}))
		if queryErr != nil {
			return nil, fmt.Errorf("search-hub unavailable: %v", queryErr)
		}
		for _, hit := range orderedSearchHits(search.Msg) {
			if hit != nil && hit.GetId() != "" {
				if previous, exists := byID[hit.GetId()]; !exists || hit.GetScore() > previous.GetScore() {
					byID[hit.GetId()] = hit
				}
			}
		}
	}
	ranked := make([]*routingv1.SearchHit, 0, len(byID))
	for _, hit := range byID {
		ranked = append(ranked, hit)
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].GetScore() == ranked[j].GetScore() {
			return ranked[i].GetId() < ranked[j].GetId()
		}
		return ranked[i].GetScore() > ranked[j].GetScore()
	})
	return ranked, nil
}

func (s *service) retrieveLibraryCandidates(ctx context.Context, intent string, limit int32) ([]*routingv1.SearchHit, error) {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "search-hub")
	if err != nil {
		return nil, fmt.Errorf("search-hub unavailable: %v", err)
	}
	client := routingconnect.NewRoutingServiceClient(http.DefaultClient, strings.TrimRight(base, "/"))
	search, err := client.Query(ctx, connect.NewRequest(&routingv1.QueryRequest{Query: intent, Types: []string{"library"}, Limit: limit}))
	if err != nil {
		return nil, fmt.Errorf("search-hub unavailable: %v", err)
	}
	return orderedSearchHits(search.Msg), nil
}

func (s *service) judgeIntent(ctx context.Context, intent string, candidates []*bindingsv1.Binding) (*bindingsv1.Binding, string, error) {
	if len(candidates) == 0 {
		return nil, "low", nil
	}
	judgeCandidates := make([]map[string]any, 0, len(candidates))
	for rank, candidate := range candidates {
		if rank >= maxJudgeCandidates {
			break
		}
		described, _ := s.registry.Describe(candidate.GetId())
		fields := make([]string, 0)
		if described != nil {
			for _, argument := range described.GetArguments() {
				fields = append(fields, argument.GetName())
			}
		}
		judgeCandidates = append(judgeCandidates, map[string]any{"binding_id": candidate.GetId(), "rank": rank + 1, "effect": candidate.GetEffect(), "scenario": candidate.GetScenario(), "group": candidate.GetGroup(), "command": candidate.GetCommand(), "arguments": fields, "description": candidate.GetDescription(), "intent_hints": bindingIntentAliases(candidate)})
	}
	source, err := json.Marshal(map[string]any{"intent": intent, "candidates": judgeCandidates})
	if err != nil {
		return nil, "low", fmt.Errorf("encode discovery judge input: %w", err)
	}
	schema := `{"type":"object","properties":{"binding_id":{"type":"string"},"confidence":{"enum":["high","medium","low"]}},"required":["binding_id","confidence"]}`
	result, err := s.registry.Execute(ctx, "ai-gateway/inference/run", map[string]any{"source": string(source), "schema_json": schema, "instruction": "Select exactly one candidate when it directly serves the intent. Return an empty binding_id when no candidate is justified. Confidence must reflect evidence in the candidate contract, not guesswork.", "role": "judge.default"}, nil, false, bindings.InvocationMetadata{Provenance: "program-runtime.discovery"}, &http.Client{Timeout: budgets.BridgeCall})
	if err != nil {
		return nil, "low", fmt.Errorf("judge.default unavailable: %v", err)
	}
	valueJSON, _ := result["valueJson"].(string)
	if valueJSON == "" {
		valueJSON, _ = result["value_json"].(string)
	}
	var verdict struct {
		BindingID  string `json:"binding_id"`
		Confidence string `json:"confidence"`
	}
	if err := json.Unmarshal([]byte(valueJSON), &verdict); err != nil {
		return nil, "low", fmt.Errorf("judge.default returned invalid verdict: %w", err)
	}
	if verdict.BindingID == "" {
		return nil, verdict.Confidence, nil
	}
	for _, candidate := range candidates {
		if candidate.GetId() == verdict.BindingID {
			if verdict.Confidence != "high" && verdict.Confidence != "medium" && verdict.Confidence != "low" {
				return nil, "low", fmt.Errorf("judge.default returned invalid confidence %q", verdict.Confidence)
			}
			return candidate, verdict.Confidence, nil
		}
	}
	return nil, "low", fmt.Errorf("judge.default selected binding %q outside the candidate set", verdict.BindingID)
}

func unboundedDestructiveIntent(intent string) bool {
	intent = strings.ToLower(strings.TrimSpace(intent))
	for _, phrase := range []string{"delete every", "delete all", "erase every", "erase all", "destroy every", "destroy all", "remove every", "without authorization"} {
		if strings.Contains(intent, phrase) {
			return true
		}
	}
	return false
}

// orderedSearchHits makes the typed selection honor Search Hub's score
// contract. A provider's response order is not a stable identity signal: the
// router may return grouped and federated hits in a transport-dependent order.
// Score-descending order with an id tie-break is deterministic and remains
// independent of registry/map iteration order.
func orderedSearchHits(response *routingv1.QueryResponse) []*routingv1.SearchHit {
	if response == nil {
		return nil
	}
	hits := append([]*routingv1.SearchHit(nil), response.GetRanked()...)
	if len(hits) == 0 {
		for _, group := range response.GetGroups() {
			if group != nil {
				hits = append(hits, group.GetHits()...)
			}
		}
	}
	hits = slices.DeleteFunc(hits, func(hit *routingv1.SearchHit) bool { return hit == nil })
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].GetScore() == hits[j].GetScore() {
			return hits[i].GetId() < hits[j].GetId()
		}
		return hits[i].GetScore() > hits[j].GetScore()
	})
	return hits
}

// unavailableDiscoveryResponse reports that discovery could not reach a
// verdict because a dependency failed. It is not the same as deciding that
// nothing serves the intent — see nullVerdictDiscoveryResponse — and the two
// were conflated until `unavailable` existed to tell them apart.
func unavailableDiscoveryResponse(response *bindingsv1.ResolveIntentResponse, reason string) *bindingsv1.ResolveIntentResponse {
	response.Reason = reason
	response.Fallback = true
	response.Result = &bindingsv1.DiscoverResult{Method: response.GetMode() + ".unavailable", Reason: reason, Unavailable: true}
	return response
}

// nullVerdictDiscoveryResponse reports the honest verdict that no governed
// capability serves the intent. Retrieval and judging both worked; the answer
// is no. A caller should stop rather than retry.
func nullVerdictDiscoveryResponse(response *bindingsv1.ResolveIntentResponse, reason string) *bindingsv1.ResolveIntentResponse {
	response.Reason = reason
	response.Result = &bindingsv1.DiscoverResult{Method: response.GetMode() + ".null", Reason: reason}
	return response
}

func topBindingScore(response *routingv1.QueryResponse) float64 {
	if response == nil {
		return 0
	}
	var best float64
	for _, hit := range orderedSearchHits(response) {
		if hit.GetScore() > best {
			best = hit.GetScore()
		}
	}
	return best
}

func joinSearchHits(registry *bindings.Registry, ranked []*routingv1.SearchHit, groups []*routingv1.ProviderResultGroup, limit int) []*bindingsv1.Binding {
	seen := make(map[string]struct{})
	out := make([]*bindingsv1.Binding, 0, limit)
	consume := func(hit *routingv1.SearchHit) {
		if hit == nil || len(out) >= limit {
			return
		}
		// Search Hub's binding provider returns the canonical binding id in the
		// result id. Identity is resolved directly; no path suffix or map order
		// heuristic is permitted here.
		id := strings.TrimSpace(hit.GetId())
		if id == "" {
			return
		}
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
	cells := req.Msg.GetCells()
	if len(cells) == 0 {
		definition, err := s.loadActSpace()
		if err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, err)
		}
		verdicts := actspace.Audit(ctx, s.registry, definition)
		return connect.NewResponse(s.actResponse(ctx, verdicts, int32(len(verdicts)), string(actspace.Confidence(verdicts)))), nil
	}
	verdicts := s.registry.ResolveActCells(ctx, cells)
	confidence := string(actspace.Confidence(verdicts))
	return connect.NewResponse(s.actResponse(ctx, verdicts, int32(len(cells)), confidence)), nil
}

func (s *service) actResponse(ctx context.Context, verdicts []*bindingsv1.ActCellVerdict, total int32, confidence string) *bindingsv1.ResolveActCellsResponse {
	response := &bindingsv1.ResolveActCellsResponse{Cells: verdicts, AuditedCells: int32(len(verdicts)), TotalCells: total, DenominatorConfidence: confidence}
	doctor := s.registry.DoctorContext(ctx, "")
	response.ManifestScenarios = doctor.GetManifestScenarios()
	response.TotalScenarios = doctor.GetTotalScenarios()
	response.ReachableScenarios = int32(len(doctor.GetReachableScenarios()))
	response.UnreachableScenarios = int32(len(doctor.GetUnreachableScenarios()))
	response.ReachabilityCheckedAt = doctor.GetReachabilityCheckedAt()
	return response
}

func (s *service) loadActSpace() (*spacedoc.SpaceDefinition, error) {
	path := s.actSpacePath
	if path == "" {
		root, err := repocontract.FindRepoRootFromEnvOrCWD()
		if err != nil {
			return nil, fmt.Errorf("resolve repository root for Act denominator: %w", err)
		}
		path = filepath.Join(root, "scenarios", "program-runtime", "docs", "spaces", "act-space.md")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Act denominator %q unavailable: %w", path, err)
	}
	definition, err := spacedoc.Parse(spacedoc.ProjectionAct, data)
	if err != nil {
		return nil, fmt.Errorf("parse Act denominator %q: %w", path, err)
	}
	return definition, nil
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
