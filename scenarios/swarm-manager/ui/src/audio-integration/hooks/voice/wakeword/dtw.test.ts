// Unit tests for the DTW distance + score mappings. These are pure functions,
// so the tests run with no mic / Web Audio — they pin the distance-correctness
// guarantees from the wake-word scoring overhaul:
//   - symmetric step weighting consistent with the (n+m) normalization
//   - c0 (log-energy) excluded from the comparison Euclidean
//   - calibrated / uncalibrated score maps are monotonic and hit their anchors.

import { describe, expect, it } from "vitest";

import { dtwDistance, calibratedScore, uncalibratedScore } from "./dtw";
import { SCORE_MIDPOINT_Z, type EngineCalibration } from "./types";

/** Deterministic 13-coeff frame from a seed (no Math.random → reproducible). */
function frame(seed: number): number[] {
  const out: number[] = [];
  for (let c = 0; c < 13; c++) {
    out.push(Math.sin(seed * 0.7 + c * 1.3) * 2 + Math.cos(seed * 0.31 + c));
  }
  return out;
}

function makeSeq(n: number, offset = 0): number[][] {
  return Array.from({ length: n }, (_, i) => frame(i + offset));
}

describe("dtwDistance", () => {
  it("returns ~0 for identical sequences", () => {
    const seq = makeSeq(40);
    expect(dtwDistance(seq, seq)).toBeLessThan(1e-9);
  });

  it("is ~0 under pure time-warping (frame duplication)", () => {
    // Symmetric weighting + (n+m) normalization means a time-stretched copy
    // (every frame duplicated) still aligns at zero cost.
    const seq = makeSeq(30);
    const stretched = seq.flatMap((f) => [f, f]);
    expect(dtwDistance(seq, stretched)).toBeLessThan(1e-9);
  });

  it("reaches the corner (finite distance) for very unequal lengths", () => {
    const a = makeSeq(10);
    const b = makeSeq(25, 100);
    const d = dtwDistance(a, b);
    expect(Number.isFinite(d)).toBe(true);
    expect(d).toBeGreaterThan(0);
  });

  it("excludes c0 by default but counts it when startCoeff=0", () => {
    // Two sequences identical except for a constant offset on coefficient 0
    // (the log-energy). With the default startCoeff (1) that offset is invisible.
    const base = makeSeq(20);
    const c0Shifted = base.map((f) => [(f[0] ?? 0) + 5, ...f.slice(1)]);
    expect(dtwDistance(base, c0Shifted)).toBeLessThan(1e-9); // c0 excluded
    expect(dtwDistance(base, c0Shifted, undefined, 0)).toBeGreaterThan(0.1); // c0 included
  });

  it("returns Infinity for an empty sequence", () => {
    expect(dtwDistance([], makeSeq(5))).toBe(Infinity);
    expect(dtwDistance(makeSeq(5), [])).toBe(Infinity);
  });
});

describe("calibratedScore", () => {
  const cal: EngineCalibration = { kind: "mfcc-v1", mu: 2, sigma: 0.5 };

  it("scores a distance at the enrollment mean very high", () => {
    expect(calibratedScore(cal.mu, cal)).toBeGreaterThanOrEqual(0.85);
  });

  it("crosses 0.5 at mu + SCORE_MIDPOINT_Z*sigma", () => {
    const atMidpoint = calibratedScore(cal.mu + SCORE_MIDPOINT_Z * cal.sigma, cal);
    expect(atMidpoint).toBeGreaterThan(0.45);
    expect(atMidpoint).toBeLessThan(0.55);
  });

  it("scores a far-out distance (mu + 4*sigma) low", () => {
    expect(calibratedScore(cal.mu + 4 * cal.sigma, cal)).toBeLessThanOrEqual(0.3);
  });

  it("is monotonically non-increasing in distance", () => {
    let prev = Infinity;
    for (let d = 0; d <= 6; d += 0.25) {
      const s = calibratedScore(d, cal);
      expect(s).toBeLessThanOrEqual(prev + 1e-12);
      prev = s;
    }
  });

  it("clamps to [0,1] and maps non-finite distance to 0", () => {
    expect(calibratedScore(Infinity, cal)).toBe(0);
    expect(calibratedScore(-100, cal)).toBeLessThanOrEqual(1);
    expect(calibratedScore(1000, cal)).toBeGreaterThanOrEqual(0);
  });
});

describe("uncalibratedScore", () => {
  it("scores zero distance near 1 and large distance near 0", () => {
    expect(uncalibratedScore(0)).toBeGreaterThan(0.9);
    expect(uncalibratedScore(10)).toBeLessThan(0.05);
  });

  it("is monotonically non-increasing and maps non-finite to 0", () => {
    expect(uncalibratedScore(Infinity)).toBe(0);
    let prev = Infinity;
    for (let d = 0; d <= 6; d += 0.5) {
      const s = uncalibratedScore(d);
      expect(s).toBeLessThanOrEqual(prev + 1e-12);
      prev = s;
    }
  });
});
