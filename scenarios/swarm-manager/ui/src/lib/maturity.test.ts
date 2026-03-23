import { describe, it, expect } from "vitest";
import { buildReadinessData, computeNextNudge, READINESS_DIMENSIONS } from "./maturity";
import type { ReadinessIndicatorData } from "./maturity";
import type { MaturityItemSummary, ReadinessDimension } from "../types/domain";

const allScores = (n: number): Record<ReadinessDimension, number> => ({
  problem_clarity: n,
  scope_defined: n,
  approach_solid: n,
  testable: n,
  risk_awareness: n,
});

const makeSummary = (overrides?: Partial<MaturityItemSummary>): MaturityItemSummary => ({
  kind: "idea",
  name: "test-item",
  title: "Test Item",
  rounds_completed: 1,
  raw_scores: allScores(2),
  effective_scores: allScores(2),
  ready: false,
  pending_items: 0,
  has_plan: false,
  ...overrides,
});

const makeData = (overrides?: Partial<ReadinessIndicatorData>): ReadinessIndicatorData => ({
  rawScores: allScores(2),
  effectiveScores: allScores(2),
  roundsCompleted: 1,
  ready: false,
  pendingItems: 0,
  hasPlan: false,
  nextNudge: null,
  ...overrides,
});

describe("buildReadinessData", () => {
  it("maps MaturityItemSummary fields to ReadinessIndicatorData", () => {
    const summary = makeSummary({
      rounds_completed: 3,
      raw_scores: allScores(1),
      effective_scores: allScores(2),
      ready: false,
      pending_items: 2,
      has_plan: true,
    });

    const result = buildReadinessData(summary);

    expect(result.roundsCompleted).toBe(3);
    expect(result.rawScores).toEqual(allScores(1));
    expect(result.effectiveScores).toEqual(allScores(2));
    expect(result.ready).toBe(false);
    expect(result.pendingItems).toBe(2);
    expect(result.hasPlan).toBe(true);
  });

  it("computes nextNudge automatically", () => {
    const summary = makeSummary({ rounds_completed: 0 });
    const result = buildReadinessData(summary);
    expect(result.nextNudge).toMatch(/Run Workshop/i);
  });

  it("sets nextNudge for a ready item", () => {
    const summary = makeSummary({
      ready: true,
      effective_scores: allScores(3),
    });
    const result = buildReadinessData(summary);
    expect(result.nextNudge).toMatch(/Ready for execution/);
  });
});

describe("computeNextNudge", () => {
  it("returns workshop start nudge when no rounds completed", () => {
    const result = computeNextNudge(makeData({ roundsCompleted: 0 }));
    expect(result).toBe("Run Workshop to start refining this item");
  });

  it("returns pending items nudge when items are pending", () => {
    const result = computeNextNudge(makeData({ pendingItems: 3 }));
    expect(result).toMatch(/Respond to 3 pending items/);
  });

  it("uses singular form for 1 pending item", () => {
    const result = computeNextNudge(makeData({ pendingItems: 1 }));
    expect(result).toMatch(/1 pending item from/);
    expect(result).not.toMatch(/items/);
  });

  it("returns ready nudge when ready", () => {
    const result = computeNextNudge(makeData({ ready: true, effectiveScores: allScores(3) }));
    expect(result).toMatch(/Ready for execution/);
  });

  it("lists weak dimensions when some scores are below 3", () => {
    const scores = allScores(3);
    scores.testable = 1;
    scores.risk_awareness = 2;
    const result = computeNextNudge(makeData({ effectiveScores: scores }));
    expect(result).toMatch(/strengthen/i);
    expect(result).toMatch(/Testability/);
    expect(result).toMatch(/Risk Awareness/);
  });

  it("returns null when all scores are 3 but not marked ready", () => {
    const result = computeNextNudge(makeData({ effectiveScores: allScores(3), ready: false }));
    expect(result).toBeNull();
  });
});
