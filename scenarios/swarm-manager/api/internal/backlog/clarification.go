// DOC: docs/internal/SEAMS.md#clarification-api
// DOC: docs/reference/api-endpoints.md#clarification
//
// HTTP handlers for workshop decision clarification threads. Each handler
// delegates to the workshop.Clarification* storage functions and the
// agentmanager.Service for agent spawning and run continuation.
package backlog

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
		log.Printf("[backlog] clarification-create: prompt build failed: %v", err)
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
		log.Printf("[backlog] clarification-create: agent spawn failed: %v", err)
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
		log.Printf("[backlog] clarification-create: save failed: %v", err)
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
		log.Printf("[backlog] clarification-create: failed to link clarification to item: %v", err)
	}

	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, &apipb.CreateClarificationResponse{
		Thread: clarificationThreadToProto(thread),
	}); err != nil {
		log.Printf("[backlog] clarification-create: failed to write response: %v", err)
	}
}

// GetClarification returns an existing clarification thread, checking for
// agent response completion.
//
// GET /api/v1/backlog/{kind}/{name}/workshop/clarification/{threadId}
func (h *Handler) GetClarification(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "clarification-get")
	if !ok {
		return
	}

	threadID := mux.Vars(r)["threadId"]
	if strings.TrimSpace(threadID) == "" {
		apierr.MapError(w, "[backlog] clarification-get", apierr.BadRequest("threadId is required"))
		return
	}

	itemDir := h.store.ItemDir(kind, name)
	thread, err := workshop.LoadClarificationByID(itemDir, threadID)
	if err != nil {
		apierr.MapError(w, "[backlog] clarification-get", apierr.Internal("failed to load clarification"))
		return
	}
	if thread == nil {
		apierr.MapError(w, "[backlog] clarification-get", apierr.NotFound("clarification thread not found"))
		return
	}

	// If thread is active and last message is from user, check agent completion.
	if thread.Status == "active" && len(thread.Messages) > 0 &&
		thread.Messages[len(thread.Messages)-1].Role == "user" &&
		thread.RunID != "" && h.agentService.IsEnabled() {

		state, err := h.agentService.GetRunState(r.Context(), thread.RunID)
		if err == nil && isTerminalStatus(state.Status) {
			// Agent finished — check for response in the run output.
			// The agent should have written output that we can read from the run.
			// For now, we rely on the agent writing to the thread file directly,
			// or we synthesize a response from the run state.
			if state.Status == "complete" || state.Status == "completed" || state.Status == "success" {
				// Check if the agent wrote directly to the thread file.
				refreshed, _ := workshop.LoadClarificationByID(itemDir, threadID)
				if refreshed != nil && len(refreshed.Messages) > len(thread.Messages) {
					thread = refreshed
				} else if state.Summary != "" {
					// Agent didn't write to thread file — use the run summary as the response.
					thread.Messages = append(thread.Messages, workshop.ClarificationMessage{
						Role:      "assistant",
						Content:   state.Summary,
						CreatedAt: time.Now().UTC().Format(time.RFC3339),
					})
					thread.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
					_ = workshop.SaveClarification(itemDir, thread)
				}
			} else if state.Status == "failed" || state.Status == "error" {
				// Agent failed — add an error message.
				errMsg := "The clarification agent encountered an error. Please try again."
				if state.ErrorMsg != "" {
					errMsg = fmt.Sprintf("The clarification agent failed: %s", state.ErrorMsg)
				}
				thread.Messages = append(thread.Messages, workshop.ClarificationMessage{
					Role:      "assistant",
					Content:   errMsg,
					CreatedAt: time.Now().UTC().Format(time.RFC3339),
				})
				thread.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
				_ = workshop.SaveClarification(itemDir, thread)
			}

			// Parse impact from the latest assistant message.
			if latest := lastAssistantMessage(thread); latest != nil {
				if impact := workshop.ParseImpactXML(latest.Content); impact != nil {
					thread.LatestImpact = impact
					thread.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
					_ = workshop.SaveClarification(itemDir, thread)
				}
			}
		}
	}

	if err := httputil.ProtoJSONWithStatus(w, http.StatusOK, &apipb.GetClarificationResponse{
		Thread: clarificationThreadToProto(thread),
	}); err != nil {
		log.Printf("[backlog] clarification-get: failed to write response: %v", err)
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
			log.Printf("[backlog] clarification-continue: ContinueRun failed: %v", err)
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
		log.Printf("[backlog] clarification-continue: failed to write response: %v", err)
	}
}

