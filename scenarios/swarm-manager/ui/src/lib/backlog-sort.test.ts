import { describe, it, expect } from "vitest";
import { buildBacklogCompareFn, buildCommandPostCompare, sortBacklogItems } from "./backlog-sort";
import { computeUnblockingMap } from "./dependency-sort";
import type { BacklogItem } from "../types";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeItem(
  name: string,
  overrides?: Partial<BacklogItem>,
): BacklogItem {
  return {
    name,
    title: name,
    description: "",
    kind: "execute" as const,
    status: "backlog",
    priority: 5,
    tags: [],
    suggestedSkills: [],
    created: "2026-03-20T00:00:00Z",
    updated: "2026-03-20T00:00:00Z",
    ...overrides,
  } as BacklogItem;
}

function names(items: ReadonlyArray<Pick<BacklogItem, "name">>): string[] {
  return items.map((i) => i.name);
}

// ---------------------------------------------------------------------------
// buildBacklogCompareFn
// ---------------------------------------------------------------------------

describe("buildBacklogCompareFn", () => {
  const a = makeItem("alpha", { priority: 1, updated: "2026-03-20T00:00:00Z", status: "backlog" });
  const b = makeItem("beta", { priority: 3, updated: "2026-03-21T00:00:00Z", status: "ready" });
  const c = makeItem("gamma", { priority: 2, updated: "2026-03-19T00:00:00Z", status: "completed" });

  it("sorts by priority ascending", () => {
    const cmp = buildBacklogCompareFn({ field: "priority", direction: "asc" });
    const sorted = [b, c, a].sort(cmp);
    expect(names(sorted)).toEqual(["alpha", "gamma", "beta"]);
  });

  it("sorts by priority descending", () => {
    const cmp = buildBacklogCompareFn({ field: "priority", direction: "desc" });
    const sorted = [a, c, b].sort(cmp);
    expect(names(sorted)).toEqual(["beta", "gamma", "alpha"]);
  });

  it("sorts by recency ascending (newest first — default recency direction)", () => {
    const cmp = buildBacklogCompareFn({ field: "recency", direction: "asc" });
    const sorted = [a, b, c].sort(cmp);
    // The base compareFn is (b.updated - a.updated), so asc preserves newest-first
    expect(names(sorted)).toEqual(["beta", "alpha", "gamma"]);
  });

  it("sorts by recency descending (oldest first — reversed)", () => {
    const cmp = buildBacklogCompareFn({ field: "recency", direction: "desc" });
    const sorted = [c, a, b].sort(cmp);
    expect(names(sorted)).toEqual(["gamma", "alpha", "beta"]);
  });

  it("sorts by status alphabetically ascending", () => {
    const cmp = buildBacklogCompareFn({ field: "status", direction: "asc" });
    const sorted = [b, c, a].sort(cmp);
    // "backlog" < "completed" < "ready"
    expect(names(sorted)).toEqual(["alpha", "gamma", "beta"]);
  });

  it("sorts by status alphabetically descending", () => {
    const cmp = buildBacklogCompareFn({ field: "status", direction: "desc" });
    const sorted = [a, c, b].sort(cmp);
    expect(names(sorted)).toEqual(["beta", "gamma", "alpha"]);
  });

  it("sorts alphabetically ascending by title", () => {
    const cmp = buildBacklogCompareFn({ field: "alphabetical", direction: "asc" });
    const sorted = [c, a, b].sort(cmp);
    expect(names(sorted)).toEqual(["alpha", "beta", "gamma"]);
  });

  it("sorts alphabetically descending by title", () => {
    const cmp = buildBacklogCompareFn({ field: "alphabetical", direction: "desc" });
    const sorted = [a, b, c].sort(cmp);
    expect(names(sorted)).toEqual(["gamma", "beta", "alpha"]);
  });

  it("falls back to name when title is empty", () => {
    const x = makeItem("x-name", { title: "" });
    const y = makeItem("y-name", { title: "" });
    const cmp = buildBacklogCompareFn({ field: "alphabetical", direction: "asc" });
    expect(cmp(x, y)).toBeLessThan(0);
  });
});

// ---------------------------------------------------------------------------
// buildCommandPostCompare
// ---------------------------------------------------------------------------

/** No-unblocking comparator for tests that don't need unblocking data. */
const COMPARE = buildCommandPostCompare(new Map());

describe("buildCommandPostCompare", () => {
  it("sorts by priority ascending", () => {
    const low = makeItem("low", { priority: 1 });
    const high = makeItem("high", { priority: 5 });
    expect(COMPARE(low, high)).toBeLessThan(0);
    expect(COMPARE(high, low)).toBeGreaterThan(0);
  });

  it("breaks ties by recency (newest first)", () => {
    const older = makeItem("older", { priority: 3, updated: "2026-03-01T00:00:00Z" });
    const newer = makeItem("newer", { priority: 3, updated: "2026-03-20T00:00:00Z" });
    expect(COMPARE(older, newer)).toBeGreaterThan(0);
    expect(COMPARE(newer, older)).toBeLessThan(0);
  });

  it("returns 0 for identical priority and timestamp", () => {
    const a = makeItem("a", { priority: 3, updated: "2026-03-20T00:00:00Z" });
    const b = makeItem("b", { priority: 3, updated: "2026-03-20T00:00:00Z" });
    expect(COMPARE(a, b)).toBe(0);
  });
});

