import { describe, it, expect } from "vitest";
import { cn, randomCanvasPosition, deduplicateEdges } from "./utils";
import type { ThoughtEdge } from "./types";

// [REQ:P0-002] [REQ:P0-003] Utility functions

describe("cn", () => {
  it("merges simple class names", () => {
    expect(cn("px-2", "py-1")).toBe("px-2 py-1");
  });

  it("resolves conflicting Tailwind classes (last wins)", () => {
    expect(cn("px-2", "px-4")).toBe("px-4");
  });

  it("handles conditional classes via clsx", () => {
    const showHidden = false;
    expect(cn("base", showHidden && "hidden", "text-sm")).toBe("base text-sm");
  });

  it("handles undefined and null inputs", () => {
    expect(cn("a", undefined, null, "b")).toBe("a b");
  });

  it("returns empty string for no inputs", () => {
    expect(cn()).toBe("");
  });
});

describe("randomCanvasPosition", () => {
  it("returns x within [0, width)", () => {
    for (let i = 0; i < 50; i++) {
      const { x } = randomCanvasPosition(600, 400);
      expect(x).toBeGreaterThanOrEqual(0);
      expect(x).toBeLessThan(600);
    }
  });

  it("returns y within [0, height)", () => {
    for (let i = 0; i < 50; i++) {
      const { y } = randomCanvasPosition(600, 400);
      expect(y).toBeGreaterThanOrEqual(0);
      expect(y).toBeLessThan(400);
    }
  });

  it("respects custom dimensions", () => {
    for (let i = 0; i < 50; i++) {
      const { x, y } = randomCanvasPosition(100, 50);
      expect(x).toBeLessThan(100);
      expect(y).toBeLessThan(50);
    }
  });

  it("returns an object with x and y properties", () => {
    const pos = randomCanvasPosition(500, 300);
    expect(pos).toHaveProperty("x");
    expect(pos).toHaveProperty("y");
    expect(typeof pos.x).toBe("number");
    expect(typeof pos.y).toBe("number");
  });
});

// [REQ:P0-002] Edge deduplication for graph queries
describe("deduplicateEdges", () => {
  const makeEdge = (id: string, sourceId: string, targetId: string): ThoughtEdge => ({
    id,
    source_id: sourceId,
    target_id: targetId,
    label: "",
    created_at: "2026-01-01T00:00:00Z",
  });

  it("returns empty array for empty input", () => {
    expect(deduplicateEdges([])).toEqual([]);
  });

  it("returns edges from a single set unchanged", () => {
    const edges = [makeEdge("e1", "a", "b"), makeEdge("e2", "b", "c")];
    expect(deduplicateEdges([edges])).toEqual(edges);
  });

  it("removes duplicates across multiple sets", () => {
    const e1 = makeEdge("e1", "a", "b");
    const e2 = makeEdge("e2", "b", "c");
    const result = deduplicateEdges([[e1, e2], [e1], [e2]]);
    expect(result).toHaveLength(2);
    expect(result).toEqual([e1, e2]);
  });

  it("preserves order of first occurrence", () => {
    const e1 = makeEdge("e1", "a", "b");
    const e2 = makeEdge("e2", "b", "c");
    const e3 = makeEdge("e3", "c", "d");
    const result = deduplicateEdges([[e2], [e1, e3], [e2, e3]]);
    expect(result.map((e) => e.id)).toEqual(["e2", "e1", "e3"]);
  });
});
