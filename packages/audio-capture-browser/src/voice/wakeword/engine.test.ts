import { describe, expect, it } from "vitest";

import { MfccDtwEngine, normalizeFeatures } from "./engine";
import type { AudioFeatures } from "./types";

function features(values: number[][]): AudioFeatures {
  return { kind: "mfcc-v1", data: values, sampleRate: 16_000, durationSec: 1 };
}

describe("package-owned wake-word engine", () => {
  it("normalizes each coefficient without changing frame shape", () => {
    const normalized = normalizeFeatures([
      [1, 10, 4],
      [2, 12, 4],
      [3, 14, 4],
    ]);

    expect(normalized).toHaveLength(3);
    expect(normalized[0]).toHaveLength(3);
    for (let coefficient = 0; coefficient < 3; coefficient += 1) {
      const mean = normalized.reduce((sum, frame) => sum + (frame[coefficient] ?? 0), 0) / normalized.length;
      expect(mean).toBeCloseTo(0, 8);
    }
  });

  it("matches the closest template and refuses empty candidates", () => {
    const engine = new MfccDtwEngine();
    const candidate = features([
      [0, 1, 2],
      [1, 2, 3],
      [2, 3, 4],
    ]);
    const near = features([
      [0, 1, 2],
      [1, 2, 3],
      [2, 3, 4],
    ]);
    const far = features([
      [20, -10, 7],
      [18, -8, 9],
      [16, -6, 11],
    ]);

    expect(engine.compareBest(candidate, [far, near], 0.5).isMatch).toBe(true);
    expect(engine.compareBest(features([]), [near], 0.5)).toEqual({ score: 0, isMatch: false });
    expect(engine.compareBest(candidate, [], 0.5)).toEqual({ score: 0, isMatch: false });
  });

  it("calibrates from multiple non-empty enrollment samples", () => {
    const engine = new MfccDtwEngine();
    const first = features([[0, 1], [1, 2], [2, 3]]);
    const second = features([[0, 1], [1, 2], [2.1, 3.1]]);

    expect(engine.calibrate([first])).toBeNull();
    expect(engine.calibrate([first, second])).toMatchObject({ kind: "mfcc-v1" });
  });
});
