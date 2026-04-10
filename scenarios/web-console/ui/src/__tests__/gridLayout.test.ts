import { describe, it, expect } from "vitest";
import {
  resolveWorkspaceLayout,
  reconcileTrackFractions,
  buildGridTrackTemplate,
  updateAdjacentFractions,
  fractionsMatch,
} from "../lib/gridLayout";

describe("resolveWorkspaceLayout", () => {
  it("returns 1×1 for a single pane", () => {
    expect(resolveWorkspaceLayout(1)).toEqual({ columns: 1, rows: 1 });
  });

  it("returns 2×1 for two panes", () => {
    expect(resolveWorkspaceLayout(2)).toEqual({ columns: 2, rows: 1 });
  });

  it("returns 2×2 for three panes", () => {
    expect(resolveWorkspaceLayout(3)).toEqual({ columns: 2, rows: 2 });
  });

  it("returns 2×2 for four panes", () => {
    expect(resolveWorkspaceLayout(4)).toEqual({ columns: 2, rows: 2 });
  });

  it("clamps zero to 1×1", () => {
    expect(resolveWorkspaceLayout(0)).toEqual({ columns: 1, rows: 1 });
  });

  it("clamps negative to 1×1", () => {
    expect(resolveWorkspaceLayout(-5)).toEqual({ columns: 1, rows: 1 });
  });

  it("respects maxColumns=1", () => {
    expect(resolveWorkspaceLayout(4, 1)).toEqual({ columns: 1, rows: 4 });
  });

  it("handles NaN", () => {
    expect(resolveWorkspaceLayout(NaN)).toEqual({ columns: 1, rows: 1 });
  });
});

describe("reconcileTrackFractions", () => {
  it("returns [1] for count 1", () => {
    expect(reconcileTrackFractions([0.5, 0.5], 1)).toEqual([1]);
  });

  it("returns equal fractions for empty input", () => {
    const result = reconcileTrackFractions([], 3);
    expect(result).toHaveLength(3);
    expect(result.every((f) => Math.abs(f - 1 / 3) < 0.001)).toBe(true);
  });

  it("preserves existing fractions when count matches", () => {
    const result = reconcileTrackFractions([0.6, 0.4], 2);
    expect(result[0]).toBeCloseTo(0.6, 5);
    expect(result[1]).toBeCloseTo(0.4, 5);
  });

  it("adds new tracks when count increases", () => {
    const result = reconcileTrackFractions([0.5, 0.5], 3);
    expect(result).toHaveLength(3);
  });

  it("trims tracks when count decreases", () => {
    const result = reconcileTrackFractions([0.3, 0.3, 0.4], 2);
    expect(result).toHaveLength(2);
  });

  it("returns [1] for zero count", () => {
    expect(reconcileTrackFractions([0.5], 0)).toEqual([1]);
  });

  it("returns [1] for negative count", () => {
    expect(reconcileTrackFractions([], -1)).toEqual([1]);
  });
});

describe("buildGridTrackTemplate", () => {
  it("returns single fraction for one track", () => {
    expect(buildGridTrackTemplate([1], 8)).toBe("minmax(0, 1fr)");
  });

  it("returns tracks with splitters for two tracks", () => {
    const result = buildGridTrackTemplate([0.5, 0.5], 8);
    expect(result).toBe("minmax(0, 0.5fr) 8px minmax(0, 0.5fr)");
  });

  it("returns tracks with splitters for three tracks", () => {
    const result = buildGridTrackTemplate([0.3, 0.3, 0.4], 6);
    expect(result).toContain("6px");
    // Should have 3 minmax() segments and 2 splitter segments
    const minmaxCount = (result.match(/minmax/g) ?? []).length;
    const splitterCount = (result.match(/6px/g) ?? []).length;
    expect(minmaxCount).toBe(3);
    expect(splitterCount).toBe(2);
  });

  it("handles empty fractions", () => {
    expect(buildGridTrackTemplate([], 8)).toBe("minmax(0, 1fr)");
  });
});