// ---------------------------------------------------------------------------
// sortBacklogItems
// ---------------------------------------------------------------------------

describe("sortBacklogItems", () => {
  it("respects dependency ordering over priority", () => {
    const parent = makeItem("parent", { priority: 5, status: "backlog" });
    const child = makeItem("child", {
      priority: 1,
      status: "backlog",
      dependsOn: ["execute/parent"],
    });
    const result = sortBacklogItems([child, parent], COMPARE, [child, parent]);
    // Parent has depth 0, child has depth 1 — parent sorts first despite lower priority number on child
    expect(names(result)).toEqual(["parent", "child"]);
  });

  it("uses compareFn as tiebreaker within same depth", () => {
    const a = makeItem("a-item", { priority: 1 });
    const b = makeItem("b-item", { priority: 3 });
    const c = makeItem("c-item", { priority: 2 });
    const result = sortBacklogItems([b, c, a], COMPARE, [a, b, c]);
    expect(names(result)).toEqual(["a-item", "c-item", "b-item"]);
  });

  it("handles completed deps correctly (child sorts alongside parent)", () => {
    const parent = makeItem("parent", { priority: 5, status: "completed" });
    const child = makeItem("child", {
      priority: 1,
      status: "backlog",
      dependsOn: ["execute/parent"],
    });
    // Parent is completed, so child's dep is resolved — both at depth 0
    const result = sortBacklogItems([parent, child], COMPARE, [parent, child]);
    // Child has lower priority number, so it sorts first within same depth
    expect(names(result)).toEqual(["child", "parent"]);
  });

  it("handles multi-level chains", () => {
    const root = makeItem("root", { priority: 5 });
    const mid = makeItem("mid", { priority: 1, dependsOn: ["execute/root"] });
    const leaf = makeItem("leaf", { priority: 1, dependsOn: ["execute/mid"] });
    const result = sortBacklogItems([leaf, mid, root], COMPARE, [root, mid, leaf]);
    expect(names(result)).toEqual(["root", "mid", "leaf"]);
  });

  it("uses allItems for depth resolution when items is a filtered subset", () => {
    const dep = makeItem("dep", { priority: 5, status: "backlog" });
    const child = makeItem("child", {
      priority: 1,
      status: "backlog",
      dependsOn: ["execute/dep"],
    });
    const unrelated = makeItem("unrelated", { priority: 2 });
    // Only sorting [child, unrelated] but dep exists in allItems
    const result = sortBacklogItems(
      [child, unrelated],
      COMPARE,
      [dep, child, unrelated],
    );
    // child has depth 1 (dep is incomplete), unrelated has depth 0
    expect(names(result)).toEqual(["unrelated", "child"]);
  });

  it("items with higher fan-out sort before same-priority peers within same depth", () => {
    // blocker has 2 dependents, standalone has 0
    const blocker = makeItem("blocker", { priority: 5 });
    const standalone = makeItem("standalone", { priority: 5 });
    const depA = makeItem("dep-a", { priority: 5, dependsOn: ["execute/blocker"] });
    const depB = makeItem("dep-b", { priority: 5, dependsOn: ["execute/blocker"] });
    const allItems = [blocker, standalone, depA, depB];

    const unblockingMap = computeUnblockingMap(allItems);
    const compare = buildCommandPostCompare(unblockingMap);
    const result = sortBacklogItems([standalone, blocker], compare, allItems);
    // blocker has 2 dependents → effective priority 5 - min(2*0.5, 3) = 4
    // standalone has 0 dependents → effective priority 5
    expect(names(result)).toEqual(["blocker", "standalone"]);
  });

  it("unblocking boost does NOT override depth ordering", () => {
    // high-fan-out item at depth 1 should still sort below depth-0 items
    const root = makeItem("root", { priority: 5, status: "backlog" });
    const mid = makeItem("mid", { priority: 5, status: "backlog", dependsOn: ["execute/root"] });
    const leaf1 = makeItem("leaf1", { priority: 5, status: "backlog", dependsOn: ["execute/mid"] });
    const leaf2 = makeItem("leaf2", { priority: 5, status: "backlog", dependsOn: ["execute/mid"] });
    const leaf3 = makeItem("leaf3", { priority: 5, status: "backlog", dependsOn: ["execute/mid"] });
    const allItems = [root, mid, leaf1, leaf2, leaf3];

    const unblockingMap = computeUnblockingMap(allItems);
    const compare = buildCommandPostCompare(unblockingMap);
    const result = sortBacklogItems([mid, root], compare, allItems);
    // mid has 3 dependents but is at depth 1; root is at depth 0 — root sorts first
    expect(names(result)).toEqual(["root", "mid"]);
  });

  it("buildBacklogCompareFn uses effective priority for 'priority' sort field", () => {
    const blocker = makeItem("blocker", { priority: 5 });
    const normal = makeItem("normal", { priority: 5 });
    const dep = makeItem("dep", { priority: 5, dependsOn: ["execute/blocker"] });
    const allItems = [blocker, normal, dep];

    const unblockingMap = computeUnblockingMap(allItems);
    const cmp = buildBacklogCompareFn({ field: "priority", direction: "asc" }, unblockingMap);
    // blocker has effective priority 4.5, normal has 5
    expect(cmp(blocker, normal)).toBeLessThan(0);
  });
});
