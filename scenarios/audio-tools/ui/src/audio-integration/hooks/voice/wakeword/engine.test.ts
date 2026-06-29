import { describe, it, expect } from "vitest";

import { applyCms, MfccDtwEngine, createWakeWordEngine } from "./engine";
import type { AudioFeatures } from "./types";
import { NUM_MFCC_COEFFICIENTS } from "./types";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeMfccs(numFrames: number, numCoeffs: number, fill: number | ((f: number, c: number) => number) = 0): number[][] {
  return Array.from({ length: numFrames }, (_, f) =>
    Array.from({ length: numCoeffs }, (__, c) =>
      typeof fill === "function" ? fill(f, c) : fill,
    ),
  );
}

function makeFeatures(mfccs: number[][]): AudioFeatures {
  return { kind: "mfcc-v1", data: mfccs, sampleRate: 16_000, durationSec: 1 };
}

// ---------------------------------------------------------------------------
// applyCms
// ---------------------------------------------------------------------------

describe("applyCms", () => {
  it("returns the input unchanged for an empty sequence", () => {
    expect(applyCms([])).toEqual([]);
  });

  it("returns the input unchanged for a single-frame sequence (mean = value → sub = 0)", () => {
    const mfccs = [[1, 2, 3]];
    const result = applyCms(mfccs);
    // After CMS: frame[c] = v - mean[c] = v - v = 0
    expect(result[0]).toEqual([0, 0, 0]);
  });

  it("centers a zero-mean sequence (no change)", () => {
    // Two frames that are already zero-mean per coefficient
    const mfccs = [
      [1, -1, 0],
      [-1, 1, 0],
    ];
    const result = applyCms(mfccs);
    expect(result[0]![0]).toBeCloseTo(1, 10);
    expect(result[0]![1]).toBeCloseTo(-1, 10);
    expect(result[0]![2]).toBeCloseTo(0, 10);
    expect(result[1]![0]).toBeCloseTo(-1, 10);
  });

  it("subtracts the per-coefficient mean across all frames", () => {
    // mean[0] = (2 + 4) / 2 = 3
    const mfccs = [[2], [4]];
    const result = applyCms(mfccs);
    expect(result[0]![0]).toBeCloseTo(-1, 10); // 2 - 3 = -1
    expect(result[1]![0]).toBeCloseTo(1, 10);  // 4 - 3 = +1
  });

  it("handles a jagged frame shorter than the first (frame[c] falls back to 0)", () => {
    // numCoeffs = 2 (from first frame); second frame has only 1 element.
    // The means accumulation loop iterates c=0..1; at c=1 frame[c] is undefined → ?? 0 fires.
    const mfccs: number[][] = [[4, 6], [2]]; // second frame shorter
    const result = applyCms(mfccs);
    // means[0] = (4+2)/2 = 3; means[1] = (6+0)/2 = 3
    expect(result[0]![0]).toBeCloseTo(4 - 3, 10);
    expect(result[0]![1]).toBeCloseTo(6 - 3, 10);
    expect(result[1]![0]).toBeCloseTo(2 - 3, 10);
  });

  it("handles a jagged frame longer than the first (means[c] falls back to 0 for extra coeff)", () => {
    // numCoeffs = 2 (from first frame); second frame has 3 elements.
    // frame.map on the second frame reaches c=2, where means[2] is undefined → ?? 0 fires.
    const mfccs: number[][] = [[1, 2], [3, 4, 5]];
    const result = applyCms(mfccs);
    // means[0] = (1+3)/2 = 2; means[1] = (2+4)/2 = 3
    // second frame output: [3-2=1, 4-3=1, 5-0=5]
    expect(result[1]![2]).toBeCloseTo(5, 10);
  });

  it("works on 13-coefficient MFCC frames", () => {
    const constant = 5;
    const mfccs = makeMfccs(10, NUM_MFCC_COEFFICIENTS, constant);
    const result = applyCms(mfccs);
    // All values identical → mean = 5 → after sub = 0
    for (const frame of result) {
      for (const v of frame) {
        expect(v).toBeCloseTo(0, 10);
      }
    }
  });
});

// ---------------------------------------------------------------------------
// MfccDtwEngine.extractFeatures
// ---------------------------------------------------------------------------

describe("MfccDtwEngine.extractFeatures", () => {
  const engine = new MfccDtwEngine();
  const SR = 16_000;

  it("returns an AudioFeatures with kind=mfcc-v1", () => {
    const audio = new Float32Array(Math.round(0.025 * SR)).fill(0.1);
    const feats = engine.extractFeatures(audio, SR);
    expect(feats.kind).toBe("mfcc-v1");
  });

  it("sets sampleRate and durationSec correctly", () => {
    const samples = 8000;
    const audio = new Float32Array(samples).fill(0.05);
    const feats = engine.extractFeatures(audio, SR);
    expect(feats.sampleRate).toBe(SR);
    expect(feats.durationSec).toBeCloseTo(samples / SR, 6);
  });

  it("returns a 2D array in data (frames × coefficients)", () => {
    const audio = new Float32Array(Math.round(0.025 * SR)).fill(0.1);
    const feats = engine.extractFeatures(audio, SR);
    const data = feats.data as number[][];
    expect(Array.isArray(data)).toBe(true);
    expect(data.length).toBe(1);
    expect(data[0]!.length).toBe(NUM_MFCC_COEFFICIENTS);
  });

  it("returns empty data for empty audio", () => {
    const feats = engine.extractFeatures(new Float32Array(0), SR);
    expect((feats.data as number[][]).length).toBe(0);
  });
});

