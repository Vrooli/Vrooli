import { describe, it, expect } from "vitest";
import { computeDepthMap, dependencyAwareSort, SORT_RESOLVED_STATUSES } from "./dependency-sort";
import type { BacklogItem, BacklogStatus } from "../types";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type DepthItem = Pick<BacklogItem, "kind" | "name" | "status" | "dependsOn">;

function makeItem(
  name: string,
  status: BacklogStatus = "backlog",
  dependsOn?: string[],
  overrides?: Partial<BacklogItem>,
): BacklogItem {
  return {
    name,
    title: name,
    description: "",
    kind: "execute" as const,
    status,
    priority: 5,
    tags: [],
    suggestedSkills: [],
    created: "2026-03-20T00:00:00Z",
    updated: "2026-03-20T00:00:00Z",
    dependsOn,
    ...overrides,
  } as BacklogItem;
}

function key(item: Pick<BacklogItem, "kind" | "name">): string {
  return `${item.kind}/${item.name}`;
}

function depths(items: ReadonlyArray<DepthItem>): Record<string, number> {
  const map = computeDepthMap(items);
  const result: Record<string, number> = {};
  for (const [k, v] of map) result[k] = v;
  return result;
}

// ---------------------------------------------------------------------------
// SORT_RESOLVED_STATUSES
// ---------------------------------------------------------------------------

describe("SORT_RESOLVED_STATUSES", () => {
  it("contains exactly completed", () => {
    expect(SORT_RESOLVED_STATUSES).toEqual(new Set(["completed"]));
  });
});

// ---------------------------------------------------------------------------
// computeDepthMap
// ---------------------------------------------------------------------------

describe("computeDepthMap", () => {
  it("returns empty map for empty array", () => {
    expect(computeDepthMap([])).toEqual(new Map());
  });

  it("assigns depth 0 to a single item with no deps", () => {
    const a = makeItem("a");
    expect(depths([a])).toEqual({ "execute/a": 0 });
  });

  it("assigns depth 0 to unrelated items", () => {
    const a = makeItem("a");
    const b = makeItem("b");
    expect(depths([a, b])).toEqual({ "execute/a": 0, "execute/b": 0 });
  });

  // Table-driven: dependency status determines whether it counts as incomplete
  const statusCases: Array<{ status: BacklogStatus; expectedDepth: number; label: string }> = [
    { status: "backlog", expectedDepth: 1, label: "backlog (incomplete)" },
    { status: "researching", expectedDepth: 1, label: "researching (incomplete)" },
    { status: "ready", expectedDepth: 1, label: "ready (incomplete)" },
    { status: "queued", expectedDepth: 1, label: "queued (incomplete)" },
    { status: "in_progress", expectedDepth: 1, label: "in_progress (incomplete)" },
    { status: "failed", expectedDepth: 1, label: "failed (incomplete)" },
    { status: "completed", expectedDepth: 0, label: "completed (resolved)" },
  ];

  describe.each(statusCases)("dep status: $label", ({ status, expectedDepth }) => {
    it(`A depending on B(${status}) has depth ${expectedDepth}`, () => {
      const b = makeItem("b", status);
      const a = makeItem("a", "backlog", ["execute/b"]);
      const result = computeDepthMap([a, b]);
      expect(result.get("execute/a")).toBe(expectedDepth);
      expect(result.get("execute/b")).toBe(0);
    });
  });

  it("handles a three-level chain: A→B→C", () => {
    const c = makeItem("c", "backlog");
    const b = makeItem("b", "backlog", ["execute/c"]);
    const a = makeItem("a", "backlog", ["execute/b"]);
    expect(depths([a, b, c])).toEqual({
      "execute/c": 0,
      "execute/b": 1,
      "execute/a": 2,
    });
  });

  it("handles a diamond: A→B,C; B,C→D", () => {
    const d = makeItem("d", "backlog");
    const b = makeItem("b", "backlog", ["execute/d"]);
    const c = makeItem("c", "backlog", ["execute/d"]);
    const a = makeItem("a", "backlog", ["execute/b", "execute/c"]);
    const result = depths([a, b, c, d]);
    expect(result).toEqual({
      "execute/d": 0,
      "execute/b": 1,
      "execute/c": 1,
      "execute/a": 2,
    });
  });

  it("ignores dangling refs (dep not in list)", () => {
    const a = makeItem("a", "backlog", ["execute/nonexistent"]);
    expect(depths([a])).toEqual({ "execute/a": 0 });
  });

  it("handles cycles gracefully without crashing", () => {
    const a = makeItem("a", "backlog", ["execute/b"]);
    const b = makeItem("b", "backlog", ["execute/a"]);
    // Should not throw; both get some depth (exact value is an implementation detail)
    expect(() => computeDepthMap([a, b])).not.toThrow();
    const result = computeDepthMap([a, b]);
    expect(result.has("execute/a")).toBe(true);
    expect(result.has("execute/b")).toBe(true);
  });

  it("mixed: A depends on B(completed) and C(ready) — depth from C only", () => {
    const b = makeItem("b", "completed");
    const c = makeItem("c", "ready");
    const a = makeItem("a", "backlog", ["execute/b", "execute/c"]);
    const result = computeDepthMap([a, b, c]);
    expect(result.get("execute/a")).toBe(1); // blocked by c (ready)
    expect(result.get("execute/b")).toBe(0);
    expect(result.get("execute/c")).toBe(0);
  });

  it("all deps resolved: A depends on B(completed) and C(completed+archived)", () => {
    const b = makeItem("b", "completed");
    const c = makeItem("c", "completed", undefined, { archivedAt: "2026-01-01T00:00:00Z" });
    const a = makeItem("a", "backlog", ["execute/b", "execute/c"]);
    const result = computeDepthMap([a, b, c]);
    expect(result.get("execute/a")).toBe(0);
  });

  it("handles items with empty dependsOn array", () => {
    const a = makeItem("a", "backlog", []);
    expect(depths([a])).toEqual({ "execute/a": 0 });
  });

  it("handles items with undefined dependsOn", () => {
    const a = makeItem("a", "backlog", undefined);
    expect(depths([a])).toEqual({ "execute/a": 0 });
  });

  it("handles cross-kind dependencies", () => {
    const dep = makeItem("dep", "ready", undefined, { kind: "idea" });
    const a = makeItem("a", "backlog", ["idea/dep"]);
    const result = computeDepthMap([a, dep]);
    expect(result.get("execute/a")).toBe(1);
    expect(result.get("idea/dep")).toBe(0);
  });
});

