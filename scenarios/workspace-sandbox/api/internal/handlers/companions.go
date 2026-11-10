package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"workspace-sandbox/internal/types"
)

// Companion services are deliberately declarative. The API accepts only
// known service families and pinned versions; it never accepts images,
// commands, host paths, credentials, or arbitrary network destinations.
type CompanionServiceSpec struct {
	Name    string            `json:"name"`
	Kind    string            `json:"kind"`
	Version string            `json:"version"`
	Options map[string]string `json:"options,omitempty"`
}

type CompanionServiceState struct {
	Spec      CompanionServiceSpec `json:"spec"`
	Status    string               `json:"status"`
	RuntimeID string               `json:"runtimeId,omitempty"`
	Address   string               `json:"address,omitempty"`
	Error     string               `json:"error,omitempty"`
}

type CompanionRuntime struct {
	ID      string
	Address string
}

// CompanionServiceAPI is the provider seam for scenario-owned service
// runtimes. Implementations must isolate each sandbox and make Start
// idempotent for (sandboxID, service name, spec version).
type CompanionServiceAPI interface {
	Start(context.Context, uuid.UUID, CompanionServiceSpec) (CompanionRuntime, error)
	Stop(context.Context, uuid.UUID, CompanionServiceState) error
}

type companionSandboxStore interface {
	Get(context.Context, uuid.UUID) (*types.Sandbox, error)
	UpdateWithVersionCheck(context.Context, *types.Sandbox, int64) error
	LogAuditEvent(context.Context, *types.AuditEvent) error
}

var companionName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
var companionVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+(?:\.[0-9]+)?(?:[-+][A-Za-z0-9.-]+)?$`)

// PutCompanionServices records a complete, reproducible service declaration.
// A replacement is rejected while any existing service is running.
func (h *Handlers) PutCompanionServices(w http.ResponseWriter, r *http.Request) {
	store, id, ok := h.companionStore(w, r)
	if !ok {
		return
	}
	var input struct {
		Services []CompanionServiceSpec `json:"services"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		h.JSONError(w, "invalid companion service declaration: "+err.Error(), http.StatusBadRequest)
		return
	}
	states, err := validateCompanionSpecs(input.Services)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	sb, err := store.Get(r.Context(), id)
	if err != nil {
		h.HandleDomainError(w, err)
		return
	}
	if sb.Status != types.StatusActive && sb.Status != types.StatusStopped {
		h.JSONError(w, "companion services can only be declared for active or stopped sandboxes", http.StatusConflict)
		return
	}
	old := readCompanionStates(sb.Metadata)
	for _, state := range old {
		if state.Status == "running" {
			h.JSONError(w, "stop running companion services before replacing the declaration", http.StatusConflict)
			return
		}
	}
	if sb.Metadata == nil {
		sb.Metadata = map[string]interface{}{}
	}
	sb.Metadata["companionServices"] = states
	version := sb.Version
	if err := store.UpdateWithVersionCheck(r.Context(), sb, version); err != nil {
		h.JSONError(w, "sandbox changed while saving companion services; retry with the current sandbox", http.StatusConflict)
		return
	}
	_ = store.LogAuditEvent(r.Context(), &types.AuditEvent{SandboxID: &id, EventType: "companion_services.declared", ActorType: "system", Details: map[string]interface{}{"count": len(states)}})
	h.JSONSuccess(w, map[string]interface{}{"sandboxId": id, "services": states, "version": sb.Version})
}

func (h *Handlers) GetCompanionServices(w http.ResponseWriter, r *http.Request) {
	store, id, ok := h.companionStore(w, r)
	if !ok {
		return
	}
	sb, err := store.Get(r.Context(), id)
	if err != nil {
		h.HandleDomainError(w, err)
		return
	}
	h.JSONSuccess(w, map[string]interface{}{"sandboxId": id, "services": readCompanionStates(sb.Metadata)})
}

func (h *Handlers) StartCompanionService(w http.ResponseWriter, r *http.Request) {
	h.changeCompanionService(w, r, true)
}
func (h *Handlers) StopCompanionService(w http.ResponseWriter, r *http.Request) {
	h.changeCompanionService(w, r, false)
}