// ---------------------------------------------------------------------------
// MfccDtwEngine.compare
// ---------------------------------------------------------------------------

describe("MfccDtwEngine.compare", () => {
  const engine = new MfccDtwEngine();

  it("returns score=0 and isMatch=false for empty candidate", () => {
    const empty = makeFeatures([]);
    const template = makeFeatures(makeMfccs(10, NUM_MFCC_COEFFICIENTS, 0.5));
    const result = engine.compare(empty, template, 0.5);
    expect(result.score).toBe(0);
    expect(result.isMatch).toBe(false);
  });

  it("returns score=0 and isMatch=false for empty template", () => {
    const candidate = makeFeatures(makeMfccs(10, NUM_MFCC_COEFFICIENTS, 0.5));
    const empty = makeFeatures([]);
    const result = engine.compare(candidate, empty, 0.5);
    expect(result.score).toBe(0);
    expect(result.isMatch).toBe(false);
  });

  it("returns score=1 for identical non-empty features", () => {
    const mfccs = makeMfccs(10, NUM_MFCC_COEFFICIENTS, 0.5);
    const f = makeFeatures(mfccs);
    const result = engine.compare(f, f, 0.9);
    expect(result.score).toBeCloseTo(1, 5);
    expect(result.isMatch).toBe(true);
  });

  it("isMatch=true when score >= threshold", () => {
    const mfccs = makeMfccs(5, NUM_MFCC_COEFFICIENTS, 0);
    const f = makeFeatures(mfccs);
    const result = engine.compare(f, f, 0.99);
    expect(result.score).toBeGreaterThanOrEqual(0.99);
    expect(result.isMatch).toBe(true);
  });

  it("isMatch=false when score < threshold", () => {
    // Anti-correlated sequences: after CMS the patterns remain distinct.
    // candidate frames alternate +1, template frames alternate -1 → non-zero DTW distance.
    const candidate = makeFeatures(makeMfccs(10, NUM_MFCC_COEFFICIENTS, (f) => f % 2 === 0 ? 1 : -1));
    const template = makeFeatures(makeMfccs(10, NUM_MFCC_COEFFICIENTS, (f) => f % 2 === 0 ? -1 : 1));
    const result = engine.compare(candidate, template, 0.99);
    expect(result.isMatch).toBe(false);
    expect(result.score).toBeGreaterThanOrEqual(0);
    expect(result.score).toBeLessThan(0.99);
  });
});

// ---------------------------------------------------------------------------
// MfccDtwEngine.compareBest
// ---------------------------------------------------------------------------

describe("MfccDtwEngine.compareBest", () => {
  const engine = new MfccDtwEngine();

  it("returns score=0 isMatch=false for empty templates array", () => {
    const candidate = makeFeatures(makeMfccs(5, NUM_MFCC_COEFFICIENTS, 0.5));
    const result = engine.compareBest(candidate, [], 0.5);
    expect(result.score).toBe(0);
    expect(result.isMatch).toBe(false);
  });

  it("returns the best (highest) score across multiple templates", () => {
    const candidate = makeFeatures(makeMfccs(5, NUM_MFCC_COEFFICIENTS, 0));

    // A matching template (identical → score ≈ 1)
    const goodTemplate = makeFeatures(makeMfccs(5, NUM_MFCC_COEFFICIENTS, 0));
    // A bad template (very different → low score)
    const badTemplate = makeFeatures(makeMfccs(5, NUM_MFCC_COEFFICIENTS, 1000));

    const result = engine.compareBest(candidate, [badTemplate, goodTemplate], 0.9);
    expect(result.score).toBeGreaterThan(0.9);
    expect(result.isMatch).toBe(true);
  });

  it("isMatch=false when all templates produce a score below threshold", () => {
    // Anti-correlated sequences survive CMS normalization (constant sequences collapse to zero).
    const candidate = makeFeatures(makeMfccs(10, NUM_MFCC_COEFFICIENTS, (f) => f % 2 === 0 ? 1 : -1));
    const bad1 = makeFeatures(makeMfccs(10, NUM_MFCC_COEFFICIENTS, (f) => f % 2 === 0 ? -1 : 1));
    const bad2 = makeFeatures(makeMfccs(10, NUM_MFCC_COEFFICIENTS, (f) => f % 2 === 0 ? -2 : 2));
    const result = engine.compareBest(candidate, [bad1, bad2], 0.99);
    expect(result.isMatch).toBe(false);
  });

  it("returns a single-template result identical to compare", () => {
    const mfccs = makeMfccs(5, NUM_MFCC_COEFFICIENTS, 0.3);
    const candidate = makeFeatures(mfccs);
    const template = makeFeatures(makeMfccs(5, NUM_MFCC_COEFFICIENTS, 0.3));
    const single = engine.compare(candidate, template, 0.5);
    const best = engine.compareBest(candidate, [template], 0.5);
    expect(best.score).toBeCloseTo(single.score, 5);
    expect(best.isMatch).toBe(single.isMatch);
  });
});

// ---------------------------------------------------------------------------
// createWakeWordEngine
// ---------------------------------------------------------------------------

describe("createWakeWordEngine", () => {
  it("returns an object with extractFeatures, compare, compareBest", () => {
    const engine = createWakeWordEngine();
    expect(typeof engine.extractFeatures).toBe("function");
    expect(typeof engine.compare).toBe("function");
    expect(typeof engine.compareBest).toBe("function");
  });

  it("is an instance of MfccDtwEngine", () => {
    expect(createWakeWordEngine()).toBeInstanceOf(MfccDtwEngine);
  });
});
