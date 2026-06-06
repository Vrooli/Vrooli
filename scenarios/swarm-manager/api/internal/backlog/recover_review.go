// recover-review: the in-band exit for an item stranded in a review-gated
// status (`in_review` / `review_pending`) with no live review round behind it.
//
// The review state machine assumes the `in_review → review_pending` flip always
// eventually fires when a review-agent run completes. When work is done
// out-of-band, a review run dies, or an item is marked `in_review` prematurely,
// that flip never happens and the item is stuck: PATCH is rejected by the
// review-gate guard, review-decide requires `review_pending`, and retry only
// reopens terminal items. This endpoint is the guarded escape hatch — it routes
// an orphaned item to `review_pending` (decide it honestly) or back to
// `backlog` (re-do never-started work), refusing while a real review is live.
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
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/storage"
)

// ReviewRoundInspector reports whether a backlog item has a review round that is
// still actively gathering. Injected (rather than importing the review package,
// which imports backlog) so recover-review can refuse to short-circuit a
// legitimate in-flight review.
//
// seam: backlog.ReviewRoundInspector
type ReviewRoundInspector interface {
	HasLiveReviewRound(kind, name string) bool
}

// SetReviewRoundInspector wires the review-round liveness seam used by the
// recover-review guard. Nil leaves recovery ungated (no live-round check).
func (h *Handler) SetReviewRoundInspector(i ReviewRoundInspector) {
	h.reviewRoundInspector = i
}

// RecoverReviewRequest is the JSON body for the recover-review endpoint.
//
// To selects the recovery target: "review_pending" (default) surfaces the item
// for an honest terminal decision via review-decide; "backlog" returns
// never-started work so it can be planned and executed again.
type RecoverReviewRequest struct {
	To        string `json:"to,omitempty"`
	Rationale string `json:"rationale,omitempty"`
	DecidedBy string `json:"decided_by,omitempty"`
}

// RecoverReviewResponse echoes the resulting item and transition.
type RecoverReviewResponse struct {
	Item        *BacklogItem `json:"item"`
	PriorStatus string       `json:"prior_status"`
	Status      string       `json:"status"`
	Reason      string       `json:"reason,omitempty"`
	RecoveredAt string       `json:"recovered_at"`
}

// recoverReviewRecord is the on-disk audit record for a recovery, stored under
// `review/decisions/{ts}-recover.json` alongside review-decide and retry
// records so the full lifecycle audit lives in one place.
type recoverReviewRecord struct {
	Decision    string `json:"decision"` // "recover"
	Status      string `json:"status"`   // new status (review_pending | backlog)
	PriorStatus string `json:"prior_status"`
	Reason      string `json:"reason,omitempty"`
	DecidedBy   string `json:"decided_by"`
	DecidedAt   string `json:"decided_at"`
}

// RecoverReview is the HTTP handler for
// POST /api/v1/backlog/{kind}/{name}/recover-review.
//
// Behavior:
//   - 400 if the item is not in a review-gated status (nothing to recover).
//   - 409 if a review round is actively gathering (recovery would short-circuit
//     a legitimate review — let it finish, or wait for it to time out).
//   - On success: flips to the requested target, writes an audit record, emits
//     the status-changed event, and returns 200 with the updated item.
func (h *Handler) RecoverReview(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "recover-review")
	if !ok {
		return
	}

	var req RecoverReviewRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.MapError(w, "[backlog] recover-review", apierr.BadRequest("invalid request body"))
			return
		}
	}

	target, err := recoverTargetStatus(req.To)
	if err != nil {
		apierr.MapError(w, "[backlog] recover-review", apierr.BadRequest("%s", err.Error()))
		return
	}

	decidedBy := strings.TrimSpace(req.DecidedBy)
	if decidedBy == "" {
		decidedBy = "user:recover-review"
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) || os.IsNotExist(err) {
			apierr.MapError(w, "[backlog] recover-review", apierr.NotFound("backlog item not found"))
			return
		}
		slog.Error("recover-review: load item", "kind", kind, "name", name, "err", err)
		apierr.MapError(w, "[backlog] recover-review", apierr.Internal("%s", httputil.TruncateErrorMessage(err, 240)))
		return
	}

	if !IsReviewStatus(item.Status) {
		apierr.MapError(w, "[backlog] recover-review",
			apierr.BadRequest("item is in status %q; recover-review only applies to review-gated items (in_review / review_pending)", item.Status))
		return
	}

	// Refuse to short-circuit a legitimate, in-flight review.
	if h.reviewRoundInspector != nil && h.reviewRoundInspector.HasLiveReviewRound(string(kind), name) {
		apierr.MapError(w, "[backlog] recover-review",
			apierr.Conflict("a review round is actively gathering for this item; let it finish (or wait for it to time out) before recovering"))
		return
	}

	reason := strings.TrimSpace(req.Rationale)
	if reason == "" {
		reason = "manual recovery of an orphaned review-gated item"
	}

	priorStatus := item.Status
	recoveredAt := time.Now().UTC().Format(time.RFC3339)
	if err := h.recoverReviewTo(r.Context(), &item, target, reason, decidedBy, recoveredAt); err != nil {
		slog.Error("recover-review: apply recovery", "kind", kind, "name", name, "err", err)
		apierr.MapError(w, "[backlog] recover-review", apierr.Internal("failed to recover item: %s", err.Error()))
		return
	}

	resp := RecoverReviewResponse{
		Item:        &item,
		PriorStatus: string(priorStatus),
		Status:      string(target),
		Reason:      reason,
		RecoveredAt: recoveredAt,
	}
	if err := httputil.JSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] recover-review", apierr.Internal("failed to encode response"))
	}
}

