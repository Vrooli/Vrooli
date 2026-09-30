import { describe, expect, it } from "vitest";
import type { Reading } from "./api";
import type { LadderReading, LadderRung } from "./ladder";
import { beatReadingSeconds, READING_SECONDS, roomReadingSeconds } from "./readingTime";

const { headline, line, token, tile, stripCap, beatCap } = READING_SECONDS;

const reading = (id: string, extra: Partial<Reading> = {}): Reading => ({ id, label: id, value: 1, ...extra }) as Reading;
const panel = (id: string, rows: number) => reading(id, { kind: "panel", rows: Array.from({ length: rows }, (_, index) => ({ key: `${index}`, label: `row ${index}`, value: index, share: 0.1 })) });
const rung = (rank: number, blockers: number): LadderRung => ({
  rank, id: `r${rank}`, name: `rung ${rank}`, status: "IDEA", ramps: ["ramp"], streams: ["stream"], audiences: ["audience"],
  blockers: Array.from({ length: blockers }, (_, index) => ({ name: `work ${index}`, status: "ACTIVE", urgency: rank })),
  goals: [], readiness: { reported: false },
});
const ladder = (blockers: number, lanes = 0): Reading => {
  const value: LadderReading = {
    rungs: [rung(1, blockers), rung(2, 0)], nextRank: 1, enabling: [], unscheduled: [],
    reach: Array.from({ length: lanes }, (_, index) => ({ kind: index % 2 ? "ramp" : "stream", name: `lane ${index}`, opensAt: 1 })),
  };
  return reading("release_ladder", { kind: "ladder", ladder: value });
};

describe("beat reading time", () => { // [REQ:CC-P1-018]
  it("reads a wall figure as its headline and a glance across the strip", () => {
    expect(beatReadingSeconds(reading("visitors"), "standard", 5)).toBeCloseTo(headline + 5 * tile);
  });

  it("caps the strip's share so a crowded room does not lengthen every beat", () => {
    expect(beatReadingSeconds(reading("visitors"), "standard", 40)).toBeCloseTo(headline + stripCap);
  });

  it("gives a panel a line per row", () => {
    expect(beatReadingSeconds(panel("countries", 8), "standard", 0) - beatReadingSeconds(panel("countries", 2), "standard", 0)).toBeCloseTo(6 * line);
  });

  it("gives Next Rung a line per blocker, fact and following rung, and the Reach Map a line per lane", () => {
    expect(beatReadingSeconds(ladder(4), "standard", 0)).toBeCloseTo(headline + (line + 3 * token) + 4 * line + 2 * line + line);
    expect(beatReadingSeconds(ladder(6), "standard", 0) - beatReadingSeconds(ladder(2), "standard", 0)).toBeCloseTo(4 * line);
    expect(beatReadingSeconds(ladder(0, 10), "wide", 0)).toBeCloseTo(headline + 10 * line + 2 * token);
  });

  it("caps a beat so one long list cannot stall the room", () => {
    expect(beatReadingSeconds(panel("everything", 200), "standard", 10)).toBe(beatCap);
  });

  it("gives a beat with no hero no reading time of its own", () => {
    expect(beatReadingSeconds(null, "standard", 3)).toBe(0);
  });

  it("chooses hero and strip as the room does: a ladder beat has no strip", () => {
    const readings = [reading("visitors"), panel("countries", 3), ladder(2)];
    const [scalar, rows, rungs] = roomReadingSeconds([{ hero: "visitors" }, { hero: "countries" }, { hero: "release_ladder" }], readings);
    expect(scalar).toBeCloseTo(headline + 2 * tile);
    expect(rows).toBeCloseTo(headline + 3 * line + 2 * tile);
    expect(rungs).toBeCloseTo(beatReadingSeconds(ladder(2), "standard", 0));
  });

  it("times only the readings assigned to each beat", () => {
    const readings = [reading("hero"), reading("included"), reading("outside")];
    const [scoped, legacy] = roomReadingSeconds([
      { hero: "hero", readingIds: ["hero", "included"] },
      { hero: "hero" },
    ], readings);
    expect(scoped).toBeCloseTo(headline + tile);
    expect(legacy).toBeCloseTo(headline + 2 * tile);
  });
});
