import { describe, it, expect } from "vitest";
import {
  parseWorkshopRound,
  buildWorkshopRoundContent,
  getPendingDecisionCount,
} from "./workshop-files";
import type { WorkshopRound } from "../types/domain";

const makeRound = (overrides?: Partial<WorkshopRound>): WorkshopRound => ({
  round: 1,
  generated_at: "2026-03-20T00:00:00Z",
  readiness: {
    problem_clarity: 2,
    scope_defined: 1,
    approach_solid: 1,
    testable: 0,
    risk_awareness: 0,
  },
  items: [],
  ...overrides,
});

describe("parseWorkshopRound", () => {
  it("parses valid JSON into a WorkshopRound", () => {
    const round = makeRound({
      items: [
        { id: "q1", type: "decision", topic: "Scope", text: "Pick a scope", options: [], selected: null },
      ],
    });
    const content = JSON.stringify(round);
    const result = parseWorkshopRound(content);

    expect(result.round).not.toBeNull();
    expect(result.error).toBeUndefined();
    expect(result.round?.round).toBe(1);
    expect(result.round?.items).toHaveLength(1);
    expect(result.round?.items[0]?.type).toBe("decision");
  });

  it("returns null round for empty input", () => {
    expect(parseWorkshopRound("")).toEqual({ round: null });
    expect(parseWorkshopRound(null)).toEqual({ round: null });
    expect(parseWorkshopRound(undefined)).toEqual({ round: null });
    expect(parseWorkshopRound("   ")).toEqual({ round: null });
  });

  it("returns an error for completely malformed JSON", () => {
    const result = parseWorkshopRound("not json at all {{{{");
    expect(result.round).toBeNull();
    expect(result.error).toBeDefined();
  });

  it("recovers from truncated JSON", () => {
    const round = makeRound({
      items: [
        { id: "q1", type: "decision", topic: "A", text: "First" },
        { id: "q2", type: "info", text: "Second" },
      ],
    });
    const full = JSON.stringify(round, null, 2);
    // Cut off the last ~20 characters to simulate a truncated write
    const truncated = full.slice(0, full.length - 20);

    const result = parseWorkshopRound(truncated);
    // Should recover at least the first item
    expect(result.round).not.toBeNull();
    expect(result.error).toBeDefined();
    expect(result.error).toMatch(/truncated/i);
    expect(result.round?.items.length).toBeGreaterThanOrEqual(1);
  });

  it("normalizes items without explicit type to type 'info'", () => {
    const content = JSON.stringify({
      round: 1,
      generated_at: "2026-03-20T00:00:00Z",
      readiness: { problem_clarity: 0, scope_defined: 0, approach_solid: 0, testable: 0, risk_awareness: 0 },
      items: [{ id: "x1" }],
    });
    const result = parseWorkshopRound(content);
    expect(result.round?.items[0]?.type).toBe("info");
  });

  it("creates empty items array when items field is missing", () => {
    const content = JSON.stringify({
      round: 1,
      generated_at: "2026-03-20T00:00:00Z",
      readiness: { problem_clarity: 0, scope_defined: 0, approach_solid: 0, testable: 0, risk_awareness: 0 },
    });
    const result = parseWorkshopRound(content);
    expect(result.round?.items).toEqual([]);
  });
});

describe("buildWorkshopRoundContent", () => {
  it("produces valid JSON that re-parses identically", () => {
    const round = makeRound({
      items: [
        { id: "d1", type: "decision", topic: "DB", text: "Pick DB", options: [{ key: "pg", label: "Postgres", rationale: "Solid" }], selected: "pg", freeform: null, notes: null },
        { id: "i1", type: "info", text: "Note about constraints" },
      ],
    });

    const content = buildWorkshopRoundContent(round);
    const reparsed = parseWorkshopRound(content);

    expect(reparsed.error).toBeUndefined();
    expect(reparsed.round).not.toBeNull();
    expect(reparsed.round?.round).toBe(round.round);
    expect(reparsed.round?.items).toHaveLength(2);
    expect(reparsed.round?.items[0]?.selected).toBe("pg");
  });
});

describe("getPendingDecisionCount", () => {
  it("returns 0 when there are no items", () => {
    expect(getPendingDecisionCount(makeRound())).toBe(0);
  });

  it("counts decisions with no selection as pending", () => {
    const round = makeRound({
      items: [
        { id: "d1", type: "decision", selected: null },
        { id: "d2", type: "decision", selected: "" },
        { id: "d3", type: "decision", selected: "  " },
      ],
    });
    expect(getPendingDecisionCount(round)).toBe(3);
  });

  it("does not count decided items", () => {
    const round = makeRound({
      items: [
        { id: "d1", type: "decision", selected: "option_a" },
        { id: "d2", type: "decision", selected: null },
      ],
    });
    expect(getPendingDecisionCount(round)).toBe(1);
  });

  it("ignores info items entirely", () => {
    const round = makeRound({
      items: [
        { id: "i1", type: "info", text: "Just FYI" },
        { id: "i2", type: "info", text: "Another note" },
        { id: "d1", type: "decision", selected: "yes" },
      ],
    });
    expect(getPendingDecisionCount(round)).toBe(0);
  });

  it("handles a mix of decided, undecided, and info items", () => {
    const round = makeRound({
      items: [
        { id: "i1", type: "info" },
        { id: "d1", type: "decision", selected: "a" },
        { id: "d2", type: "decision", selected: null },
        { id: "d3", type: "decision", selected: "" },
        { id: "i2", type: "info" },
        { id: "d4", type: "decision", selected: "b" },
      ],
    });
    expect(getPendingDecisionCount(round)).toBe(2);
  });
});
