import { describe, expect, it } from "vitest";

import { MfccDtwEngine, normalizeFeatures } from "./engine";
import type { AudioFeatures } from "./types";

function features(values: number[][]): AudioFeatures {
  return { kind: "mfcc-v1", data: values, sampleRate: 16_000, durationSec: 1 };
}

const SAMPLE_RATE = 16_000;
const WORD_A = [[400, 1000, 2400], [700, 1200, 2600], [500, 1800, 2800]];
const WORD_B = [[620, 1500, 3000], [350, 900, 2200], [820, 2000, 3400]];

function rng(seed: number): () => number {
  let value = seed >>> 0;
  return () => {
    value = (value + 0x6d2b79f5) | 0;
    let mixed = Math.imul(value ^ (value >>> 15), 1 | value);
    mixed = (mixed + Math.imul(mixed ^ (mixed >>> 7), 61 | mixed)) ^ mixed;
    return ((mixed ^ (mixed >>> 14)) >>> 0) / 4_294_967_296;
  };
}

function synthWord(formants: number[][], seed: number, amplitude = 1): Float32Array {
  const segmentLength = Math.floor(0.25 * SAMPLE_RATE);
  const output = new Float32Array(segmentLength * formants.length);
  const random = rng(seed);
  const jitter = 0.9 + random() * 0.2;
  let offset = 0;
  for (const segment of formants) {
    for (let sample = 0; sample < segmentLength; sample++) {
      const tone = segment.reduce((sum, frequency) => sum + Math.sin((2 * Math.PI * frequency * sample) / SAMPLE_RATE), 0) / segment.length;
      output[offset++] = tone * amplitude * jitter + (random() - 0.5) * 0.02;
    }
  }
  return output;
}

function silence(seconds: number): Float32Array { return new Float32Array(Math.floor(seconds * SAMPLE_RATE)); }

function concat(...parts: Float32Array[]): Float32Array {
  const output = new Float32Array(parts.reduce((total, part) => total + part.length, 0));
  let offset = 0;
  for (const part of parts) { output.set(part, offset); offset += part.length; }
  return output;
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

  it("produces zero-mean, unit-variance normalized MFCC coefficients", () => {
    const engine = new MfccDtwEngine();
    const normalized = normalizeFeatures(engine.extractFeatures(synthWord(WORD_A, 1), SAMPLE_RATE).data as number[][]);
    for (let coefficient = 0; coefficient < (normalized[0]?.length ?? 0); coefficient++) {
      const mean = normalized.reduce((sum, frame) => sum + (frame[coefficient] ?? 0), 0) / normalized.length;
      const variance = normalized.reduce((sum, frame) => sum + ((frame[coefficient] ?? 0) - mean) ** 2, 0) / normalized.length;
      expect(Math.abs(mean)).toBeLessThan(1e-6);
      if (variance > 1e-3) expect(variance).toBeGreaterThan(0.9);
    }
  });

  it("emits 13-dimensional features and handles silence", () => {
    const engine = new MfccDtwEngine();
    const result = engine.extractFeatures(synthWord(WORD_A, 2), SAMPLE_RATE);
    expect(result.kind).toBe("mfcc-v1");
    expect((result.data as number[][])[0]?.length).toBe(13);
    expect(() => engine.extractFeatures(silence(1), SAMPLE_RATE)).not.toThrow();
  });

  it("is stable across loudness changes and endpoint silence", () => {
    const engine = new MfccDtwEngine();
    const word = synthWord(WORD_A, 3);
    const loud = engine.extractFeatures(word, SAMPLE_RATE);
    const quiet = engine.extractFeatures(synthWord(WORD_A, 3, 0.4), SAMPLE_RATE);
    expect(engine.compareBest(quiet, [loud], 0.5).score).toBeGreaterThan(0.8);

    const padded = engine.extractFeatures(concat(silence(0.5), word, silence(0.5)), SAMPLE_RATE);
    expect(engine.compareBest(padded, [loud], 0.5).score).toBeGreaterThan(0.7);
    expect(engine.compareBest(engine.extractFeatures(silence(1), SAMPLE_RATE), [loud], 0.7).isMatch).toBe(false);
  });

  it("separates held-out synthetic words after enrollment calibration", () => {
    const engine = new MfccDtwEngine();
    const enrollment = [20, 21, 22, 23, 24].map((seed) => engine.extractFeatures(synthWord(WORD_A, seed), SAMPLE_RATE));
    const calibration = engine.calibrate(enrollment);
    expect(calibration).not.toBeNull();
    const same = [30, 31, 32, 33, 34].map((seed) => engine.compareBest(engine.extractFeatures(synthWord(WORD_A, seed), SAMPLE_RATE), enrollment, 0.7, calibration).score).sort((a, b) => a - b);
    const different = [40, 41, 42, 43, 44].map((seed) => engine.compareBest(engine.extractFeatures(synthWord(WORD_B, seed), SAMPLE_RATE), enrollment, 0.7, calibration).score);
    const sameMedian = same[Math.floor(same.length / 2)] ?? 0;
    const differentMax = Math.max(...different);
    expect(sameMedian).toBeGreaterThanOrEqual(0.8);
    expect(differentMax).toBeLessThanOrEqual(0.4);
    expect(sameMedian - differentMax).toBeGreaterThanOrEqual(0.3);
  });
});
