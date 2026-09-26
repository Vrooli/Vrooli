package handlers

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"agent-manager/internal/conversationsearch"
	"github.com/gorilla/mux"
	aisearch "github.com/vrooli/ai-go/search"
)

// ConversationSearchGenerationOwner is the narrow owner API used by the
// operator retention endpoints. It keeps cleanup policy and Qdrant semantics
// out of the transport layer.
type ConversationSearchGenerationOwner interface {
	InspectGenerationLifecycle(context.Context, aisearch.GenerationRetentionPolicy) (aisearch.GenerationInspection, error)
	PreviewGenerationCleanup(context.Context, aisearch.GenerationRetentionPolicy, string) (aisearch.GenerationCleanupPlan, error)
	ApplyGenerationCleanup(context.Context, aisearch.GenerationCleanupPlan) (aisearch.GenerationCleanupReceipt, error)
}

type ConversationSearchGenerationHandler struct {
	owner ConversationSearchGenerationOwner
	token func() string
}

func NewConversationSearchGenerationHandler(owner ConversationSearchGenerationOwner, token func() string) *ConversationSearchGenerationHandler {
	return &ConversationSearchGenerationHandler{owner: owner, token: token}
}

func (h *ConversationSearchGenerationHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/v1/conversation-search/generations", h.inspect).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/conversation-search/generations/cleanup/preview", h.preview).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/conversation-search/generations/cleanup/apply", h.apply).Methods(http.MethodPost)
}

func (h *ConversationSearchGenerationHandler) authorize(request *http.Request) bool {
	want := ""
	if h != nil && h.token != nil {
		want = h.token()
	}
	presented := strings.TrimSpace(request.Header.Get("X-Search-Control-Token"))
	return want != "" && presented != "" && len(want) == len(presented) && subtle.ConstantTimeCompare([]byte(want), []byte(presented)) == 1
}

func (h *ConversationSearchGenerationHandler) inspect(writer http.ResponseWriter, request *http.Request) {
	if !h.authorize(request) {
		h.writeError(writer, http.StatusUnauthorized, errors.New("conversation search generation control is unavailable"))
		return
	}
	if h.owner == nil {
		h.writeError(writer, http.StatusServiceUnavailable, errors.New("conversation search generation owner is unavailable"))
		return
	}
	result, err := h.owner.InspectGenerationLifecycle(request.Context(), aisearch.DefaultGenerationRetentionPolicy())
	if err != nil {
		h.writeError(writer, http.StatusServiceUnavailable, err)
		return
	}
	h.writeJSON(writer, http.StatusOK, result)
}

func (h *ConversationSearchGenerationHandler) preview(writer http.ResponseWriter, request *http.Request) {
	if !h.authorize(request) {
		h.writeError(writer, http.StatusUnauthorized, errors.New("conversation search generation control is unavailable"))
		return
	}
	var input struct {
		PlanIdentity string                             `json:"planIdentity"`
		Policy       aisearch.GenerationRetentionPolicy `json:"policy"`
	}
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		h.writeError(writer, http.StatusBadRequest, err)
		return
	}
	policy := input.Policy
	if policy.PolicyRevision == "" {
		policy = aisearch.DefaultGenerationRetentionPolicy()
	}
	result, err := h.owner.PreviewGenerationCleanup(request.Context(), policy, input.PlanIdentity)
	if err != nil {
		h.writeError(writer, http.StatusBadRequest, err)
		return
	}
	h.writeJSON(writer, http.StatusOK, result)
}

func (h *ConversationSearchGenerationHandler) apply(writer http.ResponseWriter, request *http.Request) {
	if !h.authorize(request) {
		h.writeError(writer, http.StatusUnauthorized, errors.New("conversation search generation control is unavailable"))
		return
	}
	var plan aisearch.GenerationCleanupPlan
	if err := json.NewDecoder(request.Body).Decode(&plan); err != nil {
		h.writeError(writer, http.StatusBadRequest, err)
		return
	}
	result, err := h.owner.ApplyGenerationCleanup(request.Context(), plan)
	if err != nil {
		h.writeError(writer, http.StatusConflict, err)
		return
	}
	h.writeJSON(writer, http.StatusOK, result)
}

func (h *ConversationSearchGenerationHandler) writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func (h *ConversationSearchGenerationHandler) writeError(writer http.ResponseWriter, status int, err error) {
	h.writeJSON(writer, status, map[string]string{"error": err.Error()})
}

var _ ConversationSearchGenerationOwner = (*conversationsearch.SemanticRuntime)(nil)
