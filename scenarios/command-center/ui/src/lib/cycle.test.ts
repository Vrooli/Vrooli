import { describe, expect, it } from "vitest";
import { beatEndProgress, beatPositionAtProgress, buildBeatDurations, crossesBeat, cycleScale, parseBeat, progressAtBeat, remapProgress, roomNavigationSuffix, tickCycle } from "./cycle";

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

  it("scales every authored time by the same cycle factor", () => {
    expect(cycleScale(60)).toBe(1);
    expect(cycleScale(30)).toBe(0.5);
    expect(cycleScale(1)).toBe(5 / 60);
    expect(cycleScale(120)).toBe(2);
  });

  it("lengthens a beat to its reading time, never below its authored dwell", () => { // [REQ:CC-P1-018]
    const beats = [{ dwellSeconds: 10 }, { dwellSeconds: 16 }, {}];
    expect(buildBeatDurations(beats, 60, [22, 12, 30])).toEqual([22, 16, 30]);
    expect(buildBeatDurations(beats, 30, [22, 12, 30])).toEqual([11, 8, 15]);
    expect(buildBeatDurations(beats, 60)).toEqual([10, 16, 20]);
  });

  it("keeps the current beat and its progress when durations change", () => { // [REQ:CC-P1-018]
    const from = [10, 10, 10, 10];
    const to = [12, 20, 10, 30];
    const at = beatPositionAtProgress(0.6, from);
    const remapped = beatPositionAtProgress(remapProgress(0.6, from, to), to);
    expect(remapped.index).toBe(at.index);
    expect(remapped.progress).toBeCloseTo(at.progress);
    expect(remapProgress(0, from, to)).toBe(0);
    expect(remapProgress(0.5, from, [10, 10])).toBe(0.5);
  });

  it("detects where a held beat must wait: its own boundary and the room's end", () => { // [REQ:CC-P1-017]
    const durations = buildBeatDurations([{ dwellSeconds: 10 }, { dwellSeconds: 20 }], 60);
    expect(crossesBeat(0.1, 0.2, durations)).toBe(false);
    expect(crossesBeat(0.33, 0.34, durations)).toBe(true);
    expect(crossesBeat(0.9, 1, durations)).toBe(true);
    expect(crossesBeat(0.5, 0.9, [])).toBe(false);
    expect(crossesBeat(0.9, 1.01, [])).toBe(true);
  });

  it("draws a held beat's segment full without moving into the next beat", () => {
    const durations = [10, 20, 30];
    for (let index = 0; index < durations.length; index += 1) {
      const end = beatEndProgress(index, durations);
      expect(beatPositionAtProgress(end, durations).index).toBe(index);
      expect(beatPositionAtProgress(end, durations).progress).toBeCloseTo(1, 3);
    }
    expect(beatEndProgress(2, durations)).toBeCloseTo(0.999999, 6);
  });
});

describe("cycle clock", () => { // [REQ:CC-P1-008] [REQ:CC-P1-017]
  const durations = [10, 20, 30];
  const dwellMs = 60_000;

  it("advances progress from the wall clock every frame and never backwards", () => {
    let progress = 0;
    for (let frame = 1; frame <= 120; frame += 1) {
      const tick = tickCycle({ now: frame * 16, startedAt: 0, dwellMs, progress, durations, holding: false, heldSince: null, maxHoldMs: 90_000 });
      expect(tick.progress).toBeGreaterThanOrEqual(progress);
      expect(tick.navigate).toBe(false);
      progress = tick.progress;
    }
    expect(progress).toBeCloseTo((120 * 16) / dwellMs, 5);
  });

  it("waits a held beat at its segment's end, then releases into the next beat", () => {
    const firstEnd = beatEndProgress(0, durations);
    // The clock has already crossed into the second beat; the strip is still reading.
    const held = tickCycle({ now: 11_000, startedAt: 0, dwellMs, progress: 0.16, durations, holding: true, heldSince: null, maxHoldMs: 90_000 });
    expect(held.held).toBe(true);
    expect(held.progress).toBeCloseTo(firstEnd, 6);
    expect(beatPositionAtProgress(held.progress, durations).index).toBe(0);
    // The same frame, one tick later, is still held at the same point.
    const stillHeld = tickCycle({ now: 11_016, startedAt: 0, dwellMs, progress: held.progress, durations, holding: true, heldSince: held.heldSince, maxHoldMs: 90_000 });
    expect(stillHeld.held).toBe(true);
    expect(stillHeld.progress).toBeCloseTo(firstEnd, 6);
    // The strip releases; the same moment now advances into the second beat.
    const released = tickCycle({ now: 11_032, startedAt: 0, dwellMs, progress: stillHeld.progress, durations, holding: false, heldSince: stillHeld.heldSince, maxHoldMs: 90_000 });
    expect(released.held).toBe(false);
    expect(beatPositionAtProgress(released.progress, durations).index).toBe(1);
  });

  it("stops holding after the bound so a stuck hold cannot freeze the room forever", () => {
    const held = tickCycle({ now: 200_000, startedAt: 200_000 - 11_000, dwellMs, progress: 0.16, durations, holding: true, heldSince: 0, maxHoldMs: 90_000 });
    expect(held.held).toBe(false);
    expect(held.progress).toBeGreaterThan(0.16);
  });

  it("navigates exactly once when the room runs out", () => {
    const end = tickCycle({ now: dwellMs + 5, startedAt: 0, dwellMs, progress: 0.99, durations, holding: false, heldSince: null, maxHoldMs: 90_000 });
    expect(end.progress).toBe(1);
    expect(end.navigate).toBe(true);
  });

  it("keeps an unsegmented room's rail moving", () => {
    const tick = tickCycle({ now: 15_000, startedAt: 0, dwellMs, progress: 0, durations: [], holding: false, heldSince: null, maxHoldMs: 90_000 });
    expect(tick.progress).toBeCloseTo(0.25, 5);
    expect(tick.navigate).toBe(false);
  });
});
