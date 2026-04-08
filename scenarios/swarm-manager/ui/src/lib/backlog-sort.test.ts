import { describe, it, expect } from "vitest";
import { buildBacklogCompareFn, sortBacklogItems, COMMAND_POST_COMPARE } from "./backlog-sort";
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
// COMMAND_POST_COMPARE
// ---------------------------------------------------------------------------

describe("COMMAND_POST_COMPARE", () => {
  it("sorts by priority ascending", () => {
    const low = makeItem("low", { priority: 1 });
    const high = makeItem("high", { priority: 5 });
    expect(COMMAND_POST_COMPARE(low, high)).toBeLessThan(0);
    expect(COMMAND_POST_COMPARE(high, low)).toBeGreaterThan(0);
  });

  it("breaks ties by recency (newest first)", () => {
    const older = makeItem("older", { priority: 3, updated: "2026-03-01T00:00:00Z" });
    const newer = makeItem("newer", { priority: 3, updated: "2026-03-20T00:00:00Z" });
    expect(COMMAND_POST_COMPARE(older, newer)).toBeGreaterThan(0);
    expect(COMMAND_POST_COMPARE(newer, older)).toBeLessThan(0);
  });

  it("returns 0 for identical priority and timestamp", () => {
    const a = makeItem("a", { priority: 3, updated: "2026-03-20T00:00:00Z" });
    const b = makeItem("b", { priority: 3, updated: "2026-03-20T00:00:00Z" });
    expect(COMMAND_POST_COMPARE(a, b)).toBe(0);
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
    const result = sortBacklogItems([child, parent], COMMAND_POST_COMPARE, [child, parent]);
    // Parent has depth 0, child has depth 1 — parent sorts first despite lower priority number on child
    expect(names(result)).toEqual(["parent", "child"]);
  });

  it("uses compareFn as tiebreaker within same depth", () => {
    const a = makeItem("a-item", { priority: 1 });
    const b = makeItem("b-item", { priority: 3 });
    const c = makeItem("c-item", { priority: 2 });
    const result = sortBacklogItems([b, c, a], COMMAND_POST_COMPARE, [a, b, c]);
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
    const result = sortBacklogItems([parent, child], COMMAND_POST_COMPARE, [parent, child]);
    // Child has lower priority number, so it sorts first within same depth
    expect(names(result)).toEqual(["child", "parent"]);
  });

  it("handles multi-level chains", () => {
    const root = makeItem("root", { priority: 5 });
    const mid = makeItem("mid", { priority: 1, dependsOn: ["execute/root"] });
    const leaf = makeItem("leaf", { priority: 1, dependsOn: ["execute/mid"] });
    const result = sortBacklogItems([leaf, mid, root], COMMAND_POST_COMPARE, [root, mid, leaf]);
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
      COMMAND_POST_COMPARE,
      [dep, child, unrelated],
    );
    // child has depth 1 (dep is incomplete), unrelated has depth 0
    expect(names(result)).toEqual(["unrelated", "child"]);
  });
});
