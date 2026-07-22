/**
 * Attention signals
 *
 * Computes why a backlog item needs user attention (pending decisions,
 * and research-complete states. Powers the sidebar tab badges, backlog
 * card badges, and the command-post attention counts.
 */

import type { BacklogItem } from "../types";

export type AttentionReason =
  | { kind: "pending-decisions"; count: number }
  | { kind: "research-complete" };

export interface FeedbackItem {
  kind: string;
  name: string;
  pendingDecisions: number;
}

export interface MaturityItem {
  kind: string;
  name: string;
  ready: boolean;
  pendingItems: number;
}

/**
 * Compute attention reasons for a backlog item based on feedback and maturity data.
 */
export function getAttentionReasons(
  item: BacklogItem,
  feedbackMap: Map<string, FeedbackItem>,
  _maturityMap: Map<string, MaturityItem>,
): AttentionReason[] {
  const reasons: AttentionReason[] = [];
  const key = `${item.kind}/${item.name}`;

  const feedback = feedbackMap.get(key);
  if (feedback) {
    if (feedback.pendingDecisions > 0) {
      reasons.push({ kind: "pending-decisions", count: feedback.pendingDecisions });
    }
  }

  if (item.status === "researching") {
    reasons.push({ kind: "research-complete" });
  }

  return reasons;
}