// RecoverOrphanedReview is the programmatic recovery entry point used by the
// review sweeper. It loads the item, confirms it is still in `in_review`, and
// routes it to `review_pending` with an audit record. It is intentionally
// conservative: it only acts on `in_review` (never `review_pending`, which is
// already human-decidable) so a concurrent manual decision is never clobbered.
func (h *Handler) RecoverOrphanedReview(ctx context.Context, kind, name, reason string) error {
	bk, err := ParseBacklogKind(kind)
	if err != nil {
		return err
	}
	item, err := h.store.LoadItem(bk, name)
	if err != nil {
		return err
	}
	if item.Status != StatusInReview {
		return nil // already advanced; nothing to do
	}
	if strings.TrimSpace(reason) == "" {
		reason = "auto-recovered: orphaned in_review with no live review round"
	} else {
		reason = "auto-recovered: " + reason
	}
	recoveredAt := time.Now().UTC().Format(time.RFC3339)
	return h.recoverReviewTo(ctx, &item, StatusReviewPending, reason, "swarm-manager-review-sweeper", recoveredAt)
}

// recoverReviewTo is the shared core for both the manual endpoint and the
// sweeper: it flips the item to target, persists it, writes the audit record,
// and emits the status-changed event. target must be review_pending or backlog.
func (h *Handler) recoverReviewTo(ctx context.Context, item *BacklogItem, target BacklogStatus, reason, decidedBy, recoveredAt string) error {
	priorStatus := item.Status
	item.Status = target
	item.Updated = recoveredAt
	if err := h.store.SaveItem(*item); err != nil {
		return err
	}

	if writeErr := writeRecoverRecord(h.dataRoot, item.Kind, item.Name, recoverReviewRecord{
		Decision:    "recover",
		Status:      string(target),
		PriorStatus: string(priorStatus),
		Reason:      reason,
		DecidedBy:   decidedBy,
		DecidedAt:   recoveredAt,
	}); writeErr != nil {
		slog.Warn("recover-review: failed to write audit record (status change persisted)",
			"kind", item.Kind, "name", item.Name, "err", writeErr)
	}

	if h.eventLogger != nil {
		h.eventLogger.EmitBacklogStatusChanged(string(item.Kind)+"/"+item.Name, string(priorStatus), string(item.Status))
	}
	h.invalidateAllGraphLenses()
	slog.Info("recovered orphaned review item",
		"kind", item.Kind, "name", item.Name, "from", priorStatus, "to", target, "decided_by", decidedBy)
	_ = ctx
	return nil
}

// recoverTargetStatus validates and normalizes the requested recovery target.
// Empty defaults to review_pending.
func recoverTargetStatus(to string) (BacklogStatus, error) {
	switch strings.ToLower(strings.TrimSpace(to)) {
	case "", string(StatusReviewPending):
		return StatusReviewPending, nil
	case string(StatusBacklog):
		return StatusBacklog, nil
	default:
		return "", fmt.Errorf("invalid recovery target %q: must be %q (default) or %q", to, StatusReviewPending, StatusBacklog)
	}
}

// writeRecoverRecord persists the recovery decision to the item's
// review/decisions folder.
func writeRecoverRecord(rootDir string, kind BacklogKind, name string, rec recoverReviewRecord) error {
	kindDir, ok := backlogKindDirs[kind]
	if !ok {
		return fmt.Errorf("unknown kind: %s", kind)
	}
	decisionsDir := filepath.Join(rootDir, kindDir, name, "review", "decisions")
	if err := os.MkdirAll(decisionsDir, 0o755); err != nil {
		return err
	}
	safeTS := strings.ReplaceAll(strings.ReplaceAll(rec.DecidedAt, ":", ""), "-", "")
	filename := fmt.Sprintf("%s-recover.json", safeTS)
	return storage.WriteJSONAtomic(filepath.Join(decisionsDir, filename), rec)
}
