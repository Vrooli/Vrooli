/**
 * Summary and pending questions domain types.
 */

import type { BacklogKind } from "./backlog";
import type { ReviewStatus } from "./review";
import type { DecisionOption, ReadinessDimension } from "./workshop";

/**
 * Maturity/readiness data for a single backlog item (from maturity-summary endpoint).
 */
export interface MaturityItemSummary {
  kind: BacklogKind;
  name: string;
  title: string;
  rounds_completed: number;
  raw_scores: Record<ReadinessDimension, number>;
  effective_scores: Record<ReadinessDimension, number>;
  ready: boolean;
  pending_items: number;
  pending_synthesis: boolean;
  has_plan: boolean;
}

/**
 * Response from the maturity-summary endpoint.
 */
export interface MaturitySummaryResponse {
  items: MaturityItemSummary[];
}

/**
 * Combined backlog summary response (maturity + pending questions).
 */
export interface BacklogSummaryResponse {
  maturity: MaturitySummaryResponse;
  pending_questions: PendingQuestionsResponse;
}

// ============================================================================
// Pending Questions Domain (inline question stepper)
// ============================================================================

/**
 * Source of a pending question — either a workshop decision or a target/requirement review.
 */
export type PendingQuestionSource = "workshop" | "review";

/**
 * Type of review item within a pending question.
 */
export type PendingReviewItemType = "target" | "requirement";

/**
 * A single pending question from the pending-questions endpoint.
 * Unifies workshop decisions and unreviewed targets/requirements.
 */
export interface PendingQuestion {
  id: string;
  source: PendingQuestionSource;
  item_kind: BacklogKind;
  item_name: string;
  // Workshop decision fields
  topic?: string;
  text?: string;
  context?: string;
  options?: DecisionOption[];
  selected?: string | null;
  freeform?: string | null;
  notes?: string | null;
  round_number?: number;
  clarification_id?: string;
  context_note?: string;
  // Review fields
  title?: string;
  description?: string;
  criticality?: string;
  review_status?: ReviewStatus;
  review_comment?: string;
  review_type?: PendingReviewItemType;
  module_id?: string;
}

/**
 * Pending questions grouped by backlog item.
 */
export interface PendingQuestionsItem {
  kind: BacklogKind;
  name: string;
  questions: PendingQuestion[];
}

/**
 * Response from the pending-questions endpoint.
 */
export interface PendingQuestionsResponse {
  items: PendingQuestionsItem[];
}
