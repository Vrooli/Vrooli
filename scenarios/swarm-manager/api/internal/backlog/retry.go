// Backlog-level retry route. Wraps execution.Service.RetryLatestForBacklog
// and, when the item is in a user-decided terminal state, reopens it via
// ReopenForRetry so the new attempt is reflected in the item's lifecycle.
//
// ReopenForRetry is the *only* path that transitions an item *out* of a
// terminal status. Its mirror — review-decide — is the only path *into* a
// terminal status. The two together form the closed loop documented in
// docs/internal/INVARIANTS.md (Replay/Idempotency).
package backlog

import (
	"context"
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
	"swarm-manager/internal/execution"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/storage"
)

// RetryRequest is the JSON body for the backlog retry endpoint. Note is
// optional and informational (e.g., "fixed agent-manager hesitation bug").
type RetryRequest struct {
	Note string `json:"note,omitempty"`
}

// RetryResponse echoes the new execution and the parent it was spawned from.
type RetryResponse struct {
	Item              *BacklogItem `json:"item"`
	NewExecutionID    string       `json:"new_execution_id"`
	ParentExecutionID string       `json:"parent_execution_id"`
	Status            string       `json:"status"`
}

// retryReopenRecord is the on-disk audit record written when a retry reopens
// a terminal item. Stored under `review/decisions/{ts}-reopen.json` so the
// audit log shows the retry as a deliberate user action rather than an
// unexplained backward transition.
type retryReopenRecord struct {
	Decision       string `json:"decision"`         // "reopen"
	Status         string `json:"status"`           // new status (in_progress)
	PriorStatus    string `json:"prior_status"`     // terminal status being reopened
	NewExecutionID string `json:"new_execution_id"` // the spawned attempt
	DecidedBy      string `json:"decided_by"`       // "user:retry"
	DecidedAt      string `json:"decided_at"`
	Note           string `json:"note,omitempty"`
}

// Retry is the HTTP handler for POST /api/v1/backlog/{kind}/{name}/retry.
//
// Behavior:
//   - 400 if the item has never been executed (no parent to retry from).
//   - 400 if every prior attempt is in flight (nothing terminal to retry).
//   - On success: spawns a new execution, reopens the item to in_progress
//     when prior status was terminal, returns 202 with the new execution id.
func (h *Handler) Retry(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "retry")
	if !ok {
		return
	}

	if h.executionQueuer == nil {
		apierr.MapError(w, "[backlog] retry", apierr.Unavailable("execution service is not configured"))
		return
	}

	var req RetryRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.MapError(w, "[backlog] retry", apierr.BadRequest("invalid request body"))
			return
		}
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) || os.IsNotExist(err) {
			apierr.MapError(w, "[backlog] retry", apierr.NotFound("backlog item not found"))
			return
		}
		slog.Error("retry: load item", "kind", kind, "name", name, "err", err)
		apierr.MapError(w, "[backlog] retry", apierr.Internal("%s", httputil.TruncateErrorMessage(err, 240)))
		return
	}

	newRecord, hasPrior, retryErr := h.executionQueuer.RetryLatestForBacklog(r.Context(), string(kind), name, strings.TrimSpace(req.Note))
	if !hasPrior {
		apierr.MapError(w, "[backlog] retry", apierr.BadRequest("item has no prior execution to retry"))
		return
	}
	if retryErr != nil {
		apierr.MapError(w, "[backlog] retry", retryErr)
		return
	}

	if IsTerminalStatus(item.Status) {
		if err := h.reopenForRetry(r.Context(), &item, newRecord, strings.TrimSpace(req.Note)); err != nil {
			slog.Error("retry: reopen item failed", "kind", kind, "name", name, "err", err)
			// The new execution exists; surfacing the reopen failure to the
			// user is honest about partial state. Stats and the new run will
			// continue regardless.
			apierr.MapError(w, "[backlog] retry", apierr.Internal("retry dispatched but failed to reopen item: %s", err.Error()))
			return
		}
	}

	resp := RetryResponse{
		Item:              &item,
		NewExecutionID:    newRecord.ExecutionID,
		ParentExecutionID: newRecord.ParentExecutionID,
		Status:            string(newRecord.Status),
	}
	w.WriteHeader(http.StatusAccepted)
	if err := httputil.JSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] retry", apierr.Internal("failed to encode response"))
	}
}

// reopenForRetry transitions a terminal item back to in_progress as part of
// a retry flow. Persists the item, writes an audit record, and emits the
// status-changed event. This is the only legitimate writer that moves an
// item *out* of a terminal status.
func (h *Handler) reopenForRetry(_ context.Context, item *BacklogItem, newRecord execution.Record, note string) error {
	if !IsTerminalStatus(item.Status) {
		return fmt.Errorf("reopenForRetry called on non-terminal item (status=%s)", item.Status)
	}
	priorStatus := item.Status
	now := time.Now().UTC().Format(time.RFC3339)
	item.Status = StatusInProgress
	item.Updated = now
	if err := h.store.SaveItem(*item); err != nil {
		return err
	}

	if writeErr := writeRetryReopenRecord(h.rootDir, item.Kind, item.Name, retryReopenRecord{
		Decision:       "reopen",
		Status:         string(StatusInProgress),
		PriorStatus:    string(priorStatus),
		NewExecutionID: newRecord.ExecutionID,
		DecidedBy:      "user:retry",
		DecidedAt:      now,
		Note:           note,
	}); writeErr != nil {
		slog.Warn("retry: failed to write reopen audit record (status change persisted)",
			"kind", item.Kind, "name", item.Name, "err", writeErr)
	}

	if h.eventLogger != nil {
		h.eventLogger.EmitBacklogStatusChanged(string(item.Kind)+"/"+item.Name, string(priorStatus), string(item.Status))
	}
	h.invalidateAllGraphLenses()
	return nil
}

// writeRetryReopenRecord persists the reopen decision to the item's
// review/decisions folder, alongside review-decide records so the full audit
// trail is in one place.
func writeRetryReopenRecord(rootDir string, kind BacklogKind, name string, rec retryReopenRecord) error {
	kindDir, ok := backlogKindDirs[kind]
	if !ok {
		return fmt.Errorf("unknown kind: %s", kind)
	}
	decisionsDir := filepath.Join(rootDir, kindDir, name, "review", "decisions")
	if err := os.MkdirAll(decisionsDir, 0o755); err != nil {
		return err
	}
	safeTS := strings.ReplaceAll(strings.ReplaceAll(rec.DecidedAt, ":", ""), "-", "")
	filename := fmt.Sprintf("%s-reopen.json", safeTS)
	return storage.WriteJSONAtomic(filepath.Join(decisionsDir, filename), rec)
}
