/**
 * GENERATED FILE — DO NOT EDIT.
 *
 * Mirror of the backlog status vocabulary. The source of truth is
 * api/internal/backlogstatus/statuses.go; regenerate with `make gen-status`
 * from the scenario root after changing the status table.
 *
 * Status colors are intentionally NOT generated — they live in
 * types/constants.ts so TypeScript's exhaustiveness check on
 * Record<BacklogStatus, string> forces a new status to be given one.
 */

/** Valid lifecycle states for a backlog item, in lifecycle order. */
export type BacklogStatus =
  /** Proposed by the auto-filer and not yet accepted. Not user-settable:
   * operators accept a suggestion by moving it into the backlog, rather than
   * authoring the suggested state by hand. */
  | "suggested"
  /** Accepted work that has not been shaped yet. */
  | "backlog"
  /** Being investigated before it can be specified. */
  | "researching"
  /** Shaped and queueable. */
  | "ready"
  /** Accepted into the execution queue. Owned by the execution system; an
   * operator PATCH must not fabricate it. */
  | "queued"
  /** A run is active. Owned by the execution system. */
  | "in_progress"
  /** A review round is actively gathering evidence. Set by the review system.
   * Invariant: a code path that cannot start or continue a round must route to
   * review_pending rather than leave an item here — the review sweeper and the
   * recover-review endpoint drain any item stranded in_review with no live
   * round, so this can never become a dead end. */
  | "in_review"
  /** Review finished; awaiting the operator's verdict. Exit via review-decide so
   * the decision carries an audit trail. */
  | "review_pending"
  /** The work was achieved. The only status that counts toward goal progress. */
  | "completed"
  /** The work was attempted and did not land. NOT resolved: failed work may still
   * be retried, so its dependents are genuinely still blocked. */
  | "failed"
  /** Delivered, but more work is needed. A live attention state, not an archive
   * dead end. NOT resolved: dependents are still waiting on the remainder. Do
   * not conflate with execution.StatusNeedsFixup, a run-level state on a
   * different enum. */
  | "needs_followup"
  /** Closed by operator decision: not going to be done, or no longer relevant.
   * Carries no verdict about the work, so unlike the other terminals it needs no
   * run or review round behind it and may be set straight from a planning
   * status. Resolved, because an item nobody will ever finish must stop blocking
   * its dependents. */
  | "dropped";

/** Every status, in lifecycle order. */
export const BACKLOG_STATUSES: readonly BacklogStatus[] = [
  "suggested",
  "backlog",
  "researching",
  "ready",
  "queued",
  "in_progress",
  "in_review",
  "review_pending",
  "completed",
  "failed",
  "needs_followup",
  "dropped",
] as const;

/** Human-readable label for each status. */
export const BACKLOG_STATUS_LABELS: Record<BacklogStatus, string> = {
  suggested: "Suggested",
  backlog: "Backlog",
  researching: "Researching",
  ready: "Ready",
  queued: "Queued",
  in_progress: "In Progress",
  in_review: "In Review",
  review_pending: "Review Pending",
  completed: "Completed",
  failed: "Failed",
  needs_followup: "Needs Follow-up",
  dropped: "Dropped",
};

/**
 * Statuses an operator may set directly via the generic status patch.
 * Execution-owned and review-gated statuses are excluded: the former belong to
 * the execution system, the latter must exit through review-decide so the
 * decision carries an audit trail.
 */
export const USER_SETTABLE_STATUSES: readonly BacklogStatus[] = [
  "backlog",
  "researching",
  "ready",
  "completed",
  "failed",
  "needs_followup",
  "dropped",
] as const;

/**
 * Settled statuses — the item is not coming back without an explicit revival.
 */
export const TERMINAL_STATUSES: readonly BacklogStatus[] = [
  "completed",
  "failed",
  "needs_followup",
  "dropped",
] as const;

/**
 * Statuses meaning nothing depending on the item is still waiting.
 * Note this is NOT the same as completed: dropped work resolves a dependency
 * without having achieved anything, and failed work does not resolve at all.
 */
export const RESOLVED_STATUSES: readonly BacklogStatus[] = [
  "completed",
  "dropped",
] as const;

/**
 * Statuses from which an item can be queued for execution.
 */
export const QUEUEABLE_BACKLOG_STATUSES: readonly BacklogStatus[] = [
  "backlog",
  "researching",
  "ready",
] as const;

/**
 * Statuses owned by the execution system while a run is live.
 */
export const IN_FLIGHT_STATUSES: readonly BacklogStatus[] = [
  "queued",
  "in_progress",
] as const;

/**
 * Statuses where a review round is gathering evidence or awaiting a verdict.
 */
export const REVIEW_STATUSES: readonly BacklogStatus[] = [
  "in_review",
  "review_pending",
] as const;
