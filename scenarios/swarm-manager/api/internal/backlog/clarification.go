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
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/workshop"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// CreateClarification starts a new clarification thread for a workshop decision.
//
// POST /api/v1/backlog/{kind}/{name}/workshop/clarification
// clarificationInput is the parsed CreateClarification payload, normalized
// across the JSON and multipart/form-data request shapes.
type clarificationInput struct {
	roundNumber   int32
	itemID        string
	message       string
	attachmentIDs []string
}

// parseClarificationInput decodes a CreateClarification request from either a
// multipart/form-data body (the file-upload UI path, which also persists
// attachments) or a protojson body. On any malformed input it writes the
// appropriate error response and returns ok=false.
func (h *Handler) parseClarificationInput(w http.ResponseWriter, r *http.Request, kind BacklogKind, name string) (clarificationInput, bool) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			apierr.MapError(w, "[backlog] clarification-create", apierr.BadRequest("invalid multipart form"))
			return clarificationInput{}, false
		}
		rn, convErr := strconv.Atoi(r.FormValue("round_number"))
		if convErr != nil || rn < 1 || rn > math.MaxInt32 {
			apierr.MapError(w, "[backlog] clarification-create", apierr.BadRequest("round_number is required and must be between 1 and 2147483647"))
			return clarificationInput{}, false
		}
		itemID := strings.TrimSpace(r.FormValue("item_id"))
		if itemID == "" {
			apierr.MapError(w, "[backlog] clarification-create", apierr.BadRequest("item_id is required"))
			return clarificationInput{}, false
		}
		// Save attached files and collect attachment IDs.
		savedIDs, saveErr := h.saveClarificationAttachments(h.store.ItemDir(kind, name), r)
		if saveErr != nil {
			apierr.MapError(w, "[backlog] clarification-create", apierr.Internal("failed to save attachments"))
			return clarificationInput{}, false
		}
		return clarificationInput{
			roundNumber:   int32(rn), // #nosec G109 -- rn is bounded to [1, math.MaxInt32] above; the int32 conversion cannot overflow.
			itemID:        itemID,
			message:       strings.TrimSpace(r.FormValue("message")),
			attachmentIDs: savedIDs,
		}, true
	}

	var req apipb.CreateClarificationRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		apierr.MapError(w, "[backlog] clarification-create", apierr.BadRequest("invalid request body"))
		return clarificationInput{}, false
	}
	if !httputil.ValidateProtoRequest(w, "[backlog] clarification-create", "invalid request body", &req) {
		return clarificationInput{}, false
	}
	return clarificationInput{
		roundNumber:   req.RoundNumber,
		itemID:        req.ItemId,
		message:       req.Message,
		attachmentIDs: req.AttachmentIds,
	}, true
}

