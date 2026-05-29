package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"prompt-manager/store"

	"github.com/gorilla/mux"
)

type GraphInvalidator interface {
	Invalidate()
}

type AISearchIndexer interface {
	IndexAction(ctx context.Context, actionID string) error
	DeleteActionFromIndex(ctx context.Context, actionID string) error
}

type Handlers struct {
	service          *Service
	aiIndexer        AISearchIndexer
	graphInvalidator GraphInvalidator
}

func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

func (h *Handlers) SetGraphInvalidator(inv GraphInvalidator) {
	h.graphInvalidator = inv
}

func (h *Handlers) SetAIIndexer(indexer AISearchIndexer) {
	h.aiIndexer = indexer
}

func (h *Handlers) invalidateGraph() {
	if h.graphInvalidator != nil {
		h.graphInvalidator.Invalidate()
	}
}

func (h *Handlers) triggerIndexAsync(actionID string) {
	if h.aiIndexer == nil {
		return
	}
	go func() {
		if err := h.aiIndexer.IndexAction(context.Background(), actionID); err != nil {
			fmt.Printf("[actions] AI index update failed for %s: %v\n", actionID, err)
		}
	}()
}

func (h *Handlers) triggerDeleteAsync(actionID string) {
	if h.aiIndexer == nil {
		return
	}
	go func() {
		if err := h.aiIndexer.DeleteActionFromIndex(context.Background(), actionID); err != nil {
			fmt.Printf("[actions] AI index delete failed for %s: %v\n", actionID, err)
		}
	}()
}

func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	actions, err := h.service.List(r.Context(), ListFilters{
		Pack:   r.URL.Query().Get("pack"),
		Status: r.URL.Query().Get("status"),
		Owner:  r.URL.Query().Get("owner"),
		Tag:    r.URL.Query().Get("tag"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeAPIJSON(w, http.StatusOK, actions)
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	action, err := h.service.Get(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeActionError(w, err)
		return
	}
	writeAPIJSON(w, http.StatusOK, action)
}

func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	action := req.Action
	created, validation, err := h.service.Create(r.Context(), req.Pack, &action)
	if err != nil {
		writeActionError(w, err)
		return
	}
	if !validation.Valid {
		writeAPIJSON(w, http.StatusUnprocessableEntity, validation)
		return
	}
	h.invalidateGraph()
	h.triggerIndexAsync(created.ID)
	writeAPIJSON(w, http.StatusCreated, MutationResponse{Action: created, Validation: validation})
}

// Preview renders the contract that Create would write — inferred fields,
// validation, and similar existing actions — without persisting anything.
func (h *Handlers) Preview(w http.ResponseWriter, r *http.Request) {
	var draft DraftActionInput
	if err := json.NewDecoder(r.Body).Decode(&draft); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	preview, err := h.service.PreviewCreate(r.Context(), draft)
	if err != nil {
		writeActionError(w, err)
		return
	}
	writeAPIJSON(w, http.StatusOK, preview)
}

func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var action store.Action
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	updated, validation, err := h.service.Update(r.Context(), id, &action)
	if err != nil {
		writeActionError(w, err)
		return
	}
	if !validation.Valid {
		writeAPIJSON(w, http.StatusUnprocessableEntity, validation)
		return
	}
	h.invalidateGraph()
	h.triggerIndexAsync(updated.ID)
	writeAPIJSON(w, http.StatusOK, MutationResponse{Action: updated, Validation: validation})
}

func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var err error
	if strings.EqualFold(r.URL.Query().Get("hard"), "true") {
		err = h.service.Delete(r.Context(), id)
	} else {
		err = h.service.Archive(r.Context(), id)
	}
	if err != nil {
		writeActionError(w, err)
		return
	}
	h.invalidateGraph()
	if strings.EqualFold(r.URL.Query().Get("hard"), "true") {
		h.triggerDeleteAsync(id)
	} else {
		h.triggerIndexAsync(id)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) Validate(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	result, err := h.service.ValidateByID(r.Context(), id)
	if err != nil {
		writeActionError(w, err)
		return
	}
	if !result.Valid {
		writeAPIJSON(w, http.StatusUnprocessableEntity, result)
		return
	}
	writeAPIJSON(w, http.StatusOK, result)
}

func (h *Handlers) Run(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var req RunRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if !errors.Is(err, io.EOF) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
	}
	result, err := h.service.Run(r.Context(), id, req)
	if err != nil {
		writeActionError(w, err)
		return
	}
	writeRunResponse(w, result)
}

func writeRunResponse(w http.ResponseWriter, result RunResponse) {
	status := http.StatusOK
	switch result.Status {
	case RunStatusRejected:
		status = http.StatusUnprocessableEntity
	case RunStatusThrottled:
		status = http.StatusTooManyRequests
	}
	writeAPIJSON(w, status, result)
}

func writeAPIJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeActionError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "not found"), strings.Contains(msg, "invalid action id"):
		status = http.StatusNotFound
	case strings.Contains(msg, "already exists"):
		status = http.StatusConflict
	case strings.Contains(msg, "invalid"), strings.Contains(msg, "required"), strings.Contains(msg, "unsupported"), strings.Contains(msg, "cannot be changed"):
		status = http.StatusBadRequest
	}
	http.Error(w, err.Error(), status)
}
