import { describe, expect, it } from "vitest";
import { beatPositionAtProgress, buildBeatDurations, parseBeat, progressAtBeat, roomNavigationSuffix } from "./cycle";

describe("cycle model", () => {
  it("scales authored beat durations to the cycle length", () => {
    expect(buildBeatDurations([{ dwellSeconds: 10 }, { dwellSeconds: 20 }], 60)).toEqual([10, 20]);
    expect(buildBeatDurations([{ dwellSeconds: 10 }, { dwellSeconds: 20 }], 30)).toEqual([5, 10]);
  });

  it("uses the same weighted positions for elapsed time and seeking", () => {
    const durations = buildBeatDurations([{ dwellSeconds: 10 }, { dwellSeconds: 20 }], 60);
    expect(beatPositionAtProgress(0, durations)).toMatchObject({ index: 0, progress: 0 });
    expect(beatPositionAtProgress(0.5, durations)).toMatchObject({ index: 1, progress: 0.25, startSeconds: 10 });
    expect(progressAtBeat(1, durations)).toBeCloseTo(1 / 3);
    expect(beatPositionAtProgress(progressAtBeat(1, durations), durations)).toMatchObject({ index: 1, progress: 0 });
  });

  it("clamps URL beat values to available sections", () => {
    expect(parseBeat("2", 3)).toBe(2);
    expect(parseBeat("99", 3)).toBe(2);
    expect(parseBeat("-2", 3)).toBe(0);
    expect(parseBeat("bad", 3)).toBe(0);
  });

  it("resets the beat but preserves other room navigation settings", () => {
    expect(roomNavigationSuffix("?beat=3&cycle=45&samples=mark")).toBe("?cycle=45&samples=mark");
    expect(roomNavigationSuffix("?beat=3")).toBe("");
  });
});
