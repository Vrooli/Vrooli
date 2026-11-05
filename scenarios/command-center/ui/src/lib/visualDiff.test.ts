import { describe, expect, it } from "vitest";
import { comparePixels, createDeterministicClock, VISUAL_DIFF_PIN } from "./visualDiff";

describe("visual diff safety net", () => {
  it("is zero for identical frames and catches a one-pixel perturbation", () => {
    const baseline = new Uint8ClampedArray([0, 10, 20, 255, 40, 50, 60, 255]);
    expect(comparePixels(baseline, baseline)).toMatchObject({ differentPixels: 0, maxDelta: 0 });
    const candidate = new Uint8ClampedArray(baseline); candidate[4] = (candidate[4] ?? 0) + 1;
    expect(comparePixels(baseline, candidate)).toMatchObject({ differentPixels: 1, maxDelta: 1 });
  });
  it("keeps the seed and time samples pinned", () => { expect(VISUAL_DIFF_PIN).toEqual({ seed: "command-center-milestone-1", timestampsMs: [0, 250, 1000, 2500, 5000, 10000, 14000] }); });
  it("provides a frozen virtual clock for capture harnesses", () => { const clock = createDeterministicClock(); expect(clock.now()).toBe(0); clock.advance(2500); expect(clock.now()).toBe(2500); });
});
