// Package channelmanager exposes the manual-operations API.  It intentionally
// never accepts a platform credential: identity creation takes a Vault path
// reference and every platform interaction remains an operator action.
package channelmanager

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	core "channel-manager/internal/channelmanager"
	"channel-manager/internal/module"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	channelmanagerv1 "github.com/vrooli/vrooli/packages/proto/gen/go/channel-manager/v1/channelmanager"
	channelmanagerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/channel-manager/v1/channelmanager/channelmanager_v1connect"
)

type api struct {
	service *core.Service
	store   core.Store
	mu      sync.Mutex
}

func Module(service *core.Service, store core.Store) module.Module {
	h := &api{service: service, store: store}
	return module.Module{Name: "channel-manager", Endpoints: Endpoints, Mount: func(r *mux.Router) {
		connectPath, connectHandler := channelmanagerconnect.NewChannelManagerServiceHandler(h)
		connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		r.HandleFunc("/api/v1/channel-manager/overview", h.overview).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/channel-manager/identities", h.createIdentity).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/channel-manager/identities/{id}/start", h.start).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/channel-manager/actions", h.enqueue).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/channel-manager/actions/{id}/complete", h.complete).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/channel-manager/identities/{id}/observations", h.observe).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/channel-manager/identities/{id}/eligibility", h.eligibility).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/channel-manager/releases", h.release).Methods(http.MethodPost)
	}}
}

func (h *api) GetOverview(_ context.Context, _ *connect.Request[channelmanagerv1.GetOverviewRequest]) (*connect.Response[channelmanagerv1.GetOverviewResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	response := &channelmanagerv1.GetOverviewResponse{}
	for _, identity := range h.service.Identities {
		response.Identities = append(response.Identities, &channelmanagerv1.Identity{Id: identity.ID, PlatformId: identity.PlatformID, Purpose: identity.Purpose, EnvironmentRef: identity.EnvironmentRef, VaultRef: identity.VaultRef, Status: identity.Status, LaneGrants: identity.LaneGrants})
	}
	for _, action := range h.service.Actions {
		response.Actions = append(response.Actions, &channelmanagerv1.Action{Id: action.ID, IdentityId: action.IdentityID, Kind: action.Kind, Window: action.Window.Format(time.RFC3339), Status: string(action.Status), RolledCount: int32(action.RolledCount)})
	}
	return connect.NewResponse(response), nil
}

func Schema() string { return core.Schema() }

var Endpoints = []module.EndpointDescriptor{{
	ID: "channel_manager_overview", Path: channelmanagerconnect.ChannelManagerServiceGetOverviewProcedure, Method: http.MethodPost,
	Summary: "Get channel manager overview", Description: "Returns identity and queued-action references without credential values.", Category: "channel-manager",
	Response: &module.Schema{Type: "object", Properties: map[string]string{"identities": "array<Identity>", "actions": "array<Action>"}},
}}

func (h *api) overview(w http.ResponseWriter, _ *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"identities": h.service.Identities, "actions": h.service.Actions, "programs": h.service.Programs, "flags": h.service.Flags})
}

func (h *api) createIdentity(w http.ResponseWriter, r *http.Request) {
	var identity core.Identity
	if !decode(w, r, &identity) {
		return
	}
	h.mu.Lock()
	err := h.service.CreateIdentity(identity)
	if err == nil {
		err = h.store.Save(r.Context(), h.service)
	}
	h.mu.Unlock()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, identity)
}

func (h *api) start(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProgramID string `json:"program_id"`
	}
	if !decode(w, r, &req) {
		return
	}
	h.mu.Lock()
	err := h.service.StartProgram(mux.Vars(r)["id"], req.ProgramID)
	if err == nil {
		err = h.store.Save(r.Context(), h.service)
	}
	h.mu.Unlock()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "warming"})
}

func (h *api) enqueue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IdentityID     string    `json:"identity_id"`
		Kind           string    `json:"kind"`
		Window         time.Time `json:"window"`
		Seed           uint64    `json:"seed"`
		IdempotencyKey string    `json:"idempotency_key"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.Window.IsZero() {
		req.Window = time.Now().UTC()
	}
	h.mu.Lock()
	a, err := h.service.Enqueue(req.IdentityID, req.Kind, req.Window, req.Seed, req.IdempotencyKey)
	if err == nil {
		err = h.store.Save(r.Context(), h.service)
	}
	h.mu.Unlock()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (h *api) complete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Evidence string `json:"evidence"`
	}
	if !decode(w, r, &req) {
		return
	}
	h.mu.Lock()
	err := h.service.Complete(mux.Vars(r)["id"], req.Evidence, time.Now().UTC())
	if err == nil {
		err = h.store.Save(r.Context(), h.service)
	}
	h.mu.Unlock()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "succeeded"})
}

func (h *api) observe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Metric string  `json:"metric"`
		Value  float64 `json:"value"`
	}
	if !decode(w, r, &req) {
		return
	}
	h.mu.Lock()
	flag, err := h.service.RecordObservation(mux.Vars(r)["id"], req.Metric, req.Value, time.Now().UTC(), 3, .5)
	if err == nil {
		err = h.store.Save(r.Context(), h.service)
	}
	h.mu.Unlock()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"flag": flag})
}
func (h *api) eligibility(w http.ResponseWriter, r *http.Request) {
	lane := r.URL.Query().Get("lane")
	if lane == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "lane is required"})
		return
	}
	h.mu.Lock()
	result := h.service.Eligibility(mux.Vars(r)["id"], lane)
	h.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]string{"eligibility": result})
}
func (h *api) release(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IdentityID     string `json:"identity_id"`
		Lane           string `json:"lane"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.IdempotencyKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "idempotency_key is required"})
		return
	}
	h.mu.Lock()
	id, err := h.service.Release(req.IdentityID, req.Lane, req.IdempotencyKey, time.Now().UTC())
	if err == nil {
		err = h.store.Save(r.Context(), h.service)
	}
	h.mu.Unlock()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"post_id": id, "url": "manual://queued/" + id})
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON request"})
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, core.ErrPaused) || errors.Is(err, core.ErrCadence) || errors.Is(err, core.ErrForbiddenAction) || errors.Is(err, core.ErrPreconditions) {
		status = http.StatusConflict
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