func (h *Handlers) changeCompanionService(w http.ResponseWriter, r *http.Request, start bool) {
	store, id, ok := h.companionStore(w, r)
	if !ok {
		return
	}
	if h.CompanionServices == nil {
		h.JSONError(w, "companion service provider is not configured", http.StatusServiceUnavailable)
		return
	}
	sb, err := store.Get(r.Context(), id)
	if err != nil {
		h.HandleDomainError(w, err)
		return
	}
	if sb.Status != types.StatusActive && sb.Status != types.StatusStopped {
		h.JSONError(w, "companion service lifecycle requires an active or stopped sandbox", http.StatusConflict)
		return
	}
	name := mux.Vars(r)["name"]
	states := readCompanionStates(sb.Metadata)
	index := -1
	for i := range states {
		if states[i].Spec.Name == name {
			index = i
			break
		}
	}
	if index < 0 {
		h.JSONError(w, "companion service is not declared", http.StatusNotFound)
		return
	}
	state := &states[index]
	if start {
		if state.Status == "running" {
			h.JSONSuccess(w, state)
			return
		}
		runtime, err := h.CompanionServices.Start(r.Context(), id, state.Spec)
		if err != nil {
			h.JSONError(w, "companion service start failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		state.Status, state.RuntimeID, state.Address, state.Error = "running", runtime.ID, runtime.Address, ""
	} else {
		if state.Status != "running" {
			h.JSONSuccess(w, state)
			return
		}
		if err := h.CompanionServices.Stop(r.Context(), id, *state); err != nil {
			h.JSONError(w, "companion service stop failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		state.Status, state.RuntimeID, state.Address, state.Error = "declared", "", "", ""
	}
	if sb.Metadata == nil {
		sb.Metadata = map[string]interface{}{}
	}
	sb.Metadata["companionServices"] = states
	version := sb.Version
	if err := store.UpdateWithVersionCheck(r.Context(), sb, version); err != nil {
		if start {
			_ = h.CompanionServices.Stop(r.Context(), id, *state)
		}
		h.JSONError(w, "sandbox changed during companion service lifecycle; retry after reconciliation", http.StatusConflict)
		return
	}
	event := "companion_services.started"
	if !start {
		event = "companion_services.stopped"
	}
	_ = store.LogAuditEvent(r.Context(), &types.AuditEvent{SandboxID: &id, EventType: event, ActorType: "system", Details: map[string]interface{}{"name": name}})
	h.JSONSuccess(w, state)
}

func (h *Handlers) companionStore(w http.ResponseWriter, r *http.Request) (companionSandboxStore, uuid.UUID, bool) {
	store := h.CompanionStore
	if store == nil {
		h.JSONError(w, "sandbox storage does not support companion service declarations", http.StatusServiceUnavailable)
		return nil, uuid.Nil, false
	}
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox id", http.StatusBadRequest)
		return nil, uuid.Nil, false
	}
	return store, id, true
}

func validateCompanionSpecs(specs []CompanionServiceSpec) ([]CompanionServiceState, error) {
	if len(specs) > 16 {
		return nil, errors.New("at most 16 companion services may be declared")
	}
	seen := map[string]bool{}
	states := make([]CompanionServiceState, 0, len(specs))
	for _, spec := range specs {
		spec.Name, spec.Kind, spec.Version = strings.TrimSpace(spec.Name), strings.ToLower(strings.TrimSpace(spec.Kind)), strings.TrimSpace(spec.Version)
		if !companionName.MatchString(spec.Name) {
			return nil, fmt.Errorf("invalid service name %q", spec.Name)
		}
		if seen[spec.Name] {
			return nil, fmt.Errorf("duplicate service name %q", spec.Name)
		}
		seen[spec.Name] = true
		if !companionVersion.MatchString(spec.Version) {
			return nil, fmt.Errorf("service %q requires a pinned numeric version", spec.Name)
		}
		if spec.Kind != "postgres" && spec.Kind != "redis" && spec.Kind != "http-mock" {
			return nil, fmt.Errorf("unsupported service kind %q; allowed kinds: postgres, redis, http-mock", spec.Kind)
		}
		if len(spec.Options) > 8 {
			return nil, fmt.Errorf("service %q has too many options", spec.Name)
		}
		for key, value := range spec.Options {
			if !regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`).MatchString(key) || len(value) > 256 {
				return nil, fmt.Errorf("service %q has an invalid option", spec.Name)
			}
			allowedOptions := map[string]map[string]bool{
				"postgres":  {"database": true},
				"redis":     {"database": true},
				"http-mock": {"fixture": true},
			}
			if !allowedOptions[spec.Kind][key] {
				return nil, fmt.Errorf("service %q option %q is not supported for %s", spec.Name, key, spec.Kind)
			}
			if !regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`).MatchString(value) {
				return nil, fmt.Errorf("service %q option %q must be a simple identifier", spec.Name, key)
			}
			if strings.Contains(strings.ToLower(key), "password") || strings.Contains(strings.ToLower(key), "token") || strings.Contains(strings.ToLower(key), "secret") || strings.Contains(strings.ToLower(key), "credential") {
				return nil, fmt.Errorf("service %q options must not contain credentials", spec.Name)
			}
		}
		states = append(states, CompanionServiceState{Spec: spec, Status: "declared"})
	}
	return states, nil
}

func readCompanionStates(metadata map[string]interface{}) []CompanionServiceState {
	if metadata == nil || metadata["companionServices"] == nil {
		return []CompanionServiceState{}
	}
	b, err := json.Marshal(metadata["companionServices"])
	if err != nil {
		return []CompanionServiceState{}
	}
	var states []CompanionServiceState
	if json.Unmarshal(b, &states) != nil || states == nil {
		return []CompanionServiceState{}
	}
	return states
}
