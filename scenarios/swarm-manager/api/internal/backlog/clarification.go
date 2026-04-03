// DOC: docs/internal/SEAMS.md#clarification-api
// DOC: docs/reference/api-endpoints.md#clarification
//
// HTTP handlers for creating and continuing workshop decision clarification
// threads. Each handler delegates to the workshop.Clarification* storage
// functions and the agentmanager.Service for agent spawning and run
// continuation.
//
// State-reading and action-applying handlers live in clarification_state.go.
package backlog

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/workshop"
)

// CreateClarification starts a new clarification thread for a workshop decision.
//
// POST /api/v1/backlog/{kind}/{name}/workshop/clarification
func (h *Handler) CreateClarification(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "clarification-create")
	if !ok {
		return
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.MapError(w, "[backlog] clarification-create", apierr.NotFound("backlog item not found"))
			return
		}
		apierr.MapError(w, "[backlog] clarification-create", apierr.Internal("failed to load backlog item"))
		return
	}

	// Parse request — supports both JSON and multipart/form-data (for file attachments).
	var roundNumber int32
	var itemID, message string
	var attachmentIDs []string

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			apierr.MapError(w, "[backlog] clarification-create", apierr.BadRequest("invalid multipart form"))
			return
		}
		rn, convErr := strconv.Atoi(r.FormValue("round_number"))
		if convErr != nil || rn < 1 {
			apierr.MapError(w, "[backlog] clarification-create", apierr.BadRequest("round_number is required and must be >= 1"))
			return
		}
		roundNumber = int32(rn)
		itemID = strings.TrimSpace(r.FormValue("item_id"))
		if itemID == "" {
			apierr.MapError(w, "[backlog] clarification-create", apierr.BadRequest("item_id is required"))
			return
		}
		message = strings.TrimSpace(r.FormValue("message"))

		// Save attached files and collect attachment IDs.
		itemDir := h.store.ItemDir(kind, name)
		savedIDs, saveErr := h.saveClarificationAttachments(itemDir, r)
		if saveErr != nil {
			apierr.MapError(w, "[backlog] clarification-create", apierr.Internal("failed to save attachments"))
			return
		}
		attachmentIDs = savedIDs
	} else {
		var req apipb.CreateClarificationRequest
		if err := httputil.DecodeProtoJSON(r, &req); err != nil {
			apierr.MapError(w, "[backlog] clarification-create", apierr.BadRequest("invalid request body"))
			return
		}
		if !httputil.ValidateProtoRequest(w, "[backlog] clarification-create", "invalid request body", &req) {
			return
		}
		roundNumber = req.RoundNumber
		itemID = req.ItemId
		message = req.Message
		attachmentIDs = req.AttachmentIds
	}

	// Verify round and item exist.
	itemDir := h.store.ItemDir(kind, name)
	rounds, err := workshop.LoadRounds(itemDir)
	if err != nil {
		apierr.MapError(w, "[backlog] clarification-create", apierr.Internal("failed to load workshop rounds"))
		return
	}

	var targetRound *workshop.Round
	for i := range rounds {
		if rounds[i].RoundNum == int(roundNumber) {
			targetRound = &rounds[i]
			break
		}
	}
	if targetRound == nil {
		apierr.MapError(w, "[backlog] clarification-create", apierr.NotFound("round %d not found", roundNumber))
		return
	}

	var targetItem *workshop.Item
	for i := range targetRound.Items {
		if targetRound.Items[i].ID == itemID {
			targetItem = &targetRound.Items[i]
			break
		}
	}
	if targetItem == nil {
		apierr.MapError(w, "[backlog] clarification-create", apierr.BadRequest("item %q not found in round %d", itemID, roundNumber))
		return
	}
	if targetItem.Type != "decision" {
		apierr.MapError(w, "[backlog] clarification-create", apierr.BadRequest("clarification is only supported for decision items"))
		return
	}

	// Build the clarification prompt.
	userMessage := strings.TrimSpace(message)
	if userMessage == "" {
		userMessage = "Please explain this decision in detail — what it means, what each option implies, and what the practical consequences are."
	}

	prompt, err := h.buildClarificationPrompt(r.Context(), item, itemDir, targetItem, userMessage, nil)
	if err != nil {
		slog.Warn("clarification prompt build failed, using fallback", "err", err)
		// Fall back to a basic prompt.
		prompt = fmt.Sprintf("Explain this workshop decision:\n\nTopic: %s\nContext: %s\n\nUser question: %s",
			targetItem.Topic, targetItem.Context, userMessage)
	}

	// Spawn the clarification agent.
	if !h.agentService.IsEnabled() {
		apierr.MapError(w, "[backlog] clarification-create", apierr.Unavailable("agent-manager is not available"))
		return
	}

	activityCtx := agentactivity.WithSpec(r.Context(), agentactivity.Spec{
		OwnerType:   agentactivity.OwnerBacklog,
		OwnerKind:   string(kind),
		OwnerName:   item.Name,
		OwnerTitle:  item.Title,
		Purpose:     "clarify",
		RequestedBy: "swarm-manager",
		Metadata: map[string]string{
			"entrypoint":   "backlog.clarification",
			"round_number": fmt.Sprintf("%d", roundNumber),
			"item_id":      itemID,
		},
	})

	runResult, err := h.agentService.SpawnBacklog(activityCtx, agentmanager.BacklogSpawnRequest{
		Kind:        string(kind),
		Name:        item.Name,
		Title:       fmt.Sprintf("Clarify: %s — %s", item.Title, targetItem.Topic),
		Description: prompt,
		Prompt:      prompt,
		ScopePath:   ".",
		ProjectRoot: ".",
		CreatedBy:   "swarm-manager",
		Purpose:     "clarify",
		Environment: map[string]string{
			"VROOLI_SPAWN_SOURCE": string(kind) + "/" + item.Name,
		},
	})
	if err != nil {
		slog.Error("clarification agent spawn failed", "err", err)
		apierr.MapError(w, "[backlog] clarification-create", apierr.Internal("failed to spawn clarification agent"))
		return
	}

	// Create the thread.
	now := time.Now().UTC().Format(time.RFC3339)
	thread := &workshop.ClarificationThread{
		ID:          uuid.New().String(),
		RoundNumber: int(roundNumber),
		ItemID:      itemID,
		RunID:       runResult.RunID,
		Messages: []workshop.ClarificationMessage{
			{
				Role:          "user",
				Content:       userMessage,
				CreatedAt:     now,
				AttachmentIDs: attachmentIDs,
			},
		},
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := workshop.SaveClarification(itemDir, thread); err != nil {
		slog.Error("clarification save failed", "err", err)
		apierr.MapError(w, "[backlog] clarification-create", apierr.Internal("failed to save clarification thread"))
		return
	}

	// Emit analytics event.
	if h.eventLogger != nil {
		h.eventLogger.EmitClarificationStarted(
			string(kind)+"/"+name,
			int(roundNumber),
			itemID,
			strings.TrimSpace(message) != "",
		)
	}

	// Link the clarification to the decision item.
	threadID := thread.ID
	targetItem.ClarificationID = &threadID
	if err := h.saveRoundItem(itemDir, targetRound, targetItem); err != nil {
		slog.Warn("failed to link clarification to item", "err", err)
	}

	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, &apipb.CreateClarificationResponse{
		Thread: clarificationThreadToProto(thread),
	}); err != nil {
		slog.Error("failed to write clarification-create response", "err", err)
	}
}

