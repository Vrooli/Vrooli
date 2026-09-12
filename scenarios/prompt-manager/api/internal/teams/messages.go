package teams

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"prompt-manager/internal/store"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// ListTeamMessages handles GET /teams/{id}/members/{agentId}/messages.
func (h *Handlers) ListTeamMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]

	if agentID == "" {
		http.Error(w, "agentId is required", http.StatusBadRequest)
		return
	}

	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	memberSet, err := h.teamMemberSet(ctx, teamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := validateMemberExists(memberSet, agentID, "agentId"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	inbox, err := h.teamStore.GetInbox(ctx, teamID, agentID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	messages := make([]TeamMessageDTO, 0, len(inbox.Messages))
	for _, message := range inbox.Messages {
		messages = append(messages, storeMessageToDTO(message))
	}

	sort.SliceStable(messages, func(i, j int) bool {
		return messages[i].CreatedAt < messages[j].CreatedAt
	})

	resp := TeamInboxResponse{
		TeamID:   teamID,
		AgentID:  agentID,
		Messages: messages,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// SendTeamMessage handles POST /teams/{id}/members/{agentId}/messages.
func (h *Handlers) SendTeamMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]

	if agentID == "" {
		http.Error(w, "agentId is required", http.StatusBadRequest)
		return
	}

	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	var req SendTeamMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.FromAgentID == "" {
		http.Error(w, "fromAgentId is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}
	if req.FromAgentID == agentID {
		http.Error(w, "fromAgentId cannot equal agentId", http.StatusBadRequest)
		return
	}

	memberSet, err := h.teamMemberSet(ctx, teamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := validateMemberExists(memberSet, req.FromAgentID, "fromAgentId"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateMemberExists(memberSet, agentID, "agentId"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	inbox, err := h.teamStore.GetInbox(ctx, teamID, agentID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	message := store.TeamMessage{
		ID:          uuid.New().String(),
		TeamID:      teamID,
		FromAgentID: req.FromAgentID,
		ToAgentID:   agentID,
		Content:     req.Content,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	inbox.Messages = append(inbox.Messages, message)
	if err := h.teamStore.SetInbox(ctx, teamID, agentID, inbox); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(storeMessageToDTO(message))
}

// DeleteTeamMessage handles DELETE /teams/{id}/members/{agentId}/messages/{messageId}.
func (h *Handlers) DeleteTeamMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]
	messageID := vars["messageId"]

	if agentID == "" || messageID == "" {
		http.Error(w, "agentId and messageId are required", http.StatusBadRequest)
		return
	}

	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	memberSet, err := h.teamMemberSet(ctx, teamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := validateMemberExists(memberSet, agentID, "agentId"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	inbox, err := h.teamStore.GetInbox(ctx, teamID, agentID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	updated := make([]store.TeamMessage, 0, len(inbox.Messages))
	found := false
	for _, message := range inbox.Messages {
		if message.ID == messageID {
			found = true
			continue
		}
		updated = append(updated, message)
	}

	if !found {
		http.Error(w, "message not found", http.StatusNotFound)
		return
	}

	inbox.Messages = updated
	if err := h.teamStore.SetInbox(ctx, teamID, agentID, inbox); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ClearTeamMessages handles DELETE /teams/{id}/members/{agentId}/messages.
func (h *Handlers) ClearTeamMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]

	if agentID == "" {
		http.Error(w, "agentId is required", http.StatusBadRequest)
		return
	}

	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	memberSet, err := h.teamMemberSet(ctx, teamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := validateMemberExists(memberSet, agentID, "agentId"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	inbox, err := h.teamStore.GetInbox(ctx, teamID, agentID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	inbox.Messages = []store.TeamMessage{}
	if err := h.teamStore.SetInbox(ctx, teamID, agentID, inbox); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
