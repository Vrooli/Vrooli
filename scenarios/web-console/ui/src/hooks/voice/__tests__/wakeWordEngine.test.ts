import { describe, it, expect } from "vitest";
import { MfccDtwEngine, createWakeWordEngine, applyCms } from "../wakeword/engine";
import { DEFAULT_WAKE_WORD_THRESHOLD } from "../wakeword/types";

const sampleRate = 16000;

/** Generate a complex synthetic signal (sum of harmonics, more speech-like). */
function generateComplex(
  fundamentalHz: number,
  durationSec: number,
  rate: number = sampleRate,
): Float32Array {
  const numSamples = Math.round(rate * durationSec);
  const audio = new Float32Array(numSamples);
  for (let i = 0; i < numSamples; i++) {
    const t = i / rate;
    // Fundamental + 3 harmonics with decreasing amplitude (mimics vowel formants)
    audio[i] =
      0.4 * Math.sin(2 * Math.PI * fundamentalHz * t) +
      0.25 * Math.sin(2 * Math.PI * fundamentalHz * 2 * t) +
      0.15 * Math.sin(2 * Math.PI * fundamentalHz * 3 * t) +
      0.1 * Math.sin(2 * Math.PI * fundamentalHz * 5 * t);
  }
  return audio;
}

/** Add slight amplitude variation to simulate natural speech dynamics. */
function addAmplitudeVariation(audio: Float32Array, seed: number = 1): Float32Array {
  const result = new Float32Array(audio.length);
  for (let i = 0; i < audio.length; i++) {
    // Slow amplitude envelope variation
    const env = 1 + 0.1 * Math.sin(2 * Math.PI * 3 * i / sampleRate + seed);
    result[i] = audio[i] * env;
  }
  return result;
}

/** Generate deterministic noise. */
function generateNoise(durationSec: number, amplitude: number = 0.3): Float32Array {
  const numSamples = Math.round(sampleRate * durationSec);
  const audio = new Float32Array(numSamples);
  let seed = 42;
  for (let i = 0; i < numSamples; i++) {
    seed = (seed * 1103515245 + 12345) & 0x7fffffff;
    audio[i] = ((seed / 0x7fffffff) * 2 - 1) * amplitude;
  }
  return audio;
}

describe("applyCms", () => {
  it("returns empty for empty input", () => {
    expect(applyCms([])).toEqual([]);
  });

  it("produces zero-mean coefficients", () => {
    const input = [
      [1, 2, 3],
      [3, 4, 5],
      [5, 6, 7],
    ];
    const result = applyCms(input);
    // Mean of each column should be ~0
    for (let c = 0; c < 3; c++) {
      const mean = result.reduce((sum, row) => sum + row[c], 0) / result.length;
      expect(mean).toBeCloseTo(0, 10);
    }
  });
});

