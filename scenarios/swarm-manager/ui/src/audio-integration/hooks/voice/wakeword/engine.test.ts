// Unit tests for the MFCC+DTW engine: CMVN normalization, c0/energy invariance,
// enrollment self-calibration, endpoint silence trimming, and a deterministic
// SEPARATION harness (synthetic time-varying "words" stand in for speech) that
// asserts same-word pairs score high and cross-word pairs score low WITHOUT a
// microphone. The recorded-fixture harness (real audio) lives separately and is
// the live-validation gate; this is the CI gate.

import { describe, expect, it } from "vitest";

import { MfccDtwEngine, normalizeFeatures } from "./engine";
import type { AudioFeatures } from "./types";

const SR = 16000;

/** mulberry32 — small deterministic PRNG so tests never use Math.random. */
function rng(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a |= 0;
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/**
 * Synthesize a "word": a sequence of segments, each a sum of (formant) sines, so
 * the spectral envelope changes over TIME (stationary tones are killed by CMVN
 * and would all look alike). `seed` adds small per-take amplitude/noise jitter so
 * repeated takes of the same word differ slightly, like real recordings.
 */
function synthWord(formantSegments: number[][], seed: number, amp = 1): Float32Array {
  const segDur = 0.25;
  const segLen = Math.floor(segDur * SR);
  const out = new Float32Array(segLen * formantSegments.length);
  const rand = rng(seed);
  const jitter = 0.9 + rand() * 0.2; // ±10% rate-ish amplitude wobble
  let idx = 0;
  for (const formants of formantSegments) {
    for (let i = 0; i < segLen; i++) {
      let v = 0;
      for (const f of formants) {
        v += Math.sin((2 * Math.PI * f * i) / SR);
      }
      v = (v / formants.length) * amp * jitter;
      v += (rand() - 0.5) * 0.02; // small noise
      out[idx++] = v;
    }
  }
  return out;
}

// Two distinct "words" — different formant-segment sequences.
const WORD_A = [
  [400, 1000, 2400],
  [700, 1200, 2600],
  [500, 1800, 2800],
];
const WORD_B = [
  [620, 1500, 3000],
  [350, 900, 2200],
  [820, 2000, 3400],
];

function silence(seconds: number): Float32Array {
  return new Float32Array(Math.floor(seconds * SR));
}

function concat(...parts: Float32Array[]): Float32Array {
  const total = parts.reduce((n, p) => n + p.length, 0);
  const out = new Float32Array(total);
  let off = 0;
  for (const p of parts) { out.set(p, off); off += p.length; }
  return out;
}

describe("normalizeFeatures (CMVN)", () => {
  it("produces ~zero mean and ~unit variance per coefficient", () => {
    const engine = new MfccDtwEngine();
    const feats = engine.extractFeatures(synthWord(WORD_A, 1), SR).data as number[][];
    const norm = normalizeFeatures(feats);
    const numCoeffs = norm[0]?.length ?? 0;
    const numFrames = norm.length;
    for (let c = 0; c < numCoeffs; c++) {
      let mean = 0;
      for (const f of norm) mean += f[c] ?? 0;
      mean /= numFrames;
      let varc = 0;
      for (const f of norm) varc += ((f[c] ?? 0) - mean) ** 2;
      varc /= numFrames;
      expect(Math.abs(mean)).toBeLessThan(1e-6);
      // Unit variance for any coeff that actually varies (silent coeffs floor out).
      if (varc > 1e-3) expect(varc).toBeGreaterThan(0.9);
    }
  });
});

describe("MfccDtwEngine.extractFeatures", () => {
  it("emits 13-d mfcc-v1 features and does not throw on silence", () => {
    const engine = new MfccDtwEngine();
    const feats = engine.extractFeatures(synthWord(WORD_A, 2), SR);
    expect(feats.kind).toBe("mfcc-v1");
    expect((feats.data as number[][])[0]?.length).toBe(13);
    expect(() => engine.extractFeatures(silence(1), SR)).not.toThrow();
  });
});

describe("loudness / c0 invariance", () => {
  it("scores a quiet vs loud take of the same word about the same as a self-match", () => {
    const engine = new MfccDtwEngine();
    const word = synthWord(WORD_A, 3, 1.0);
    const quiet = synthWord(WORD_A, 3, 0.4); // identical shape, lower amplitude
    const fLoud = engine.extractFeatures(word, SR);
    const fQuiet = engine.extractFeatures(quiet, SR);
    const selfScore = engine.compareBest(fLoud, [fLoud], 0.5).score;
    const crossLoudness = engine.compareBest(fQuiet, [fLoud], 0.5).score;
    expect(crossLoudness).toBeGreaterThan(0.8);
    expect(Math.abs(crossLoudness - selfScore)).toBeLessThan(0.1);
  });
});

describe("endpoint silence trimming", () => {
  it("a silence-padded clip self-matches the unpadded clip", () => {
    const engine = new MfccDtwEngine();
    const word = synthWord(WORD_A, 4);
    const unpadded = engine.extractFeatures(word, SR);
    const padded = engine.extractFeatures(concat(silence(0.5), word, silence(0.5)), SR);
    const score = engine.compareBest(padded, [unpadded], 0.5).score;
    expect(score).toBeGreaterThan(0.7);
  });

  it("an all-silence candidate does not match a real template", () => {
    const engine = new MfccDtwEngine();
    const template = engine.extractFeatures(synthWord(WORD_A, 5), SR);
    const sil = engine.extractFeatures(silence(1), SR);
    const result = engine.compareBest(sil, [template], 0.7);
    expect(result.isMatch).toBe(false);
  });
});

describe("calibrate", () => {
  it("returns null for fewer than 2 samples", () => {
    const engine = new MfccDtwEngine();
    expect(engine.calibrate([])).toBeNull();
    expect(engine.calibrate([engine.extractFeatures(synthWord(WORD_A, 6), SR)])).toBeNull();
  });

  it("returns finite mu and floored sigma for >= 2 samples", () => {
    const engine = new MfccDtwEngine();
    const samples: AudioFeatures[] = [10, 11, 12].map((s) =>
      engine.extractFeatures(synthWord(WORD_A, s), SR),
    );
    const cal = engine.calibrate(samples);
    if (cal === null) throw new Error("expected non-null calibration");
    expect(Number.isFinite(cal.mu)).toBe(true);
    expect(cal.sigma).toBeGreaterThan(0);
    expect(cal.kind).toBe("mfcc-v1");
  });
});

describe("separation harness (synthetic words)", () => {
  it("calibrated scores separate same-word from cross-word", () => {
    const engine = new MfccDtwEngine();
    // Enroll on several WORD_A takes; calibrate on them.
    const enroll: AudioFeatures[] = [20, 21, 22, 23, 24].map((s) =>
      engine.extractFeatures(synthWord(WORD_A, s), SR),
    );
    const cal = engine.calibrate(enroll);
    if (cal === null) throw new Error("expected non-null calibration");

    // Held-out same-word takes should score high.
    const sameScores = [30, 31, 32, 33, 34].map(
      (s) => engine.compareBest(engine.extractFeatures(synthWord(WORD_A, s), SR), enroll, 0.7, cal).score,
    );
    // Different-word takes should score low.
    const crossScores = [40, 41, 42, 43, 44].map(
      (s) => engine.compareBest(engine.extractFeatures(synthWord(WORD_B, s), SR), enroll, 0.7, cal).score,
    );

    const median = (xs: number[]): number => {
      const sorted = [...xs].sort((a, b) => a - b);
      return sorted[Math.floor(sorted.length / 2)] ?? 0;
    };
    const sameMedian = median(sameScores);
    const crossMax = Math.max(...crossScores);

    expect(sameMedian).toBeGreaterThanOrEqual(0.8);
    expect(crossMax).toBeLessThanOrEqual(0.4);
    expect(sameMedian - crossMax).toBeGreaterThanOrEqual(0.3);
  });
});
