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

/**
 * Minimal shape any entity must satisfy to participate in dependency-aware
 * sorting. Callers supply a `kind` namespace ("idea", "milestone", ...) plus
 * a stable `name` so that keys (`"kind/name"`) don't collide across domains.
 *
 * BacklogItem structurally satisfies this; milestones pass plain objects
 * with `kind: "milestone"`.
 */
export interface DepthItem {
  kind: string;
  name: string;
  status: string;
  dependsOn?: string[] | null;
  archivedAt?: string | null;
}

/**
 * Statuses where a dependency is considered resolved for sort-ordering purposes.
 * Only `completed` items are "done" — everything else means the dependent
 * should sort below its dependency. Archived items are also considered resolved
 * (checked via archivedAt at the callsite).
 */
export const SORT_RESOLVED_STATUSES: ReadonlySet<string> = new Set<string>([
  "completed",
]);

// ---------------------------------------------------------------------------
// Unblocking value scoring
// DOC: docs/concepts/ARCHITECTURE.md#priority-ranking
// ---------------------------------------------------------------------------

/**
 * Weight applied per transitive dependent when computing unblocking boost.
 * Each transitive dependent adds this much to the priority boost.
 */
export const UNBLOCK_WEIGHT = 0.5;

/**
 * Maximum priority boost from unblocking value. Prevents items with very
 * high fan-out from completely overriding manual priority.
 */
export const UNBLOCK_CAP = 3;

/**
 * Compute effective priority incorporating unblocking value.
 * Lower values = higher priority (sorted first).
 *
 * @param manualPriority - The item's manual priority (1-10).
 * @param transitiveDependentCount - Number of incomplete items transitively
 *   depending on this item.
 * @returns Effective priority (may be fractional).
 */
export function computeEffectivePriority(
  manualPriority: number,
  transitiveDependentCount: number,
): number {
  return manualPriority - Math.min(transitiveDependentCount * UNBLOCK_WEIGHT, UNBLOCK_CAP);
}

/**
 * Compute transitive dependent counts for all items.
 *
 * Builds a reverse dependency graph (item → its dependents) and counts how
 * many incomplete items are transitively reachable from each node. Items that
 * unblock more downstream work get higher counts.
 *
 * Only incomplete dependents (not `completed`, not archived) are counted —
 * a completed item that depends on X should not inflate X's unblocking value.
 *
 * Performance: O(V+E) for graph construction + O(V+E) total for memoized DFS.
 *
 * @param items - The full set of items to analyze.
 * @returns A map from item key (`"kind/name"`) to transitive dependent count.
 */
export function computeUnblockingMap(
  items: ReadonlyArray<DepthItem>,
): Map<string, number> {
  const itemsByKey = new Map<string, DepthItem>();
  for (const item of items) {
    itemsByKey.set(itemKey(item), item);
  }

  // Build reverse adjacency list: dependency → incomplete dependents.
  const reverseDeps = new Map<string, string[]>();
  for (const item of items) {
    // Skip completed/archived items — they don't count as dependents.
    if (SORT_RESOLVED_STATUSES.has(item.status) || item.archivedAt) continue;

    const key = itemKey(item);
    for (const dep of item.dependsOn ?? []) {
      // Only record if the dependency exists in our item set.
      if (!itemsByKey.has(dep)) continue;
      let list = reverseDeps.get(dep);
      if (!list) {
        list = [];
        reverseDeps.set(dep, list);
      }
      list.push(key);
    }
  }

  // Memoized DFS to count all transitively reachable dependents.
  // Uses visited-set approach to handle diamonds and cycles correctly.
  const cache = new Map<string, Set<string>>();

  function getTransitiveDependents(key: string): Set<string> {
    const cached = cache.get(key);
    if (cached) return cached;

    // Place an empty set in the cache before recursing to handle cycles.
    const result = new Set<string>();
    cache.set(key, result);

    const directDeps = reverseDeps.get(key);
    if (directDeps) {
      for (const dep of directDeps) {
        result.add(dep);
        for (const transitive of getTransitiveDependents(dep)) {
          result.add(transitive);
        }
      }
    }

    return result;
  }

  const unblockingMap = new Map<string, number>();
  for (const item of items) {
    const key = itemKey(item);
    unblockingMap.set(key, getTransitiveDependents(key).size);
  }

  return unblockingMap;
}

/**
 * Build a canonical key: `"kind/name"`. Pure helper — any object with the two
 * fields works.
 */
function itemKey(item: { kind: string; name: string }): string {
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
export function dependencyAwareSort<T extends { kind: string; name: string; dependsOn?: string[] | null }>(
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
