// Review-decide: the terminal transition for items that have reached the
// review gate. Items in `review_pending` are flipped to `completed`, `failed`,
// or `needs_followup` only through this endpoint — the generic PATCH validator
// rejects status changes when the existing status is in_review/review_pending,
// and the execution system never writes terminal statuses.
//
// This is the user's checkpoint: after the review agent gathers evidence,
// the user inspects the review round and explicitly decides the terminal
// status via this endpoint. The decision (with rationale) is persisted
// alongside the review rounds for audit and future meta-optimization.
package backlog

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/storage"
)

// ReviewDecision is the user's terminal verdict on a review round.
type ReviewDecision string

const (
	ReviewDecisionAccept   ReviewDecision = "accept"
	ReviewDecisionFail     ReviewDecision = "fail"
	ReviewDecisionFollowup ReviewDecision = "followup"
)

// ReviewDecideRequest is the JSON body for the review-decide endpoint.
//
// NoRecord, when true, suppresses the records terminal-status hook (used for
// system-level reverts or fixups where a narrative artifact would be noise).
// Default false: agents and humans should write a record by default; only
// opt out when the work genuinely shouldn't be remembered.
type ReviewDecideRequest struct {
	Decision  ReviewDecision `json:"decision"`
	Rationale string         `json:"rationale,omitempty"`
	DecidedBy string         `json:"decided_by,omitempty"`
	NoRecord  bool           `json:"no_record,omitempty"`
}

// ReviewDecideResponse echoes the decision back plus the resulting status.
// RecordStubID, when non-empty, identifies the stub record auto-created by
// the records terminal-status hook — clients surface it so agents can
// `records edit <id>` to fill the narrative.
type ReviewDecideResponse struct {
	Item         *BacklogItem `json:"item"`
	Decision     string       `json:"decision"`
	Status       string       `json:"status"`
	Rationale    string       `json:"rationale,omitempty"`
	DecidedAt    string       `json:"decided_at"`
	RecordStubID string       `json:"record_stub_id,omitempty"`
}

// reviewDecisionRecord is the on-disk record of a terminal decision. Stored
// under `review/decisions/{timestamp}-{decision}.json` inside the item folder.
type reviewDecisionRecord struct {
	Decision    string `json:"decision"`
	Status      string `json:"status"`
	Rationale   string `json:"rationale,omitempty"`
	DecidedBy   string `json:"decided_by,omitempty"`
	DecidedAt   string `json:"decided_at"`
	PriorStatus string `json:"prior_status"`
}

// decisionToStatus maps a decision to the resulting backlog status.
func decisionToStatus(d ReviewDecision) (BacklogStatus, error) {
	switch d {
	case ReviewDecisionAccept:
		return StatusCompleted, nil
	case ReviewDecisionFail:
		return StatusFailed, nil
	case ReviewDecisionFollowup:
		return StatusNeedsFollowup, nil
	}
	return "", fmt.Errorf("invalid decision %q: must be accept, fail, or followup", string(d))
}

