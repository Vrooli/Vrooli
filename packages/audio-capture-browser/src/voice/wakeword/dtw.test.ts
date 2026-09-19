import { describe, expect, it } from "vitest";

import { calibratedScore, dtwDistance, uncalibratedScore } from "./dtw";
import { SCORE_MIDPOINT_Z, type EngineCalibration } from "./types";

function frame(seed: number): number[] {
  return Array.from({ length: 13 }, (_, index) => Math.sin(seed * 0.7 + index * 1.3));
}

function sequence(length: number, offset = 0): number[][] {
  return Array.from({ length }, (_, index) => frame(index + offset));
}

describe("wake-word DTW scoring", () => {
  it("is symmetric, time-warp tolerant, and rejects empty input", () => {
    const input = sequence(24);
    const stretched = input.flatMap((item) => [item, item]);

    expect(dtwDistance(input, input)).toBeLessThan(1e-9);
    expect(dtwDistance(input, stretched)).toBeLessThan(1e-9);
    expect(dtwDistance(input, sequence(24, 100))).toBeGreaterThan(0);
    expect(dtwDistance([], input)).toBe(Infinity);
    expect(dtwDistance(input, [])).toBe(Infinity);
  });

  it("reaches the corner for very unequal finite sequences", () => {
    const distance = dtwDistance(sequence(10), sequence(25, 100));
    expect(Number.isFinite(distance)).toBe(true);
    expect(distance).toBeGreaterThan(0);
  });

  it("keeps c0 energy offsets out of the default distance", () => {
    const input = sequence(12);
    const shifted = input.map((item) => [(item[0] ?? 0) + 10, ...item.slice(1)]);

    expect(dtwDistance(input, shifted)).toBeLessThan(1e-9);
    expect(dtwDistance(input, shifted, undefined, 0)).toBeGreaterThan(0.1);
  });

  it("maps calibrated and uncalibrated scores monotonically", () => {
    const calibration: EngineCalibration = { kind: "mfcc-v1", mu: 2, sigma: 0.5 };
    expect(calibratedScore(2, calibration)).toBeGreaterThanOrEqual(0.85);
    expect(calibratedScore(2 + SCORE_MIDPOINT_Z * 0.5, calibration)).toBeCloseTo(0.5, 1);
    expect(calibratedScore(Infinity, calibration)).toBe(0);
    expect(uncalibratedScore(0)).toBeGreaterThan(0.9);

    let previous = Infinity;
    for (let distance = 0; distance <= 6; distance += 0.25) {
      const score = uncalibratedScore(distance);
      expect(score).toBeLessThanOrEqual(previous + 1e-12);
      previous = score;
    }
    expect(uncalibratedScore(10)).toBeLessThan(0.05);
    expect(uncalibratedScore(Infinity)).toBe(0);
  });
});
