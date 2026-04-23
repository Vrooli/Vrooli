/**
 * Initiative Review Types
 *
 * Wire-level shapes for the initiative-scoped review surface that gates an
 * initiative's transition from `review_pending` to a terminal state. Mirrors
 * `api/internal/initiativereview/types.go` plus the shared `review.Round`.
 *
 * Separate from the per-item review types in `services/review-service.ts`
 * because the initiative surface uses a distinct verdict vocabulary
 * (accept/fail/followup) and a dedicated decisions audit log.
 */

/** User's terminal verdict on an initiative. */
export type InitiativeReviewVerdict = "accept" | "fail" | "followup";

/**
 * Round status reuses the shared review lifecycle (pending / gathering /
 * complete / failed). Classification is round-level and free-form at the
 * schema level; agents emit `delivered | partial | failed`.
 */
export type InitiativeReviewRoundStatus = "pending" | "gathering" | "complete" | "failed";

/** Classification emitted by the initiative-review agent. */
export type InitiativeReviewClassification = "delivered" | "partial" | "failed";

export interface InitiativeReviewRound {
  round: number;
  generated_at: string;
  execution_id?: string;
  status: InitiativeReviewRoundStatus;
  failure_reason?: string;
  agent_assessment?: string;
  classification?: string;
  notes?: string[];
  evidence: Array<Record<string, unknown>>;
  run_id?: string;
}

/**
 * One entry in the decisions audit log. Persisted under
 * `initiatives/{name}/review/decisions/` every time the user decides a round.
 */
export interface InitiativeReviewDecision {
  verdict: InitiativeReviewVerdict;
  status: string;
  rationale?: string;
  decided_by?: string;
  decided_at: string;
  prior_status: string;
  round?: number;
}

/** Response shape of `POST .../review/trigger`. */
export interface InitiativeReviewTriggerResult {
  started: boolean;
  reason?: string;
  round?: number;
  run_id?: string;
}

/** Response shape of `POST .../review/decide`. */
export interface InitiativeReviewDecideResponse {
  initiative: string;
  verdict: InitiativeReviewVerdict;
  status: string;
  rationale?: string;
  decided_at: string;
}
