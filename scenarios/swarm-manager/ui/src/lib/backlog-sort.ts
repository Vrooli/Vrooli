/**
 * Backlog Sort Utilities
 *
 * Composes dependency-aware sorting from dependency-sort.ts with domain-specific
 * compareFn builders. Shared by BacklogTab (sidebar) and the Command Post feed.
 *
 * DOC: docs/concepts/ARCHITECTURE.md#command-post
 * DOC: docs/concepts/ARCHITECTURE.md#priority-ranking
 */

import type { BacklogItem } from "../types";
import { computeEffectivePriority, dependencyAwareSort } from "./dependency-sort";

// Re-export sort types so consumers don't need the deep sidebar path.
export type { SortConfig, SortField, SortDirection } from "../surfaces/graph/components/sidebar/types";
import type { SortConfig } from "../surfaces/graph/components/sidebar/types";

// ---------------------------------------------------------------------------
// Compare function builders
// ---------------------------------------------------------------------------

/**
 * Build a compareFn from a SortConfig.
 *
 * When sorting by "priority" and an `unblockingMap` is provided, effective
 * priority (incorporating unblocking value) is used instead of raw priority.
 *
 * @param sort - Sort configuration (field + direction).
 * @param unblockingMap - Optional map of "kind/name" → transitive dependent count.
 */
export function buildBacklogCompareFn(
  sort: SortConfig,
  unblockingMap?: Map<string, number>,
): (a: BacklogItem, b: BacklogItem) => number {
  const dir = sort.direction === "asc" ? 1 : -1;
  return (a: BacklogItem, b: BacklogItem): number => {
    switch (sort.field) {
      case "priority": {
        const effA = computeEffectivePriority(a.priority, unblockingMap?.get(`${a.kind}/${a.name}`) ?? 0);
        const effB = computeEffectivePriority(b.priority, unblockingMap?.get(`${b.kind}/${b.name}`) ?? 0);
        return (effA - effB) * dir;
      }
      case "recency":
        return (new Date(b.updated).getTime() - new Date(a.updated).getTime()) * dir;
      case "status":
        return a.status.localeCompare(b.status) * dir;
      case "alphabetical":
        return (a.title || a.name).localeCompare(b.title || b.name) * dir;
    }
  };
}

/**
 * Build the default command-post sort compareFn: effective priority ascending,
 * recency descending tiebreaker.
 *
 * Matches the natural expectation: highest-priority (lowest effective number)
 * first, most recently updated first within the same priority.
 *
 * @param unblockingMap - Map of "kind/name" → transitive dependent count.
 */
export function buildCommandPostCompare(
  unblockingMap: Map<string, number>,
): (a: BacklogItem, b: BacklogItem) => number {
  return (a: BacklogItem, b: BacklogItem): number => {
    const effA = computeEffectivePriority(a.priority, unblockingMap.get(`${a.kind}/${a.name}`) ?? 0);
    const effB = computeEffectivePriority(b.priority, unblockingMap.get(`${b.kind}/${b.name}`) ?? 0);
    const pDiff = effA - effB;
    if (pDiff !== 0) return pDiff;
    return new Date(b.updated).getTime() - new Date(a.updated).getTime();
  };
}

// ---------------------------------------------------------------------------
// Sort entry point
// ---------------------------------------------------------------------------

/**
 * Sort backlog items with dependency-aware ordering.
 *
 * Dependencies always appear before their dependents. Within the same
 * dependency depth, items are ordered by `compareFn`.
 *
 * @param items    - Items to sort (not mutated).
 * @param compareFn - Tiebreaker within the same depth layer.
 * @param allItems  - Full unfiltered item list for depth resolution when
 *                    `items` is a filtered subset.
 */
export function sortBacklogItems(
  items: ReadonlyArray<BacklogItem>,
  compareFn: (a: BacklogItem, b: BacklogItem) => number,
  allItems: ReadonlyArray<BacklogItem>,
): BacklogItem[] {
  return dependencyAwareSort(items, compareFn, allItems);
}
