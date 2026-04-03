// DOC: docs/internal/SEAMS.md#clarification-api
// DOC: docs/reference/api-endpoints.md#clarification
//
// Clarification state management: reading thread state, checking agent
// completion, applying post-clarification actions (got_it, update_decision,
// remove_decision, invalidate_round).
package backlog

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/workshop"
)

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
		slog.Error("failed to write clarification-get response", "err", err)
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
			slog.Error("clarification-action delete round failed", "err", delErr)
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
				slog.Error("clarification-action spawn workshop failed", "err", spawnErr)
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
		slog.Error("clarification-action save failed", "err", err)
	}

	if err := httputil.ProtoJSONWithStatus(w, http.StatusOK, resp); err != nil {
		slog.Error("failed to write clarification-action response", "err", err)
	}
}
