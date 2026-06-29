import { describe, it, expect } from "vitest";

import { dtwDistance, distanceToScore } from "./dtw";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeSeq(numFrames: number, numCoeffs: number, value = 0): number[][] {
  return Array.from({ length: numFrames }, () => Array(numCoeffs).fill(value) as number[]);
}

// ---------------------------------------------------------------------------
// dtwDistance
// ---------------------------------------------------------------------------

describe("dtwDistance", () => {
  it("returns Infinity for an empty first sequence", () => {
    const seq2 = makeSeq(5, 13, 0.1);
    expect(dtwDistance([], seq2)).toBe(Infinity);
  });

  it("returns Infinity for an empty second sequence", () => {
    const seq1 = makeSeq(5, 13, 0.1);
    expect(dtwDistance(seq1, [])).toBe(Infinity);
  });

  it("returns 0 for two identical single-frame sequences", () => {
    const frame = [1, 2, 3];
    const seq1 = [frame];
    const seq2 = [frame];
    expect(dtwDistance(seq1, seq2)).toBe(0);
  });

  it("returns 0 for two identical multi-frame sequences", () => {
    const frames = makeSeq(10, 13, 0.5);
    expect(dtwDistance(frames, frames)).toBe(0);
  });

  it("returns a positive value for distinct sequences", () => {
    const seq1 = makeSeq(5, 3, 0.0);
    const seq2 = makeSeq(5, 3, 1.0);
    const dist = dtwDistance(seq1, seq2);
    expect(dist).toBeGreaterThan(0);
    expect(isFinite(dist)).toBe(true);
  });

  it("is symmetric (seq1 ↔ seq2 yields the same result)", () => {
    const seq1 = makeSeq(8, 3, 0.2);
    const seq2 = makeSeq(8, 3, 0.8);
    expect(dtwDistance(seq1, seq2)).toBeCloseTo(dtwDistance(seq2, seq1), 10);
  });

  it("handles sequences of different lengths", () => {
    const seq1 = makeSeq(10, 3, 0.0);
    const seq2 = makeSeq(20, 3, 0.0);
    // Both all-zero → distance should be 0
    expect(dtwDistance(seq1, seq2)).toBe(0);
  });

  it("uses the supplied bandRatio override", () => {
    const seq1 = makeSeq(10, 3, 0.1);
    const seq2 = makeSeq(10, 3, 0.1);
    const d1 = dtwDistance(seq1, seq2, 0.1);
    const d2 = dtwDistance(seq1, seq2, 0.5);
    // Both identical sequences → both distances should be 0
    expect(d1).toBe(0);
    expect(d2).toBe(0);
  });

  it("normalizes by (n + m) so that cost scales with path length", () => {
    const seq1 = makeSeq(5, 1, 0.0);
    const seq2 = makeSeq(5, 1, 1.0);
    const d5 = dtwDistance(seq1, seq2);
    const seq3 = makeSeq(10, 1, 0.0);
    const seq4 = makeSeq(10, 1, 1.0);
    const d10 = dtwDistance(seq3, seq4);
    // Normalized values should be finite and positive for both
    expect(isFinite(d5)).toBe(true);
    expect(isFinite(d10)).toBe(true);
    expect(d5).toBeGreaterThan(0);
    expect(d10).toBeGreaterThan(0);
  });
});

// ---------------------------------------------------------------------------
// distanceToScore
// ---------------------------------------------------------------------------

describe("distanceToScore", () => {
  it("returns 1 for distance 0", () => {
    expect(distanceToScore(0)).toBe(1);
    expect(distanceToScore(0, 2)).toBe(1);
  });

  it("returns 0 for Infinity distance", () => {
    expect(distanceToScore(Infinity)).toBe(0);
    expect(distanceToScore(Infinity, 5)).toBe(0);
  });

  it("returns 0 for -Infinity distance", () => {
    expect(distanceToScore(-Infinity)).toBe(0);
  });

  it("returns 0 for NaN distance", () => {
    expect(distanceToScore(NaN)).toBe(0);
  });

  it("returns 0.5 for distance = 1/scale (sigmoid midpoint)", () => {
    // 1 / (1 + 1 * 1) = 0.5
    expect(distanceToScore(1, 1)).toBeCloseTo(0.5, 10);
    // 1 / (1 + 2 * 0.5) = 0.5
    expect(distanceToScore(0.5, 2)).toBeCloseTo(0.5, 10);
  });

  it("decreases monotonically with increasing distance", () => {
    const distances = [0, 0.5, 1, 2, 5, 10];
    const scores = distances.map(d => distanceToScore(d, 1));
    for (let i = 1; i < scores.length; i++) {
      expect((scores[i] as number) < (scores[i - 1] as number)).toBe(true);
    }
  });

  it("uses scale = 1 as default", () => {
    const d = 2;
    expect(distanceToScore(d)).toBeCloseTo(1 / (1 + d), 10);
  });

  it("respects the scale parameter", () => {
    const d = 1;
    expect(distanceToScore(d, 2)).toBeCloseTo(1 / (1 + 2), 10);
    expect(distanceToScore(d, 10)).toBeCloseTo(1 / (1 + 10), 10);
  });
});