describe("updateAdjacentFractions", () => {
  it("returns original when index is out of bounds", () => {
    const fractions = [0.5, 0.5];
    expect(
      updateAdjacentFractions({
        startValues: fractions,
        index: 5,
        delta: 100,
        containerSize: 1000,
        splitterCount: 1,
        minTrackPx: 240,
        splitterSize: 8,
      }),
    ).toBe(fractions);
  });

  it("returns original for negative index", () => {
    const fractions = [0.5, 0.5];
    expect(
      updateAdjacentFractions({
        startValues: fractions,
        index: -1,
        delta: 100,
        containerSize: 1000,
        splitterCount: 1,
        minTrackPx: 240,
        splitterSize: 8,
      }),
    ).toBe(fractions);
  });

  it("adjusts adjacent fractions for positive delta", () => {
    const result = updateAdjacentFractions({
      startValues: [0.5, 0.5],
      index: 0,
      delta: 100,
      containerSize: 1000,
      splitterCount: 1,
      minTrackPx: 240,
      splitterSize: 8,
    });
    expect(result[0]).toBeGreaterThan(0.5);
    expect(result[1]).toBeLessThan(0.5);
    expect((result[0] ?? 0) + (result[1] ?? 0)).toBeCloseTo(1, 5);
  });

  it("clamps to minimum track size", () => {
    const result = updateAdjacentFractions({
      startValues: [0.5, 0.5],
      index: 0,
      delta: 5000, // very large delta
      containerSize: 1000,
      splitterCount: 1,
      minTrackPx: 240,
      splitterSize: 8,
    });
    // The second track should not go below minTrackPx
    expect(result[1]).toBeGreaterThan(0);
    expect((result[0] ?? 0) + (result[1] ?? 0)).toBeCloseTo(1, 5);
  });

  it("returns original for zero delta", () => {
    const fractions = [0.5, 0.5];
    const result = updateAdjacentFractions({
      startValues: fractions,
      index: 0,
      delta: 0,
      containerSize: 1000,
      splitterCount: 1,
      minTrackPx: 240,
      splitterSize: 8,
    });
    expect(result[0]).toBeCloseTo(0.5, 5);
    expect(result[1]).toBeCloseTo(0.5, 5);
  });
});

describe("fractionsMatch", () => {
  it("returns true for identical arrays", () => {
    expect(fractionsMatch([0.5, 0.5], [0.5, 0.5])).toBe(true);
  });

  it("returns false for different lengths", () => {
    expect(fractionsMatch([0.5, 0.5], [1])).toBe(false);
  });

  it("returns false for clearly different values", () => {
    expect(fractionsMatch([0.6, 0.4], [0.5, 0.5])).toBe(false);
  });

  it("returns true for empty arrays", () => {
    expect(fractionsMatch([], [])).toBe(true);
  });

  it("absorbs IEEE 754 normalization drift", () => {
    // Simulates the exact drift that causes React error #185:
    // 1/6 × 6 !== 1.0 in IEEE 754, so re-normalizing [1/6, …] produces
    // slightly different values on each pass, oscillating between two states.
    const stateA = [
      0.16666666666666669, 0.16666666666666669, 0.16666666666666669,
      0.16666666666666669, 0.16666666666666669, 0.16666666666666669,
    ];
    const stateB = [
      0.16666666666666666, 0.16666666666666666, 0.16666666666666666,
      0.16666666666666666, 0.16666666666666666, 0.16666666666666666,
    ];
    // Strict equality sees these as different (the bug)
    expect(stateA.some((f, i) => f !== stateB[i])).toBe(true);
    // fractionsMatch correctly treats them as equal (the fix)
    expect(fractionsMatch(stateA, stateB)).toBe(true);
  });
});

describe("reconcileTrackFractions + fractionsMatch stability", () => {
  // Regression tests: reconciling fractions for any pane count must stabilize
  // within one pass when checked via fractionsMatch.  Before the epsilon fix,
  // counts 6, 7, and 9 caused an infinite oscillation loop.
  const PANE_COUNTS_TO_TEST = [2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 15, 20];

  for (const n of PANE_COUNTS_TO_TEST) {
    it(`stabilizes after one reconciliation for ${n} panes (tabs mode)`, () => {
      // Simulate going from (n-1) to n panes in tabs mode (maxColumns=1)
      const prevFractions = reconcileTrackFractions(
        Array.from({ length: n - 1 }, () => 1 / (n - 1)),
        n - 1,
      );
      // Reconcile to the new count
      const first = reconcileTrackFractions(prevFractions, n);
      // Second pass should match via epsilon comparison
      const second = reconcileTrackFractions(first, n);
      expect(fractionsMatch(first, second)).toBe(true);
    });
  }

  for (const n of PANE_COUNTS_TO_TEST) {
    it(`stabilizes after one reconciliation for ${n} panes (grid mode)`, () => {
      const cols = Math.min(2, n);
      const rows = Math.ceil(n / cols);
      const prevRows = Math.max(1, rows - 1);
      const prevColFractions = reconcileTrackFractions(
        Array.from({ length: cols }, () => 1 / cols),
        cols,
      );
      const prevRowFractions = reconcileTrackFractions(
        Array.from({ length: prevRows }, () => 1 / prevRows),
        prevRows,
      );
      // Reconcile to current layout
      const colFirst = reconcileTrackFractions(prevColFractions, cols);
      const rowFirst = reconcileTrackFractions(prevRowFractions, rows);
      // Second pass
      const colSecond = reconcileTrackFractions(colFirst, cols);
      const rowSecond = reconcileTrackFractions(rowFirst, rows);
      expect(fractionsMatch(colFirst, colSecond)).toBe(true);
      expect(fractionsMatch(rowFirst, rowSecond)).toBe(true);
    });
  }
});