// ContinueClarification sends a follow-up message in an existing thread.
//
// POST /api/v1/backlog/{kind}/{name}/workshop/clarification/{threadId}/continue
func (h *Handler) ContinueClarification(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "clarification-continue")
	if !ok {
		return
	}

	threadID := mux.Vars(r)["threadId"]
	if strings.TrimSpace(threadID) == "" {
		apierr.MapError(w, "[backlog] clarification-continue", apierr.BadRequest("threadId is required"))
		return
	}

	// Parse request — supports both JSON and multipart/form-data.
	var continueMessage string
	var continueAttachmentIDs []string

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			apierr.MapError(w, "[backlog] clarification-continue", apierr.BadRequest("invalid multipart form"))
			return
		}
		continueMessage = strings.TrimSpace(r.FormValue("message"))
		if continueMessage == "" {
			apierr.MapError(w, "[backlog] clarification-continue", apierr.BadRequest("message is required"))
			return
		}
		itemDir := h.store.ItemDir(kind, name)
		savedIDs, saveErr := h.saveClarificationAttachments(itemDir, r)
		if saveErr != nil {
			apierr.MapError(w, "[backlog] clarification-continue", apierr.Internal("failed to save attachments"))
			return
		}
		continueAttachmentIDs = savedIDs
	} else {
		var req apipb.ContinueClarificationRequest
		if err := httputil.DecodeProtoJSON(r, &req); err != nil {
			apierr.MapError(w, "[backlog] clarification-continue", apierr.BadRequest("invalid request body"))
			return
		}
		if !httputil.ValidateProtoRequest(w, "[backlog] clarification-continue", "invalid request body", &req) {
			return
		}
		continueMessage = req.Message
		continueAttachmentIDs = req.AttachmentIds
	}

	itemDir := h.store.ItemDir(kind, name)
	thread, err := workshop.LoadClarificationByID(itemDir, threadID)
	if err != nil {
		apierr.MapError(w, "[backlog] clarification-continue", apierr.Internal("failed to load clarification"))
		return
	}
	if thread == nil {
		apierr.MapError(w, "[backlog] clarification-continue", apierr.NotFound("clarification thread not found"))
		return
	}
	if thread.Status != "active" {
		apierr.MapError(w, "[backlog] clarification-continue", apierr.Conflict("clarification thread is %s, cannot continue", thread.Status))
		return
	}

	// Append user message.
	now := time.Now().UTC().Format(time.RFC3339)
	thread.Messages = append(thread.Messages, workshop.ClarificationMessage{
		Role:          "user",
		Content:       continueMessage,
		CreatedAt:     now,
		AttachmentIDs: continueAttachmentIDs,
	})
	thread.UpdatedAt = now

	// Continue the agent run.
	if thread.RunID != "" && h.agentService.IsEnabled() {
		if err := h.agentService.ContinueRun(r.Context(), thread.RunID, continueMessage); err != nil {
			slog.Warn("ContinueRun failed", "err", err)
			// Non-fatal — the message is still stored, user can retry.
		}
	}

	if err := workshop.SaveClarification(itemDir, thread); err != nil {
		apierr.MapError(w, "[backlog] clarification-continue", apierr.Internal("failed to save clarification"))
		return
	}

	if err := httputil.ProtoJSONWithStatus(w, http.StatusOK, &apipb.ContinueClarificationResponse{
		Thread: clarificationThreadToProto(thread),
	}); err != nil {
		slog.Error("failed to write clarification-continue response", "err", err)
	}
}
