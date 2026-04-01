// DOC: docs/internal/SEAMS.md#clarification-api
// DOC: docs/reference/api-endpoints.md#clarification
//
// HTTP handlers for workshop decision clarification threads. Each handler
// delegates to the workshop.Clarification* storage functions and the
// agentmanager.Service for agent spawning and run continuation.
package backlog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/promptcatalog"
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
			httputil.NotFound(w, "[backlog] clarification-create", "backlog item not found")
			return
		}
		httputil.InternalError(w, "[backlog] clarification-create", "failed to load backlog item")
		return
	}

	// Parse request — supports both JSON and multipart/form-data (for file attachments).
	var roundNumber int32
	var itemID, message string
	var attachmentIDs []string

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			httputil.BadRequest(w, "[backlog] clarification-create", "invalid multipart form")
			return
		}
		rn, convErr := strconv.Atoi(r.FormValue("round_number"))
		if convErr != nil || rn < 1 {
			httputil.BadRequest(w, "[backlog] clarification-create", "round_number is required and must be >= 1")
			return
		}
		roundNumber = int32(rn)
		itemID = strings.TrimSpace(r.FormValue("item_id"))
		if itemID == "" {
			httputil.BadRequest(w, "[backlog] clarification-create", "item_id is required")
			return
		}
		message = strings.TrimSpace(r.FormValue("message"))

		// Save attached files and collect attachment IDs.
		itemDir := h.store.ItemDir(kind, name)
		savedIDs, saveErr := h.saveClarificationAttachments(itemDir, r)
		if saveErr != nil {
			httputil.InternalError(w, "[backlog] clarification-create", "failed to save attachments")
			return
		}
		attachmentIDs = savedIDs
	} else {
		var req apipb.CreateClarificationRequest
		if err := httputil.DecodeProtoJSON(r, &req); err != nil {
			httputil.BadRequest(w, "[backlog] clarification-create", "invalid request body")
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
		httputil.InternalError(w, "[backlog] clarification-create", "failed to load workshop rounds")
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
		httputil.NotFound(w, "[backlog] clarification-create", fmt.Sprintf("round %d not found", roundNumber))
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
		httputil.BadRequest(w, "[backlog] clarification-create", fmt.Sprintf("item %q not found in round %d", itemID, roundNumber))
		return
	}
	if targetItem.Type != "decision" {
		httputil.BadRequest(w, "[backlog] clarification-create", "clarification is only supported for decision items")
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
		httputil.ServiceUnavailable(w, "[backlog] clarification-create", "agent-manager is not available")
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
		httputil.InternalError(w, "[backlog] clarification-create", "failed to spawn clarification agent")
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
		httputil.InternalError(w, "[backlog] clarification-create", "failed to save clarification thread")
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

	httputil.ProtoJSONWithStatus(w, http.StatusCreated, &apipb.CreateClarificationResponse{
		Thread: clarificationThreadToProto(thread),
	})
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
		httputil.BadRequest(w, "[backlog] clarification-get", "threadId is required")
		return
	}

	itemDir := h.store.ItemDir(kind, name)
	thread, err := workshop.LoadClarificationByID(itemDir, threadID)
	if err != nil {
		httputil.InternalError(w, "[backlog] clarification-get", "failed to load clarification")
		return
	}
	if thread == nil {
		httputil.NotFound(w, "[backlog] clarification-get", "clarification thread not found")
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
			if state.Status == "completed" || state.Status == "success" {
				// Check if a new message was added (agent may write to thread file).
				refreshed, _ := workshop.LoadClarificationByID(itemDir, threadID)
				if refreshed != nil && len(refreshed.Messages) > len(thread.Messages) {
					thread = refreshed
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

	httputil.ProtoJSONWithStatus(w, http.StatusOK, &apipb.GetClarificationResponse{
		Thread: clarificationThreadToProto(thread),
	})
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
		httputil.BadRequest(w, "[backlog] clarification-continue", "threadId is required")
		return
	}

	// Parse request — supports both JSON and multipart/form-data.
	var continueMessage string
	var continueAttachmentIDs []string

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			httputil.BadRequest(w, "[backlog] clarification-continue", "invalid multipart form")
			return
		}
		continueMessage = strings.TrimSpace(r.FormValue("message"))
		if continueMessage == "" {
			httputil.BadRequest(w, "[backlog] clarification-continue", "message is required")
			return
		}
		itemDir := h.store.ItemDir(kind, name)
		savedIDs, saveErr := h.saveClarificationAttachments(itemDir, r)
		if saveErr != nil {
			httputil.InternalError(w, "[backlog] clarification-continue", "failed to save attachments")
			return
		}
		continueAttachmentIDs = savedIDs
	} else {
		var req apipb.ContinueClarificationRequest
		if err := httputil.DecodeProtoJSON(r, &req); err != nil {
			httputil.BadRequest(w, "[backlog] clarification-continue", "invalid request body")
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
		httputil.InternalError(w, "[backlog] clarification-continue", "failed to load clarification")
		return
	}
	if thread == nil {
		httputil.NotFound(w, "[backlog] clarification-continue", "clarification thread not found")
		return
	}
	if thread.Status != "active" {
		httputil.Conflict(w, "[backlog] clarification-continue", "clarification thread is "+thread.Status+", cannot continue")
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
		httputil.InternalError(w, "[backlog] clarification-continue", "failed to save clarification")
		return
	}

	httputil.ProtoJSONWithStatus(w, http.StatusOK, &apipb.ContinueClarificationResponse{
		Thread: clarificationThreadToProto(thread),
	})
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
		httputil.BadRequest(w, "[backlog] clarification-action", "threadId is required")
		return
	}

	var req apipb.ClarificationActionRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		httputil.BadRequest(w, "[backlog] clarification-action", "invalid request body")
		return
	}
	if !httputil.ValidateProtoRequest(w, "[backlog] clarification-action", "invalid request body", &req) {
		return
	}

	itemDir := h.store.ItemDir(kind, name)
	thread, err := workshop.LoadClarificationByID(itemDir, threadID)
	if err != nil {
		httputil.InternalError(w, "[backlog] clarification-action", "failed to load clarification")
		return
	}
	if thread == nil {
		httputil.NotFound(w, "[backlog] clarification-action", "clarification thread not found")
		return
	}

	resp := &apipb.ClarificationActionResponse{
		Action:  req.Action,
		Success: true,
	}

	// Load the target round for item manipulation.
	rounds, err := workshop.LoadRounds(itemDir)
	if err != nil {
		httputil.InternalError(w, "[backlog] clarification-action", "failed to load rounds")
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
			httputil.NotFound(w, "[backlog] clarification-action", "target round not found")
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
			httputil.NotFound(w, "[backlog] clarification-action", "decision item not found in round")
			return
		}
		h.saveRound(itemDir, targetRound)
		thread.Status = "resolved"
		resp.Message = "Decision updated."

	case "remove_decision":
		if targetRound == nil {
			httputil.NotFound(w, "[backlog] clarification-action", "target round not found")
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
			httputil.NotFound(w, "[backlog] clarification-action", "target round not found")
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
		httputil.BadRequest(w, "[backlog] clarification-action", "unknown action: "+req.Action)
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

	httputil.ProtoJSONWithStatus(w, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// clarificationAllowedImageTypes lists Content-Types accepted for clarification attachments.
var clarificationAllowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// saveClarificationAttachments saves uploaded files from a multipart request
// and returns their attachment IDs (relative paths). Follows the same pattern
// as capture attachment storage.
func (h *Handler) saveClarificationAttachments(itemDir string, r *http.Request) ([]string, error) {
	if r.MultipartForm == nil {
		return nil, nil
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		return nil, nil
	}

	attDir := filepath.Join(itemDir, "workshop", "attachments")
	if err := os.MkdirAll(attDir, 0o755); err != nil {
		return nil, fmt.Errorf("create attachment dir: %w", err)
	}

	var ids []string
	for _, fh := range files {
		mediaType, _, _ := mime.ParseMediaType(fh.Header.Get("Content-Type"))
		if !clarificationAllowedImageTypes[mediaType] {
			return nil, fmt.Errorf("unsupported file type: %s", mediaType)
		}

		attID := uuid.New().String()
		ext := filepath.Ext(fh.Filename)
		destName := attID + ext
		destPath := filepath.Join(attDir, destName)

		src, err := fh.Open()
		if err != nil {
			return nil, fmt.Errorf("open uploaded file: %w", err)
		}
		dst, err := os.Create(destPath)
		if err != nil {
			src.Close()
			return nil, fmt.Errorf("create attachment file: %w", err)
		}
		_, copyErr := io.Copy(dst, src)
		src.Close()
		dst.Close()
		if copyErr != nil {
			return nil, fmt.Errorf("write attachment file: %w", copyErr)
		}
		ids = append(ids, filepath.Join("workshop", "attachments", destName))
	}
	return ids, nil
}

// buildClarificationPrompt constructs the prompt for a clarification agent
// by fetching the skill from prompt-manager.
func (h *Handler) buildClarificationPrompt(
	ctx context.Context,
	item BacklogItem,
	itemDir string,
	decision *workshop.Item,
	userQuestion string,
	priorMessages []workshop.ClarificationMessage,
) (string, error) {
	entry, ok := promptcatalog.ResolveBacklogSkill("clarify", string(item.Kind))
	if !ok {
		return "", fmt.Errorf("no prompt catalog entry for mode=clarify kind=%s", item.Kind)
	}

	vars := buildVariableMap(item, itemDir)
	vars["DECISION_TOPIC"] = decision.Topic
	vars["DECISION_CONTEXT"] = decision.Context
	vars["DECISION_OPTIONS"] = workshop.FormatOptionsForPrompt(decision.Options)
	vars["USER_QUESTION"] = userQuestion
	vars["CLARIFICATION_HISTORY"] = workshop.FormatClarificationHistory(priorMessages)

	prompt, err := h.promptClient.ReadSkill(ctx, entry.SkillID, vars, false)
	if err != nil {
		return "", fmt.Errorf("prompt-manager read: %w", err)
	}
	return prompt, nil
}

// saveRound writes a round back to disk.
func (h *Handler) saveRound(itemDir string, round *workshop.Round) {
	data, err := json.MarshalIndent(round, "", "  ")
	if err != nil {
		log.Printf("[backlog] saveRound: marshal error: %v", err)
		return
	}
	path := filepath.Join(itemDir, "workshop", fmt.Sprintf("round-%03d.json", round.RoundNum))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		log.Printf("[backlog] saveRound: write error: %v", err)
	}
}

// saveRoundItem links a clarification to a decision item and saves the round.
func (h *Handler) saveRoundItem(itemDir string, round *workshop.Round, item *workshop.Item) error {
	for i := range round.Items {
		if round.Items[i].ID == item.ID {
			round.Items[i] = *item
			break
		}
	}
	h.saveRound(itemDir, round)
	return nil
}

// spawnWorkshopForClarification spawns a new workshop round after invalidation.
func (h *Handler) spawnWorkshopForClarification(
	ctx context.Context,
	kind BacklogKind,
	item BacklogItem,
	itemDir string,
) (agentmanager.RunResult, error) {
	selection, err := h.fetchResearchPrompt(ctx, item, ResearchModeWorkshop)
	if err != nil {
		return agentmanager.RunResult{}, fmt.Errorf("fetch workshop prompt: %w", err)
	}

	return h.agentService.SpawnBacklog(ctx, agentmanager.BacklogSpawnRequest{
		Kind:        string(kind),
		Name:        item.Name,
		Title:       fmt.Sprintf("Workshop: %s (re-run after clarification)", item.Title),
		Description: selection.Prompt,
		Prompt:      selection.Prompt,
		ScopePath:   ".",
		ProjectRoot: ".",
		CreatedBy:   "swarm-manager",
		Purpose:     "research",
		Environment: map[string]string{
			"VROOLI_SPAWN_SOURCE": string(kind) + "/" + item.Name,
		},
	})
}

// lastAssistantMessage returns the most recent assistant message in a thread.
func lastAssistantMessage(thread *workshop.ClarificationThread) *workshop.ClarificationMessage {
	for i := len(thread.Messages) - 1; i >= 0; i-- {
		if thread.Messages[i].Role == "assistant" {
			return &thread.Messages[i]
		}
	}
	return nil
}

// isTerminalStatus checks if a run status indicates completion.
func isTerminalStatus(status string) bool {
	switch strings.ToLower(status) {
	case "completed", "success", "failed", "error", "cancelled", "canceled":
		return true
	}
	return false
}

// clarificationThreadToProto converts a workshop.ClarificationThread to its
// proto representation. The domain types (ClarificationThread, etc.) live in
// the domain proto package; the API response messages reference them.
func clarificationThreadToProto(t *workshop.ClarificationThread) *domainpb.ClarificationThread {
	if t == nil {
		return nil
	}
	pb := &domainpb.ClarificationThread{
		Id:          t.ID,
		RoundNumber: int32(t.RoundNumber),
		ItemId:      t.ItemID,
		RunId:       t.RunID,
		Status:      t.Status,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
	for _, msg := range t.Messages {
		pb.Messages = append(pb.Messages, &domainpb.ClarificationMessage{
			Role:          msg.Role,
			Content:       msg.Content,
			CreatedAt:     msg.CreatedAt,
			AttachmentIds: msg.AttachmentIDs,
		})
	}
	if t.LatestImpact != nil {
		pb.LatestImpact = &domainpb.ClarificationImpact{
			Level:           t.LatestImpact.Level,
			Reasoning:       t.LatestImpact.Reasoning,
			ContextNote:     t.LatestImpact.ContextNote,
			SuggestedUpdate: t.LatestImpact.SuggestedUpdate,
		}
	}
	return pb
}