// ---------------------------------------------------------------------------
// dependencyAwareSort
// ---------------------------------------------------------------------------

describe("dependencyAwareSort", () => {
  const byPriority = (a: BacklogItem, b: BacklogItem) => a.priority - b.priority;

  it("returns empty array for empty input", () => {
    expect(dependencyAwareSort([], byPriority)).toEqual([]);
  });

  it("returns single item unchanged", () => {
    const a = makeItem("a");
    const result = dependencyAwareSort([a], byPriority);
    expect(result.map(key)).toEqual(["execute/a"]);
  });

  it("sorts by priority within the same depth", () => {
    const a = makeItem("a", "backlog", undefined, { priority: 1 });
    const b = makeItem("b", "backlog", undefined, { priority: 3 });
    const c = makeItem("c", "backlog", undefined, { priority: 2 });
    const result = dependencyAwareSort([b, c, a], byPriority);
    expect(result.map(key)).toEqual(["execute/a", "execute/c", "execute/b"]);
  });

  it("places dependency before dependent regardless of priority", () => {
    const dep = makeItem("dep", "backlog", undefined, { priority: 5 });
    const a = makeItem("a", "backlog", ["execute/dep"], { priority: 1 });
    const result = dependencyAwareSort([a, dep], byPriority);
    expect(result.map(key)).toEqual(["execute/dep", "execute/a"]);
  });

  it("high-priority item sorts below low-priority incomplete dep", () => {
    const dep = makeItem("dep", "ready", undefined, { priority: 10 });
    const a = makeItem("a", "backlog", ["execute/dep"], { priority: 1 });
    const result = dependencyAwareSort([a, dep], byPriority);
    expect(result.map(key)).toEqual(["execute/dep", "execute/a"]);
  });

  it("completed dep does not push dependent down", () => {
    const dep = makeItem("dep", "completed", undefined, { priority: 5 });
    const a = makeItem("a", "backlog", undefined, { priority: 1, dependsOn: ["execute/dep"] });
    const result = dependencyAwareSort([dep, a], byPriority);
    expect(result.map(key)).toEqual(["execute/a", "execute/dep"]);
  });

  it("archived (completed+archivedAt) dep does not push dependent down", () => {
    const dep = makeItem("dep", "completed", undefined, { priority: 5, archivedAt: "2026-01-01T00:00:00Z" });
    const a = makeItem("a", "backlog", undefined, { priority: 1, dependsOn: ["execute/dep"] });
    const result = dependencyAwareSort([dep, a], byPriority);
    expect(result.map(key)).toEqual(["execute/a", "execute/dep"]);
  });

  it("multi-level chain respects all levels", () => {
    const c = makeItem("c", "backlog", undefined, { priority: 3 });
    const b = makeItem("b", "backlog", ["execute/c"], { priority: 2 });
    const a = makeItem("a", "backlog", ["execute/b"], { priority: 1 });
    const result = dependencyAwareSort([a, b, c], byPriority);
    expect(result.map(key)).toEqual(["execute/c", "execute/b", "execute/a"]);
  });

  it("within same depth, tiebreaker compareFn applies", () => {
    const x = makeItem("x", "backlog", undefined, { priority: 5 });
    const a = makeItem("a", "backlog", ["execute/x"], { priority: 2 });
    const b = makeItem("b", "backlog", ["execute/x"], { priority: 1 });
    const result = dependencyAwareSort([a, x, b], byPriority);
    // x at depth 0, then b(p1) and a(p2) at depth 1
    expect(result.map(key)).toEqual(["execute/x", "execute/b", "execute/a"]);
  });

  it("allItems resolves deps from items not in the sorted set", () => {
    const dep = makeItem("dep", "ready", undefined, { priority: 5 });
    const a = makeItem("a", "backlog", ["execute/dep"], { priority: 1 });
    // Only sort [a], but allItems includes dep for depth resolution
    const result = dependencyAwareSort([a], byPriority, [a, dep]);
    // a should have depth 1 (dep is incomplete), but it's the only item in output
    expect(result.length).toBe(1);
    expect(result.map(key)).toEqual(["execute/a"]);
  });

  it("allItems depth affects relative ordering", () => {
    const dep = makeItem("dep", "ready", undefined, { priority: 5 });
    const a = makeItem("a", "backlog", ["execute/dep"], { priority: 1 });
    const b = makeItem("b", "backlog", undefined, { priority: 3 });
    // dep is filtered out but present in allItems — a gets depth 1, b gets depth 0
    const result = dependencyAwareSort([a, b], byPriority, [a, b, dep]);
    expect(result.map(key)).toEqual(["execute/b", "execute/a"]);
  });

  it("does not crash on cycles", () => {
    const a = makeItem("a", "backlog", ["execute/b"], { priority: 1 });
    const b = makeItem("b", "backlog", ["execute/a"], { priority: 2 });
    expect(() => dependencyAwareSort([a, b], byPriority)).not.toThrow();
    const result = dependencyAwareSort([a, b], byPriority);
    expect(result.length).toBe(2);
  });

  it("does not mutate the input array", () => {
    const a = makeItem("a", "backlog", undefined, { priority: 3 });
    const b = makeItem("b", "backlog", undefined, { priority: 1 });
    const input = [a, b];
    const result = dependencyAwareSort(input, byPriority);
    // Input order unchanged
    expect(input[0]?.name).toBe("a");
    expect(input[1]?.name).toBe("b");
    // Result is different order
    expect(result[0]?.name).toBe("b");
  });

  it("uses custom compareFn for non-priority tiebreaking", () => {
    const a = makeItem("a", "backlog", undefined, { priority: 1, updated: "2026-03-01T00:00:00Z" });
    const b = makeItem("b", "backlog", undefined, { priority: 1, updated: "2026-03-20T00:00:00Z" });
    const byRecency = (x: BacklogItem, y: BacklogItem) =>
      new Date(y.updated).getTime() - new Date(x.updated).getTime();
    const result = dependencyAwareSort([a, b], byRecency);
    // Same depth, same priority concept — recency tiebreaker puts b first
    expect(result.map(key)).toEqual(["execute/b", "execute/a"]);
  });
});
