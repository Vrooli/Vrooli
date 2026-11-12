import type { Overlap } from "../api/activity";

/** Two commissions that held the same topics at the same time. */
export interface OverlapPair {
  key: string;
  ids: string[];
  topics: string[];
}

/**
 * The API reports one overlap per pair per topic; the operator reads one row
 * per pair with the topics it shared. Only pairs whose commissions are all in
 * `among` are kept, so Now shows what overlaps today and history stays on the
 * commission page (rulings V5).
 */
export function overlapPairs(overlaps: Overlap[], among?: ReadonlySet<string>): OverlapPair[] {
  const pairs = new Map<string, OverlapPair>();
  for (const overlap of overlaps) {
    const ids = [...new Set(overlap.commissionIds)].sort();
    if (among && !ids.every((id) => among.has(id))) continue;
    const key = ids.join(" ");
    const pair = pairs.get(key) ?? { key, ids, topics: [] };
    if (!pair.topics.includes(overlap.topic)) pair.topics.push(overlap.topic);
    pairs.set(key, pair);
  }
  return [...pairs.values()];
}