// resolveClarificationDecision loads the workshop rounds for itemDir and
// returns the decision item identified by (roundNumber, itemID). It writes the
// appropriate error response and returns ok=false when the round or item is
// missing, or when the item is not a decision (clarification is decision-only).
func (h *Handler) resolveClarificationDecision(w http.ResponseWriter, itemDir string, roundNumber int32, itemID string) (*workshop.Round, *workshop.Item, bool) {
	rounds, err := workshop.LoadRounds(itemDir)
	if err != nil {
		apierr.MapError(w, "[backlog] clarification-create", apierr.Internal("failed to load workshop rounds"))
		return nil, nil, false
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
		return nil, nil, false
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
		return nil, nil, false
	}
	if targetItem.Type != "decision" {
		apierr.MapError(w, "[backlog] clarification-create", apierr.BadRequest("clarification is only supported for decision items"))
		return nil, nil, false
	}
	return targetRound, targetItem, true
}

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
	in, ok := h.parseClarificationInput(w, r, kind, name)
	if !ok {
		return
	}
	roundNumber, itemID, message, attachmentIDs := in.roundNumber, in.itemID, in.message, in.attachmentIDs

	// Verify round and item exist.
	itemDir := h.store.ItemDir(kind, name)
	targetRound, targetItem, ok := h.resolveClarificationDecision(w, itemDir, roundNumber, itemID)
	if !ok {
		return
	}

	userMessage := strings.TrimSpace(message)
	if userMessage == "" {
		userMessage = "Please explain this decision in detail — what it means, what each option implies, and what the practical consequences are."
	}

	// Commit the thread (identity + user message + attachments) BEFORE starting
	// the async clarification operation, so the operator's turn survives a start
	// failure or retry. The assistant turn is written later by the
	// start-clarification action handler when the operation's round completes.
	now := time.Now().UTC().Format(time.RFC3339)
	thread := &workshop.ClarificationThread{
		ID:          uuid.New().String(),
		RoundNumber: int(roundNumber),
		ItemID:      itemID,
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

	// Link the clarification to the decision item.
	threadID := thread.ID
	targetItem.ClarificationID = &threadID
	if err := h.saveRoundItem(itemDir, targetRound, targetItem); err != nil {
		slog.Warn("failed to link clarification to item", "err", err)
	}

	// Start the clarification operation, forwarding the operator's question and the
	// decision topic as typed caller context so the clarify mode's providers steer
	// the first turn (the committed thread is the durable record; these inputs are
	// the run-start steering). On a start failure the thread persists (identity +
	// user message + attachments) for retry.
	clarInputs := map[string]any{}
	putCallerString(clarInputs, "USER_QUESTION", userMessage)
	decisionTopic := strings.TrimSpace(targetItem.Topic)
	if decisionTopic == "" {
		decisionTopic = strings.TrimSpace(targetItem.Text)
	}
	putCallerString(clarInputs, "DECISION_TOPIC", decisionTopic)
	handle, err := h.invokeItemOperation(r.Context(), kind, item.Name, agentops.OpClarificationStart, "clarification-start-"+thread.ID, clarInputs)
	if err != nil {
		mapClarificationInvokeError(w, "clarification-create", err)
		return
	}
	// Correlate the thread to the live run so the completion handler finds it.
	thread.RunID = handle.RunID
	thread.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := workshop.SaveClarification(itemDir, thread); err != nil {
		slog.Error("clarification run correlation save failed", "err", err)
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

	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, &apipb.CreateClarificationResponse{
		Thread: clarificationThreadToProto(thread),
	}); err != nil {
		slog.Error("failed to write clarification-create response", "err", err)
	}
}

// mapClarificationInvokeError classifies a runner Invoke error for a
// clarification entrypoint into the API error the legacy spawn path returned.
func mapClarificationInvokeError(w http.ResponseWriter, action string, err error) {
	tag := "[backlog] " + action
	switch mapInvokeError(err).kind {
	case invokeUnavailable:
		apierr.MapError(w, tag, apierr.Unavailable("agent-manager is not available"))
	case invokeBusy:
		apierr.MapError(w, tag, apierr.Conflict("an agent is already active for this backlog item"))
	default:
		apierr.MapError(w, tag, apierr.Internal("failed to start clarification operation"))
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

	// Append the operator's follow-up (commit-before-async) so it survives a start
	// failure or retry.
	now := time.Now().UTC().Format(time.RFC3339)
	thread.Messages = append(thread.Messages, workshop.ClarificationMessage{
		Role:          "user",
		Content:       continueMessage,
		CreatedAt:     now,
		AttachmentIDs: continueAttachmentIDs,
	})
	thread.UpdatedAt = now
	turnKey := fmt.Sprintf("clarification-continue-%s-%d", thread.ID, len(thread.Messages))
	if err := workshop.SaveClarification(itemDir, thread); err != nil {
		apierr.MapError(w, "[backlog] clarification-continue", apierr.Internal("failed to save clarification"))
		return
	}

	// Start the follow-up clarification operation, forwarding the operator's message
	// as the required USER_MESSAGE caller input (the continue contract declares it
	// required, so the runner fails closed on an empty message). The committed thread
	// is the durable record; the assistant turn is written by the resolve-
	// clarification action handler on completion.
	continueInputs := map[string]any{}
	putCallerString(continueInputs, "USER_MESSAGE", continueMessage)
	handle, err := h.invokeItemOperation(r.Context(), kind, name, agentops.OpClarificationContinue, turnKey, continueInputs)
	if err != nil {
		mapClarificationInvokeError(w, "clarification-continue", err)
		return
	}
	thread.RunID = handle.RunID
	if err := workshop.SaveClarification(itemDir, thread); err != nil {
		slog.Warn("clarification-continue run correlation save failed", "err", err)
	}

	if err := httputil.ProtoJSONWithStatus(w, http.StatusOK, &apipb.ContinueClarificationResponse{
		Thread: clarificationThreadToProto(thread),
	}); err != nil {
		slog.Error("failed to write clarification-continue response", "err", err)
	}
}
