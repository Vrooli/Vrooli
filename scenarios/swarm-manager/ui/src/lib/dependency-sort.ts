/**
 * Dependency-Aware Sort Ordering
 *
 * Provides topological-depth-based sorting so that dependencies always appear
 * before their dependents in every sorted view (feed, sidebar tabs).
 *
 * **Important distinction — two different "blocking" concepts exist:**
 *
 * 1. **Queue-blocking** (`backlog-queue-utils.ts`): Controls whether an action
 *    (queue, workshop) is *available*. Uses a narrow status set (`backlog`,
 *    `researching`) because you CAN workshop an item whose dependency is `ready`.
 *
 * 2. **Sort-blocking** (this module): Controls *display ordering*. Any dependency
 *    that isn't `completed` or `archived` is considered incomplete — the dependent
 *    sorts below it. This is intentionally broader so users always see "do this
 *    first" items above the things that depend on them.
 *
 * DOC: docs/concepts/ARCHITECTURE.md#dependency-sort
 */

import type { BacklogItem, BacklogStatus } from "../types";

/**
 * Statuses where a dependency is considered resolved for sort-ordering purposes.
 * Only `completed` items are "done" — everything else means the dependent
 * should sort below its dependency. Archived items are also considered resolved
 * (checked via archivedAt at the callsite).
 */
export const SORT_RESOLVED_STATUSES: ReadonlySet<BacklogStatus> = new Set<BacklogStatus>([
  "completed",
]);

/** Minimal shape needed from a backlog item to compute dependency depths. */
type DepthItem = Pick<BacklogItem, "kind" | "name" | "status" | "dependsOn" | "archivedAt">;

/**
 * Build a canonical key for a backlog item: `"kind/name"`.
 */
function itemKey(item: Pick<BacklogItem, "kind" | "name">): string {
  return `${item.kind}/${item.name}`;
}

/**
 * Compute the topological depth of each item based on its incomplete dependencies.
 *
 * - Depth 0: no incomplete dependencies (or all deps are completed/archived).
 * - Depth N: depends on something at depth N−1.
 * - Dangling refs (deps referencing items not in the list) are ignored.
 * - Cycles are handled gracefully: iterations are capped at `items.length`,
 *   so cycle members stabilize at a shared depth rather than looping forever.
 *
 * @param items - The full set of items to compute depths for.
 * @returns A map from item key (`"kind/name"`) to its computed depth.
 */
export function computeDepthMap(
  items: ReadonlyArray<DepthItem>,
): Map<string, number> {
  const itemsByKey = new Map<string, DepthItem>();
  const depths = new Map<string, number>();

  for (const item of items) {
    const key = itemKey(item);
    itemsByKey.set(key, item);
    depths.set(key, 0);
  }

  // Pre-compute incomplete (unresolved) dependencies for each item.
  // Only deps that exist in our item set AND have a non-resolved status count.
  const incompleteDeps = new Map<string, string[]>();
  for (const item of items) {
    const deps: string[] = [];
    for (const dep of item.dependsOn ?? []) {
      const depItem = itemsByKey.get(dep);
      if (depItem && !SORT_RESOLVED_STATUSES.has(depItem.status) && !depItem.archivedAt) {
        deps.push(dep);
      }
    }
    if (deps.length > 0) {
      incompleteDeps.set(itemKey(item), deps);
    }
  }

  // Iterative relaxation: propagate depths until stable or capped.
  // At most items.length iterations handles the longest possible chain;
  // cycles will stabilize because each iteration can only increment by 1.
  const maxIterations = items.length;
  for (let i = 0; i < maxIterations; i++) {
    let changed = false;
    for (const [key, deps] of incompleteDeps) {
      let maxDepDepth = 0;
      for (const dep of deps) {
        maxDepDepth = Math.max(maxDepDepth, depths.get(dep) ?? 0);
      }
      const newDepth = maxDepDepth + 1;
      if (newDepth !== depths.get(key)) {
        depths.set(key, newDepth);
        changed = true;
      }
    }
    if (!changed) break;
  }

  return depths;
}

/**
 * Sort items so that dependencies always appear before their dependents.
 * Within the same dependency depth, items are ordered by the provided `compareFn`.
 *
 * @param items - The items to sort (not mutated; returns a new array).
 * @param compareFn - Tiebreaker sort function applied within the same depth layer.
 * @param allItems - Optional full (unfiltered) item list for resolving dependency
 *                   statuses when `items` is a filtered subset. If omitted, depth
 *                   is computed from `items` alone.
 * @returns A new sorted array.
 */
export function dependencyAwareSort<T extends Pick<BacklogItem, "kind" | "name" | "dependsOn">>(
  items: ReadonlyArray<T>,
  compareFn: (a: T, b: T) => number,
  allItems?: ReadonlyArray<DepthItem>,
): T[] {
  if (items.length <= 1) return [...items];

  // Compute depths from the full item set (which includes status info).
  // When items is a filtered subset, allItems provides the complete picture.
  const depthSource: ReadonlyArray<DepthItem> = allItems ?? (items as unknown as ReadonlyArray<DepthItem>);
  const depths = computeDepthMap(depthSource);

  return [...items].sort((a, b) => {
    const depthA = depths.get(itemKey(a)) ?? 0;
    const depthB = depths.get(itemKey(b)) ?? 0;
    if (depthA !== depthB) return depthA - depthB;
    return compareFn(a, b);
  });
}
