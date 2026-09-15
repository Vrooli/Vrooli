import { describe, expect, it } from "vitest";
import { authoredSample } from "../test-utils/readings";
import { ladderFixture, ladderReading } from "../test-utils/ladder";
import { finishLabel, lastOpening, ladderOf, nextRung, openedBy, rungsAfterNext, statusLabel, stillIdeas } from "./ladder";

describe("ladder helpers", () => {
  it("prefers the measured ladder and marks the authored one as sampled", () => { // [REQ:CC-P1-016]
    expect(ladderOf(ladderReading())?.sampled).toBe(false);
    const sampled = ladderOf(ladderReading({ ladder: undefined, sample: { ...authoredSample(4), ladder: ladderFixture() } }));
    expect(sampled?.sampled).toBe(true);
    expect(ladderOf(ladderReading({ ladder: undefined }))).toBeNull();
  });
  it("names the next rung and the rungs after it", () => {
    const ladder = ladderFixture();
    expect(nextRung(ladder)?.name).toBe("web-console");
    expect(rungsAfterNext(ladder).map((rung) => rung.rank)).toEqual([2, 3, 4, 5]);
    expect(nextRung(ladderFixture({ nextRank: 0 }))).toBeNull();
    expect(rungsAfterNext(ladderFixture({ nextRank: 0 }))).toEqual([]);
  });
  it("counts what is open by a rank and never counts an unscheduled unlock", () => {
    const opened = openedBy(ladderFixture(), 4);
    expect(opened.ramp).toEqual({ open: 2, total: 3 });
    expect(opened.stream).toEqual({ open: 2, total: 2 });
    expect(opened.audience).toEqual({ open: 2, total: 3 });
    expect(lastOpening(ladderFixture()).map((unlock) => unlock.name)).toEqual(["personal"]);
  });
  it("speaks Offer Desk's vocabulary in plain words", () => {
    expect(statusLabel("TRIGGER_MET")).toBe("Trigger met");
    expect(statusLabel("SOMETHING_NEW")).toBe("something new");
    expect(finishLabel("OPERATOR_FACING")).toBe("operator-facing");
    expect(finishLabel("PARTNER_FACING")).toBe("partner facing");
    expect(finishLabel(undefined)).toBe("");
    expect(stillIdeas(ladderFixture())).toBe(4);
  });
  it("names nothing as last to open when no scheduled rung opens anything", () => {
    expect(lastOpening(ladderFixture({ reach: [{ kind: "ramp", name: "mobile", opensAt: 0 }] }))).toEqual([]);
    expect(lastOpening(ladderFixture({ reach: [] }))).toEqual([]);
  });
});
