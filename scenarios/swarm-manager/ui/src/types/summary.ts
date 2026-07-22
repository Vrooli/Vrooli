/**
 * Summary and pending questions domain types.
 */

import type { BacklogKind } from "./backlog";
import type { ReviewStatus } from "./review";
import type { DecisionOption } from "./workshop";
/** Combined backlog summary response. */
export interface BacklogSummaryResponse {
  pending_questions: PendingQuestionsResponse;
}

// ============================================================================
// Pending Questions Domain (inline question stepper)
// ============================================================================

/**
 * Source of a pending question.
 */
export type PendingQuestionSource = "review" | "workshop";

/**
 * Type of review item within a pending question.
 */
export type PendingReviewItemType = "target" | "requirement";

/**
 * A single unreviewed target or requirement from the pending-questions endpoint.
 */
export interface PendingQuestion {
  id: string;
  source: PendingQuestionSource;
  item_kind: BacklogKind;
  item_name: string;
  // Legacy workshop fields remain parseable for archived client state. The
  // current API only returns review questions; Plan Workshop owns live choices.
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
