/**
 * Unified Action Feed
 *
 * Merges captures and backlog items into a single prioritized feed.
 * Captures appear at top, attention items (needing user input) are boosted,
 * and normal items follow their standard priority ordering.
 */

import type { BacklogItem, Capture } from "../types";

export type AttentionReason =
  | { kind: "unanswered-questions"; count: number }
  | { kind: "pending-suggestions"; count: number }
  | { kind: "unsynthesized"; count: number }
  | { kind: "research-complete" };

export type FeedItem =
  | { type: "capture"; capture: Capture }
  | { type: "attention"; item: BacklogItem; reasons: AttentionReason[] }
  | { type: "backlog"; item: BacklogItem };

export interface FeedbackItem {
  kind: string;
  name: string;
  unansweredQuestions: number;
  pendingSuggestions: number;
}

export interface MaturityItem {
  kind: string;
  name: string;
  questionsNewOrUpdated: number;
  suggestionsNewOrUpdated: number;
}

/**
 * Compute attention reasons for a backlog item based on feedback and maturity data.
 */
function getAttentionReasons(
  item: BacklogItem,
  feedbackMap: Map<string, FeedbackItem>,
  maturityMap: Map<string, MaturityItem>,
): AttentionReason[] {
  const reasons: AttentionReason[] = [];
  const key = `${item.kind}/${item.name}`;

  const feedback = feedbackMap.get(key);
  if (feedback) {
    if (feedback.unansweredQuestions > 0) {
      reasons.push({ kind: "unanswered-questions", count: feedback.unansweredQuestions });
    }
    if (feedback.pendingSuggestions > 0) {
      reasons.push({ kind: "pending-suggestions", count: feedback.pendingSuggestions });
    }
  }

  const maturity = maturityMap.get(key);
  if (maturity) {
    const unsynthesized = (maturity.questionsNewOrUpdated ?? 0) + (maturity.suggestionsNewOrUpdated ?? 0);
    if (unsynthesized > 0) {
      reasons.push({ kind: "unsynthesized", count: unsynthesized });
    }
  }

  if (item.status === "researching") {
    reasons.push({ kind: "research-complete" });
  }

  return reasons;
}

/**
 * Compute a numeric priority for sorting. Lower = higher in the feed.
 */
function computeFeedPriority(entry: FeedItem): number {
  switch (entry.type) {
    case "capture":
      return entry.capture.status === "classifying" ? -2 :
             entry.capture.status === "failed" ? -1 : 0;
    case "attention":
      return Math.max(entry.item.priority - 2, 0);
    case "backlog":
      return entry.item.priority;
  }
}

/**
 * Get a secondary sort key (timestamp) for items with the same priority.
 */
function getSortTimestamp(entry: FeedItem): number {
  switch (entry.type) {
    case "capture":
      return new Date(entry.capture.created).getTime();
    case "attention":
    case "backlog":
      return new Date(entry.item.updated).getTime();
  }
}

/**
 * Build the unified action feed from captures and backlog items.
 */
export function buildFeed(
  captures: Capture[],
  backlogItems: BacklogItem[],
  feedbackItems: FeedbackItem[],
  maturityItems: MaturityItem[],
): FeedItem[] {
  const feedbackMap = new Map<string, FeedbackItem>();
  for (const item of feedbackItems) {
    feedbackMap.set(`${item.kind}/${item.name}`, item);
  }

  const maturityMap = new Map<string, MaturityItem>();
  for (const item of maturityItems) {
    maturityMap.set(`${item.kind}/${item.name}`, item);
  }

  const feed: FeedItem[] = [];

  // Add captures.
  for (const capture of captures) {
    feed.push({ type: "capture", capture });
  }

  // Add backlog items, classifying as attention or normal.
  for (const item of backlogItems) {
    const reasons = getAttentionReasons(item, feedbackMap, maturityMap);
    if (reasons.length > 0) {
      feed.push({ type: "attention", item, reasons });
    } else {
      feed.push({ type: "backlog", item });
    }
  }

  // Sort by feed priority (ascending), then by timestamp (descending for recency).
  feed.sort((a, b) => {
    const priorityDiff = computeFeedPriority(a) - computeFeedPriority(b);
    if (priorityDiff !== 0) return priorityDiff;
    return getSortTimestamp(b) - getSortTimestamp(a);
  });

  return feed;
}

/**
 * Count items that need attention (captures + attention backlog items).
 */
export function countActionableItems(feed: FeedItem[]): number {
  return feed.filter((item) => item.type === "capture" || item.type === "attention").length;
}