// ReviewDecide is the HTTP handler for POST /api/v1/backlog/{kind}/{name}/review-decide.
func (h *Handler) ReviewDecide(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "review-decide")
	if !ok {
		return
	}

	var req ReviewDecideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[backlog] review-decide", apierr.BadRequest("invalid request body: %s", err.Error()))
		return
	}
	req.Decision = ReviewDecision(strings.ToLower(strings.TrimSpace(string(req.Decision))))
	req.Rationale = strings.TrimSpace(req.Rationale)
	req.DecidedBy = strings.TrimSpace(req.DecidedBy)
	if req.DecidedBy == "" {
		req.DecidedBy = "user"
	}

	targetStatus, err := decisionToStatus(req.Decision)
	if err != nil {
		apierr.MapError(w, "[backlog] review-decide", apierr.BadRequest("%s", err.Error()))
		return
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) || os.IsNotExist(err) {
			apierr.MapError(w, "[backlog] review-decide", apierr.NotFound("backlog item not found"))
			return
		}
		slog.Error("review-decide: load item", "kind", kind, "name", name, "err", err)
		apierr.MapError(w, "[backlog] review-decide", apierr.Internal("%s", httputil.TruncateErrorMessage(err, 240)))
		return
	}

	// Only items awaiting a user decision can be decided. This guards against
	// double-decides and accidental flips from in_progress/in_review states.
	if item.Status != StatusReviewPending {
		apierr.MapError(w, "[backlog] review-decide",
			apierr.BadRequest("item is in status %q; review-decide requires status %q", item.Status, StatusReviewPending))
		return
	}

	priorStatus := item.Status
	decidedAt := time.Now().UTC().Format(time.RFC3339)

	item.Status = targetStatus
	item.Updated = decidedAt
	if err := h.store.SaveItem(item); err != nil {
		slog.Error("review-decide: save item", "kind", kind, "name", name, "err", err)
		apierr.MapError(w, "[backlog] review-decide", apierr.Internal("failed to persist decision"))
		return
	}

	// Persist the decision record. Failure to write the audit record is
	// logged but doesn't roll back the status change — the status is the
	// source of truth, the record is supplementary context.
	if writeErr := writeDecisionRecord(h.dataRoot, kind, name, reviewDecisionRecord{
		Decision:    string(req.Decision),
		Status:      string(targetStatus),
		Rationale:   req.Rationale,
		DecidedBy:   req.DecidedBy,
		DecidedAt:   decidedAt,
		PriorStatus: string(priorStatus),
	}); writeErr != nil {
		slog.Warn("review-decide: failed to write decision record (status change persisted)",
			"kind", kind, "name", name, "err", writeErr)
	}

	if h.eventLogger != nil {
		h.eventLogger.EmitBacklogStatusChanged(string(kind)+"/"+name, string(priorStatus), string(targetStatus))
	}

	// Records soft-prompt: auto-create a stub linked back to this item so the
	// agent or human can fill it via `records edit`. Runs BEFORE
	// itemTerminalHandler so the response payload can include the stub id even
	// if a downstream handler is slow. NoRecord opts out (system reverts /
	// fixups). Errors are swallowed — stub creation must never block or fail
	// the terminal transition.
	var recordStubID string
	if !req.NoRecord && h.recordStubCreator != nil {
		id, err := h.recordStubCreator.CreateBacklogStub(r.Context(), string(kind), name, targetStatus, req.DecidedBy)
		if err != nil {
			slog.Warn("review-decide: record stub creation failed (terminal status persisted)",
				"kind", kind, "name", name, "err", err)
		} else {
			recordStubID = id
		}
	}

	// Notify downstream consumers (e.g., initiative review) that this item
	// reached a terminal status. Runs synchronously; the handler is expected
	// to self-dispatch expensive work. Errors are not surfaced — the
	// terminal decision is authoritative regardless of side-effect outcome.
	if h.itemTerminalHandler != nil {
		h.itemTerminalHandler(r.Context(), string(kind), name, targetStatus)
	}

	resp := ReviewDecideResponse{
		Item:         &item,
		Decision:     string(req.Decision),
		Status:       string(targetStatus),
		Rationale:    req.Rationale,
		DecidedAt:    decidedAt,
		RecordStubID: recordStubID,
	}
	if err := httputil.JSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] review-decide", apierr.Internal("failed to encode response"))
	}
}

// writeDecisionRecord persists the terminal decision to the item's review/decisions
// folder. The filename encodes the timestamp + decision for chronological sort
// and grep-ability in the audit log.
func writeDecisionRecord(rootDir string, kind BacklogKind, name string, rec reviewDecisionRecord) error {
	kindDir, ok := backlogKindDirs[kind]
	if !ok {
		return fmt.Errorf("unknown kind: %s", kind)
	}
	decisionsDir := filepath.Join(rootDir, kindDir, name, "review", "decisions")
	if err := os.MkdirAll(decisionsDir, 0o755); err != nil {
		return err
	}
	// Timestamp with millisecond precision keeps filenames unique even when
	// an agent and user converge on the same second.
	safeTS := strings.ReplaceAll(strings.ReplaceAll(rec.DecidedAt, ":", ""), "-", "")
	filename := fmt.Sprintf("%s-%s.json", safeTS, rec.Decision)
	return storage.WriteJSONAtomic(filepath.Join(decisionsDir, filename), rec)
}