describe("MfccDtwEngine", () => {
  const engine = new MfccDtwEngine();

  describe("extractFeatures", () => {
    it("produces mfcc-v1 features", () => {
      const audio = generateComplex(200, 1.0);
      const features = engine.extractFeatures(audio, sampleRate);
      expect(features.kind).toBe("mfcc-v1");
      expect(features.sampleRate).toBe(sampleRate);
      expect(features.durationSec).toBeCloseTo(1.0, 1);
    });

    it("data is 2D array of frames x 13 coefficients", () => {
      const audio = generateComplex(200, 1.0);
      const features = engine.extractFeatures(audio, sampleRate);
      const data = features.data as number[][];
      expect(data.length).toBeGreaterThan(0);
      expect(data[0].length).toBe(13);
    });

    it("produces empty data for empty audio", () => {
      const features = engine.extractFeatures(new Float32Array(0), sampleRate);
      expect((features.data as number[][]).length).toBe(0);
    });
  });

  describe("compare", () => {
    it("identical audio produces score of 1.0", () => {
      const audio = generateComplex(200, 1.0);
      const features = engine.extractFeatures(audio, sampleRate);
      const result = engine.compare(features, features, DEFAULT_WAKE_WORD_THRESHOLD);
      expect(result.score).toBeCloseTo(1.0, 3);
      expect(result.isMatch).toBe(true);
    });

    it("same signal with slight amplitude variation still matches", () => {
      const base = generateComplex(200, 1.0);
      const varied = addAmplitudeVariation(base, 1);
      const f1 = engine.extractFeatures(base, sampleRate);
      const f2 = engine.extractFeatures(varied, sampleRate);
      const result = engine.compare(f1, f2, DEFAULT_WAKE_WORD_THRESHOLD);
      // CMS + DTW should handle amplitude variations well
      expect(result.score).toBeGreaterThan(0.5);
    });

    it("completely different signals produce low score", () => {
      const complex = engine.extractFeatures(generateComplex(200, 1.0), sampleRate);
      const noise = engine.extractFeatures(generateNoise(1.0), sampleRate);
      const result = engine.compare(complex, noise, DEFAULT_WAKE_WORD_THRESHOLD);
      expect(result.score).toBeLessThan(DEFAULT_WAKE_WORD_THRESHOLD);
      expect(result.isMatch).toBe(false);
    });

    it("different fundamental frequencies produce lower score than identical", () => {
      const f200 = engine.extractFeatures(generateComplex(200, 1.0), sampleRate);
      const f500 = engine.extractFeatures(generateComplex(500, 1.0), sampleRate);
      const scoreSame = engine.compare(f200, f200, 0.1).score;
      const scoreDiff = engine.compare(f200, f500, 0.1).score;
      expect(scoreSame).toBeGreaterThan(scoreDiff);
    });

    it("returns score 0 for empty features", () => {
      const empty = engine.extractFeatures(new Float32Array(0), sampleRate);
      const real = engine.extractFeatures(generateComplex(200, 1.0), sampleRate);
      expect(engine.compare(empty, real, 0.5).score).toBe(0);
      expect(engine.compare(real, empty, 0.5).score).toBe(0);
    });
  });

  describe("compareBest", () => {
    it("returns the best match across multiple templates", () => {
      const audio = generateComplex(200, 1.0);
      const candidate = engine.extractFeatures(audio, sampleRate);
      const match = engine.extractFeatures(audio, sampleRate); // identical
      const noMatch = engine.extractFeatures(generateNoise(1.0), sampleRate);

      const result = engine.compareBest(candidate, [noMatch, match, noMatch], DEFAULT_WAKE_WORD_THRESHOLD);
      expect(result.score).toBeCloseTo(1.0, 3);
      expect(result.isMatch).toBe(true);
    });

    it("returns no match when all templates are different", () => {
      const candidate = engine.extractFeatures(generateComplex(200, 1.0), sampleRate);
      const t1 = engine.extractFeatures(generateNoise(1.0, 0.3), sampleRate);
      const t2 = engine.extractFeatures(generateNoise(1.0, 0.6), sampleRate);

      const result = engine.compareBest(candidate, [t1, t2], DEFAULT_WAKE_WORD_THRESHOLD);
      expect(result.isMatch).toBe(false);
    });

    it("returns score 0 for empty templates array", () => {
      const candidate = engine.extractFeatures(generateComplex(200, 1.0), sampleRate);
      const result = engine.compareBest(candidate, [], DEFAULT_WAKE_WORD_THRESHOLD);
      expect(result.score).toBe(0);
      expect(result.isMatch).toBe(false);
    });

    it("respects threshold parameter", () => {
      const audio = generateComplex(200, 1.0);
      const f = engine.extractFeatures(audio, sampleRate);
      // Identical: score ~1.0
      const highThreshold = engine.compareBest(f, [f], 0.99);
      const lowThreshold = engine.compareBest(f, [f], 0.01);
      expect(highThreshold.isMatch).toBe(true);
      expect(lowThreshold.isMatch).toBe(true);
    });
  });
});

describe("createWakeWordEngine", () => {
  it("returns a valid WakeWordEngine instance", () => {
    const engine = createWakeWordEngine();
    expect(engine.extractFeatures).toBeDefined();
    expect(engine.compare).toBeDefined();
    expect(engine.compareBest).toBeDefined();
  });
});