// ClarificationAction applies a post-clarification action to the workshop.
//
// POST /api/v1/backlog/{kind}/{name}/workshop/clarification/{threadId}/action
func (h *Handler) ClarificationAction(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "clarification-action")
	if !ok {
		return
	}

	threadID := mux.Vars(r)["threadId"]
	if strings.TrimSpace(threadID) == "" {
		apierr.MapError(w, "[backlog] clarification-action", apierr.BadRequest("threadId is required"))
		return
	}

	var req apipb.ClarificationActionRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		apierr.MapError(w, "[backlog] clarification-action", apierr.BadRequest("invalid request body"))
		return
	}
	if !httputil.ValidateProtoRequest(w, "[backlog] clarification-action", "invalid request body", &req) {
		return
	}

	itemDir := h.store.ItemDir(kind, name)
	thread, err := workshop.LoadClarificationByID(itemDir, threadID)
	if err != nil {
		apierr.MapError(w, "[backlog] clarification-action", apierr.Internal("failed to load clarification"))
		return
	}
	if thread == nil {
		apierr.MapError(w, "[backlog] clarification-action", apierr.NotFound("clarification thread not found"))
		return
	}

	resp := &apipb.ClarificationActionResponse{
		Action:  req.Action,
		Success: true,
	}

	// Load the target round for item manipulation.
	rounds, err := workshop.LoadRounds(itemDir)
	if err != nil {
		apierr.MapError(w, "[backlog] clarification-action", apierr.Internal("failed to load rounds"))
		return
	}

	var targetRound *workshop.Round
	for i := range rounds {
		if rounds[i].RoundNum == thread.RoundNumber {
			targetRound = &rounds[i]
			break
		}
	}

	switch req.Action {
	case "got_it":
		thread.Status = "resolved"
		// Attach context note to the decision item if available.
		if thread.LatestImpact != nil && thread.LatestImpact.ContextNote != "" && targetRound != nil {
			for i := range targetRound.Items {
				if targetRound.Items[i].ID == thread.ItemID {
					note := thread.LatestImpact.ContextNote
					targetRound.Items[i].ContextNote = &note
					h.saveRound(itemDir, targetRound)
					break
				}
			}
		}
		resp.Message = "Clarification dismissed. Context note attached."

	case "update_decision":
		if targetRound == nil {
			apierr.MapError(w, "[backlog] clarification-action", apierr.NotFound("target round not found"))
			return
		}
		updated := false
		for i := range targetRound.Items {
			if targetRound.Items[i].ID == thread.ItemID {
				// Apply suggested update or manual JSON.
				if req.UpdatedItemJson != nil && *req.UpdatedItemJson != "" {
					var patch workshop.Item
					if err := json.Unmarshal([]byte(*req.UpdatedItemJson), &patch); err == nil {
						if patch.Topic != "" {
							targetRound.Items[i].Topic = patch.Topic
						}
						if patch.Context != "" {
							targetRound.Items[i].Context = patch.Context
						}
						if len(patch.Options) > 0 {
							targetRound.Items[i].Options = patch.Options
						}
					}
				} else if thread.LatestImpact != nil && thread.LatestImpact.SuggestedUpdate != "" {
					targetRound.Items[i].Context = thread.LatestImpact.SuggestedUpdate
				}
				// Attach context note.
				if thread.LatestImpact != nil && thread.LatestImpact.ContextNote != "" {
					note := thread.LatestImpact.ContextNote
					targetRound.Items[i].ContextNote = &note
				}
				// Clear selection so user re-answers with updated framing.
				targetRound.Items[i].Selected = nil
				targetRound.Items[i].Freeform = nil
				updated = true
				break
			}
		}
		if !updated {
			apierr.MapError(w, "[backlog] clarification-action", apierr.NotFound("decision item not found in round"))
			return
		}
		h.saveRound(itemDir, targetRound)
		thread.Status = "resolved"
		resp.Message = "Decision updated."

	case "remove_decision":
		if targetRound == nil {
			apierr.MapError(w, "[backlog] clarification-action", apierr.NotFound("target round not found"))
			return
		}
		filtered := make([]workshop.Item, 0, len(targetRound.Items))
		for _, item := range targetRound.Items {
			if item.ID != thread.ItemID {
				filtered = append(filtered, item)
			}
		}
		targetRound.Items = filtered
		h.saveRound(itemDir, targetRound)
		thread.Status = "resolved"
		resp.Message = "Decision removed from round."

	case "invalidate_round":
		if targetRound == nil {
			apierr.MapError(w, "[backlog] clarification-action", apierr.NotFound("target round not found"))
			return
		}
		// Delete clarification files for this round.
		_ = workshop.DeleteClarificationsForRound(itemDir, thread.RoundNumber)
		// Delete the round and renumber using the existing workshop function.
		if _, delErr := workshop.DeleteRoundAndRenumber(itemDir, thread.RoundNumber); delErr != nil {
			log.Printf("[backlog] clarification-action: delete round failed: %v", delErr)
		}
		// Spawn a new workshop round.
		item, loadErr := h.store.LoadItem(kind, name)
		if loadErr == nil {
			actionCtx := agentactivity.WithSpec(r.Context(), agentactivity.Spec{
				OwnerType:   agentactivity.OwnerBacklog,
				OwnerKind:   string(kind),
				OwnerName:   item.Name,
				OwnerTitle:  item.Title,
				Purpose:     "research",
				RequestedBy: "swarm-manager",
				Metadata: map[string]string{
					"entrypoint": "backlog.clarification.invalidate_round",
				},
			})
			runResult, spawnErr := h.spawnWorkshopForClarification(actionCtx, kind, item, itemDir)
			if spawnErr == nil {
				resp.RunId = &runResult.RunID
				resp.TaskId = &runResult.TaskID
			} else {
				log.Printf("[backlog] clarification-action: spawn workshop failed: %v", spawnErr)
			}
		}
		thread.Status = "resolved"
		resp.Message = "Round invalidated and new workshop round triggered."

	default:
		apierr.MapError(w, "[backlog] clarification-action", apierr.BadRequest("unknown action: %s", req.Action))
		return
	}

	// Emit analytics events.
	entityID := string(kind) + "/" + name
	if h.eventLogger != nil {
		msgCount := len(thread.Messages)
		impactLevel := "unparsed"
		if thread.LatestImpact != nil {
			impactLevel = thread.LatestImpact.Level
		}
		h.eventLogger.EmitClarificationResolved(entityID, thread.RoundNumber, thread.ItemID, msgCount, impactLevel)
		h.eventLogger.EmitClarificationAction(entityID, thread.RoundNumber, thread.ItemID, req.Action)
	}

	thread.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := workshop.SaveClarification(itemDir, thread); err != nil {
		log.Printf("[backlog] clarification-action: save failed: %v", err)
	}

	if err := httputil.ProtoJSONWithStatus(w, http.StatusOK, resp); err != nil {
		log.Printf("[backlog] clarification-action: failed to write response: %v", err)
	}
}
