import { describe, it, expect } from "vitest";
import { dtwDistance, distanceToScore } from "../wakeword/dtw";

/** Generate a synthetic MFCC-like sequence: numFrames x numCoeffs. */
function generateSequence(numFrames: number, numCoeffs: number, seed: number = 0): number[][] {
  const seq: number[][] = [];
  for (let f = 0; f < numFrames; f++) {
    const frame: number[] = [];
    for (let c = 0; c < numCoeffs; c++) {
      // Deterministic pseudo-random using simple hash
      frame.push(Math.sin(seed * 1000 + f * 13.7 + c * 7.3) * 10);
    }
    seq.push(frame);
  }
  return seq;
}

/** Stretch a sequence by repeating each frame `factor` times (simulates slower speech). */
function stretchSequence(seq: number[][], factor: number): number[][] {
  const stretched: number[][] = [];
  for (const frame of seq) {
    for (let i = 0; i < factor; i++) {
      stretched.push([...frame]);
    }
  }
  return stretched;
}

describe("dtwDistance", () => {
  it("returns 0 for identical sequences", () => {
    const seq = generateSequence(50, 13, 1);
    const dist = dtwDistance(seq, seq);
    expect(dist).toBeCloseTo(0, 5);
  });

  it("returns small distance for very similar sequences", () => {
    const seq1 = generateSequence(50, 13, 1);
    // Add small noise
    const seq2 = seq1.map(frame => frame.map(c => c + (Math.random() - 0.5) * 0.1));
    const dist = dtwDistance(seq1, seq2);
    expect(dist).toBeLessThan(0.5);
  });

  it("returns large distance for completely different sequences", () => {
    const seq1 = generateSequence(50, 13, 1);
    const seq2 = generateSequence(50, 13, 999);
    const dist = dtwDistance(seq1, seq2);
    expect(dist).toBeGreaterThan(1);
  });

  it("handles time-warped version of same signal", () => {
    const original = generateSequence(40, 13, 42);
    const stretched = stretchSequence(original, 2); // 2x slower

    const distSame = dtwDistance(original, original);
    const distWarped = dtwDistance(original, stretched);

    // Warped should have low distance (not zero, but much less than random)
    expect(distWarped).toBeLessThan(1);
    // And the identical distance should be the lowest
    expect(distSame).toBeLessThanOrEqual(distWarped);
  });

  it("handles sequences of different lengths", () => {
    const seq1 = generateSequence(30, 13, 1);
    const seq2 = generateSequence(50, 13, 1);
    // Should not throw and should return a finite distance
    const dist = dtwDistance(seq1, seq2);
    expect(isFinite(dist)).toBe(true);
  });

  it("returns Infinity for empty sequences", () => {
    expect(dtwDistance([], [[1, 2, 3]])).toBe(Infinity);
    expect(dtwDistance([[1, 2, 3]], [])).toBe(Infinity);
    expect(dtwDistance([], [])).toBe(Infinity);
  });

  it("is symmetric", () => {
    const seq1 = generateSequence(40, 13, 1);
    const seq2 = generateSequence(45, 13, 2);
    const dist12 = dtwDistance(seq1, seq2);
    const dist21 = dtwDistance(seq2, seq1);
    expect(dist12).toBeCloseTo(dist21, 5);
  });

  it("completes in reasonable time for 200-frame sequences", () => {
    const seq1 = generateSequence(200, 13, 1);
    const seq2 = generateSequence(200, 13, 2);
    const start = performance.now();
    dtwDistance(seq1, seq2);
    const elapsed = performance.now() - start;
    expect(elapsed).toBeLessThan(50); // 50ms budget
  });
});

describe("distanceToScore", () => {
  it("returns 1 for zero distance", () => {
    expect(distanceToScore(0)).toBe(1);
  });

  it("returns 0.5 for distance of 1 (scale=1)", () => {
    expect(distanceToScore(1, 1)).toBeCloseTo(0.5, 5);
  });

  it("returns value between 0 and 1 for positive distances", () => {
    for (const d of [0.1, 0.5, 1, 5, 10, 100]) {
      const score = distanceToScore(d);
      expect(score).toBeGreaterThan(0);
      expect(score).toBeLessThanOrEqual(1);
    }
  });

  it("score decreases as distance increases", () => {
    const s1 = distanceToScore(0.5);
    const s2 = distanceToScore(1.0);
    const s3 = distanceToScore(5.0);
    expect(s1).toBeGreaterThan(s2);
    expect(s2).toBeGreaterThan(s3);
  });

  it("returns 0 for Infinity", () => {
    expect(distanceToScore(Infinity)).toBe(0);
  });

  it("returns 0 for NaN", () => {
    expect(distanceToScore(NaN)).toBe(0);
  });

  it("higher scale makes score drop faster", () => {
    const lowScale = distanceToScore(1, 0.5);
    const highScale = distanceToScore(1, 2);
    expect(lowScale).toBeGreaterThan(highScale);
  });
});
