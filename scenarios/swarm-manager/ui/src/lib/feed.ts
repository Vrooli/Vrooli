/**
 * Unified Action Feed
 *
 * Merges captures and backlog items into a single prioritized feed.
 * Captures appear at top, attention items (needing user input) are boosted,
 * and normal items follow their standard priority ordering.
 * Items blocked by incomplete dependencies are demoted below actionable items.
 *
 * DOC: docs/concepts/ARCHITECTURE.md#unified-feed
 */

import type { BacklogItem, BacklogStatus, Capture } from "../types";

export type AttentionReason =
  | { kind: "pending-decisions"; count: number }
  | { kind: "plan-ready" }
  | { kind: "research-complete" };

export type FeedItem =
  | { type: "capture"; capture: Capture }
  | { type: "attention"; item: BacklogItem; reasons: AttentionReason[] }
  | { type: "backlog"; item: BacklogItem };

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

/** Statuses that mean a dependency is not yet planned — blocking downstream items. */
const BLOCKING_DEP_STATUSES = new Set<BacklogStatus>(["backlog", "researching"]);

/**
 * Check whether a backlog item is blocked by any of its dependencies.
 */
function isBlockedByDeps(item: BacklogItem, itemsByKey: Map<string, BacklogItem>): boolean {
  if (!item.dependsOn || item.dependsOn.length === 0) return false;
  return item.dependsOn.some((dep) => {
    const depItem = itemsByKey.get(dep);
    return depItem && BLOCKING_DEP_STATUSES.has(depItem.status);
  });
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
    if (feedback.pendingDecisions > 0) {
      reasons.push({ kind: "pending-decisions", count: feedback.pendingDecisions });
    }
  }

  const maturity = maturityMap.get(key);
  if (maturity?.ready) {
    reasons.push({ kind: "plan-ready" });
  }

  if (item.status === "researching") {
    reasons.push({ kind: "research-complete" });
  }

  return reasons;
}

/** Priority penalty applied to items blocked by incomplete dependencies. */
const BLOCKED_PENALTY = 100;

/**
 * Compute a numeric priority for sorting. Lower = higher in the feed.
 * Blocked items receive a large penalty so actionable items always appear first.
 */
function computeFeedPriority(entry: FeedItem, blockedKeys: Set<string>): number {
  let base: number;
  switch (entry.type) {
    case "capture":
      return entry.capture.status === "classifying" ? -2 :
             entry.capture.status === "failed" ? -1 : 0;
    case "attention":
      base = Math.max(entry.item.priority - 2, 0);
      break;
    case "backlog":
      base = entry.item.priority;
      break;
  }
  const key = `${entry.item.kind}/${entry.item.name}`;
  return blockedKeys.has(key) ? base + BLOCKED_PENALTY : base;
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

/** Statuses excluded from the feed by default (hidden unless showFinished is true). */
const FINISHED_STATUSES = new Set(["archived"]);

/**
 * Build the unified action feed from captures and backlog items.
 */
export function buildFeed(
  captures: Capture[],
  backlogItems: BacklogItem[],
  feedbackItems: FeedbackItem[],
  maturityItems: MaturityItem[],
  options?: { showFinished?: boolean },
): FeedItem[] {
  const feedbackMap = new Map<string, FeedbackItem>();
  for (const item of feedbackItems) {
    feedbackMap.set(`${item.kind}/${item.name}`, item);
  }

  const maturityMap = new Map<string, MaturityItem>();
  for (const item of maturityItems) {
    maturityMap.set(`${item.kind}/${item.name}`, item);
  }

  // Build a lookup for dependency blocking.
  const itemsByKey = new Map<string, BacklogItem>();
  for (const item of backlogItems) {
    itemsByKey.set(`${item.kind}/${item.name}`, item);
  }

  const blockedKeys = new Set<string>();
  for (const item of backlogItems) {
    if (isBlockedByDeps(item, itemsByKey)) {
      blockedKeys.add(`${item.kind}/${item.name}`);
    }
  }

  const feed: FeedItem[] = [];

  // Add captures.
  for (const capture of captures) {
    feed.push({ type: "capture", capture });
  }

  // Add backlog items, classifying as attention or normal.
  // Exclude finished items (completed/failed/archived) unless explicitly requested.
  const includeFinished = options?.showFinished ?? false;
  for (const item of backlogItems) {
    if (!includeFinished && FINISHED_STATUSES.has(item.status)) continue;
    const reasons = getAttentionReasons(item, feedbackMap, maturityMap);
    if (reasons.length > 0) {
      feed.push({ type: "attention", item, reasons });
    } else {
      feed.push({ type: "backlog", item });
    }
  }

  // Sort by feed priority (ascending), then by timestamp (descending for recency).
  feed.sort((a, b) => {
    const priorityDiff = computeFeedPriority(a, blockedKeys) - computeFeedPriority(b, blockedKeys);
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
