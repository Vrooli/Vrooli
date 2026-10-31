import { describe, expect, it } from "vitest";
import { settlementFor, summarizeCriteria } from "./review-settlement";

describe("settlementFor", () => {
  it("treats absent evidence as unsettled, never as settled", () => {
    // The optimistic reading is how an item ships with an unproven claim.
    expect(settlementFor([])).toBe("unsettled");
  });

  it("settles only when every piece of evidence settles", () => {
    expect(settlementFor([{ settlement: "settled" }, { settlement: "settled" }])).toBe("settled");
    expect(settlementFor([{ settlement: "settled" }, { settlement: "pending" }])).toBe("pending");
  });

  it("lets a single refutation outrank any amount of settled evidence", () => {
    expect(settlementFor([{ settlement: "settled" }, { settlement: "refuted" }])).toBe("refuted");
  });

  it("does not count unavailable evidence as settled", () => {
    expect(settlementFor([{ settlement: "unavailable" }])).toBe("pending");
  });

  it("treats evidence with no settlement recorded as pending", () => {
    expect(settlementFor([{}])).toBe("pending");
  });
});

describe("summarizeCriteria", () => {
  const criteria = [{ id: "c1", gherkin: "one" }, { id: "c2", gherkin: "two" }, { id: "c3", gherkin: "three" }];

  it("attributes evidence to its criterion and counts the unsettled", () => {
    const summary = summarizeCriteria(criteria, [
      { criterion_id: "c1", settlement: "settled" },
      { criterion_id: "c2", settlement: "pending" },
    ]);
    expect(summary.rows.map((row) => row.state)).toEqual(["settled", "pending", "unsettled"]);
    expect(summary.rows.map((row) => row.evidenceCount)).toEqual([1, 1, 0]);
    expect(summary.unsettled).toBe(2);
  });

  it("reports every criterion unsettled when there is no evidence at all", () => {
    expect(summarizeCriteria(criteria, []).unsettled).toBe(3);
  });

  it("reports zero unsettled when all criteria are proven", () => {
    const summary = summarizeCriteria(criteria, criteria.map((criterion) => ({ criterion_id: criterion.id, settlement: "settled" as const })));
    expect(summary.unsettled).toBe(0);
  });

  it("ignores evidence that belongs to no listed criterion", () => {
    const summary = summarizeCriteria([{ id: "c1" }], [{ criterion_id: "other", settlement: "settled" }]);
    expect(summary.rows[0]?.evidenceCount).toBe(0);
    expect(summary.unsettled).toBe(1);
  });

  it("returns an empty summary for an item with no criteria", () => {
    expect(summarizeCriteria([], [{ criterion_id: "c1", settlement: "settled" }])).toEqual({ rows: [], unsettled: 0 });
  });
});
