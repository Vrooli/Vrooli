/**
 * Unified Action Feed
 *
 * Merges captures and backlog items into a single prioritized feed.
 * Captures appear at top, attention items (needing user input) are boosted,
 * and normal items follow their standard priority ordering.
 *
 * Dependencies are respected via topological-depth sorting: items whose
 * dependencies are still incomplete always appear below those dependencies.
 * See `dependency-sort.ts` for the sort-blocking vs queue-blocking distinction.
 *
 * DOC: docs/concepts/ARCHITECTURE.md#unified-feed
 */

import type { BacklogItem, Capture } from "../types";
import { dependencyAwareSort } from "./dependency-sort";

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

/**
 * Compute attention reasons for a backlog item based on feedback and maturity data.
 */
export function getAttentionReasons(
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

/**
 * Compute a numeric priority for sorting. Lower = higher in the feed.
 * Used as the tiebreaker within the same dependency depth.
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

/** Check if an item is archived (excluded from the feed by default unless showFinished is true). */
function isFinished(item: BacklogItem): boolean {
  return item.archivedAt != null;
}

/**
 * Build the unified action feed from captures and backlog items.
 *
 * Sorting strategy:
 * 1. Captures always appear first (sorted among themselves by capture priority).
 * 2. Backlog items are sorted with dependency-aware ordering: items whose
 *    dependencies are incomplete sort below those dependencies.
 * 3. Within the same dependency depth, items sort by feed priority (attention
 *    items boosted) then by recency.
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

  // Sort captures by their own priority (classifying > failed > normal).
  const captureFeed: FeedItem[] = captures.map((c) => ({ type: "capture" as const, capture: c }));
  captureFeed.sort((a, b) => computeFeedPriority(a) - computeFeedPriority(b));

  // Classify backlog items as attention or normal.
  const includeFinished = options?.showFinished ?? false;
  const backlogFeed: FeedItem[] = [];
  for (const item of backlogItems) {
    if (!includeFinished && isFinished(item)) continue;
    const reasons = getAttentionReasons(item, feedbackMap, maturityMap);
    if (reasons.length > 0) {
      backlogFeed.push({ type: "attention", item, reasons });
    } else {
      backlogFeed.push({ type: "backlog", item });
    }
  }

  // Build a map from item key to FeedItem for reconstruction after sorting.
  const feedByKey = new Map<string, FeedItem>();
  const backlogSubset: BacklogItem[] = [];
  for (const entry of backlogFeed) {
    const item = entry.type === "capture" ? null : entry.item;
    if (!item) continue;
    const key = `${item.kind}/${item.name}`;
    feedByKey.set(key, entry);
    backlogSubset.push(item);
  }

  // Sort with dependency awareness, using feed priority + recency as tiebreaker.
  const sortedBacklog = dependencyAwareSort(
    backlogSubset,
    (a, b) => {
      const fa = feedByKey.get(`${a.kind}/${a.name}`);
      const fb = feedByKey.get(`${b.kind}/${b.name}`);
      if (!fa || !fb) {
        return 0;
      }
      const pd = computeFeedPriority(fa) - computeFeedPriority(fb);
      if (pd !== 0) return pd;
      return getSortTimestamp(fb) - getSortTimestamp(fa);
    },
    backlogItems, // full list for depth resolution (includes archived items)
  );

  return [
    ...captureFeed,
    ...sortedBacklog.flatMap((item) => {
      const feedItem = feedByKey.get(`${item.kind}/${item.name}`);
      return feedItem ? [feedItem] : [];
    }),
  ];
}

/**
 * Count items that need attention (captures + attention backlog items).
 */
export function countActionableItems(feed: FeedItem[]): number {
  return feed.filter((item) => item.type === "capture" || item.type === "attention").length;
}
