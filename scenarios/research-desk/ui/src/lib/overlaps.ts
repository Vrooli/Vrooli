import type { Overlap } from "../api/activity";

/** Two commissions that held the same topics at the same time. */
export interface OverlapPair {
  key: string;
  ids: string[];
  topics: string[];
}

/**
 * The API reports one overlap per pair per topic; the operator reads one row
 * per pair with the topics it shared.
 */
export function overlapPairs(overlaps: Overlap[]): OverlapPair[] {
  const pairs = new Map<string, OverlapPair>();
  for (const overlap of overlaps) {
    const ids = [...new Set(overlap.commissionIds)].sort();
    const key = ids.join(" ");
    const pair = pairs.get(key) ?? { key, ids, topics: [] };
    if (!pair.topics.includes(overlap.topic)) pair.topics.push(overlap.topic);
    pairs.set(key, pair);
  }
  return [...pairs.values()];
}
