import { describe, it, expect } from "vitest";
import { extractMfcc, fft, nextPow2, hzToMel, melToHz, buildMelFilterbank, dctII } from "../wakeword/mfcc";
import { NUM_MFCC_COEFFICIENTS, FRAME_HOP_MS, FRAME_LENGTH_MS } from "../wakeword/types";

describe("nextPow2", () => {
  it("returns 1 for 1", () => expect(nextPow2(1)).toBe(1));
  it("returns power of 2 for exact match", () => expect(nextPow2(256)).toBe(256));
  it("rounds up to next power of 2", () => expect(nextPow2(400)).toBe(512));
  it("handles small values", () => expect(nextPow2(3)).toBe(4));
});

describe("fft", () => {
  it("produces correct output for known signal", () => {
    // DC signal: all 1s → DFT bin 0 should equal N, rest 0
    const n = 8;
    const real = new Float64Array(n).fill(1);
    const imag = new Float64Array(n).fill(0);
    fft(real, imag);
    expect(real[0]).toBeCloseTo(n, 5);
    for (let k = 1; k < n; k++) {
      expect(real[k]).toBeCloseTo(0, 5);
      expect(imag[k]).toBeCloseTo(0, 5);
    }
  });

  it("produces correct magnitude for a pure sine", () => {
    const n = 64;
    const real = new Float64Array(n);
    const imag = new Float64Array(n).fill(0);
    // Sine at bin frequency k=3
    for (let i = 0; i < n; i++) {
      real[i] = Math.sin((2 * Math.PI * 3 * i) / n);
    }
    fft(real, imag);
    // Magnitude at bin 3 and n-3 should be ~n/2
    const real3 = real[3] ?? 0;
    const imag3 = imag[3] ?? 0;
    const mag3 = Math.sqrt(real3 ** 2 + imag3 ** 2);
    expect(mag3).toBeCloseTo(n / 2, 1);
  });
});

describe("hzToMel / melToHz", () => {
  it("round-trips correctly", () => {
    for (const hz of [0, 300, 1000, 4000, 8000]) {
      expect(melToHz(hzToMel(hz))).toBeCloseTo(hz, 3);
    }
  });

  it("mel(1000) is approximately 1000 (O'Shaughnessy formula)", () => {
    // 2595 * log10(1 + 1000/700) = 2595 * log10(2.4286) ≈ 999.99
    expect(hzToMel(1000)).toBeCloseTo(1000, 0);
  });
});

describe("buildMelFilterbank", () => {
  it("returns correct number of filters", () => {
    const fb = buildMelFilterbank(26, 512, 16000, 300, 8000);
    expect(fb.length).toBe(26);
  });

  it("each filter has correct number of bins", () => {
    const fftSize = 512;
    const fb = buildMelFilterbank(26, fftSize, 16000, 300, 8000);
    const numBins = (fftSize >> 1) + 1;
    for (const filter of fb) {
      expect(filter.length).toBe(numBins);
    }
  });

  it("filters have non-negative values", () => {
    const fb = buildMelFilterbank(26, 512, 16000, 300, 8000);
    for (const filter of fb) {
      for (const v of filter) {
        expect(v).toBeGreaterThanOrEqual(0);
      }
    }
  });
});

describe("dctII", () => {
  it("returns correct number of coefficients", () => {
    const input = new Float64Array(26).fill(1);
    const result = dctII(input, 13);
    expect(result.length).toBe(13);
  });

  it("first coefficient captures energy", () => {
    // Uniform input: first DCT coefficient should be large, rest ~0
    const n = 8;
    const input = new Float64Array(n).fill(1);
    const result = dctII(input, 4);
    expect(Math.abs(result[0] ?? 0)).toBeGreaterThan(0);
  });
});

describe("extractMfcc", () => {
  const sampleRate = 16000;
  const durationSec = 1.0;
  const numSamples = sampleRate * durationSec;

  it("returns correct frame dimensions for 1 second of audio", () => {
    const audio = new Float32Array(numSamples);
    // White noise
    for (let i = 0; i < numSamples; i++) audio[i] = (Math.random() - 0.5) * 0.5;

    const mfccs = extractMfcc(audio, sampleRate);
    const frameLenSamples = Math.round((FRAME_LENGTH_MS / 1000) * sampleRate);
    const frameHopSamples = Math.round((FRAME_HOP_MS / 1000) * sampleRate);
    const expectedFrames = Math.floor((numSamples - frameLenSamples) / frameHopSamples) + 1;

    expect(mfccs.length).toBe(expectedFrames);
    for (const frame of mfccs) {
      expect(frame.length).toBe(NUM_MFCC_COEFFICIENTS);
    }
  });

  it("returns correct shape for 2 seconds at 16kHz", () => {
    const samples = sampleRate * 2;
    const audio = new Float32Array(samples);
    for (let i = 0; i < samples; i++) audio[i] = Math.sin(2 * Math.PI * 440 * i / sampleRate) * 0.3;

    const mfccs = extractMfcc(audio, sampleRate);
    expect(mfccs.length).toBeGreaterThan(150); // ~198 frames for 2s
    expect(mfccs[0]?.length).toBe(NUM_MFCC_COEFFICIENTS);
  });

  it("produces deterministic output for identical input", () => {
    const audio = new Float32Array(numSamples);
    for (let i = 0; i < numSamples; i++) audio[i] = Math.sin(2 * Math.PI * 440 * i / sampleRate);

    const result1 = extractMfcc(audio, sampleRate);
    const result2 = extractMfcc(audio, sampleRate);

    expect(result1.length).toBe(result2.length);
    for (let f = 0; f < result1.length; f++) {
      const row1 = result1[f];
      const row2 = result2[f];
      if (!row1 || !row2) throw new Error("missing frame");
      for (let c = 0; c < NUM_MFCC_COEFFICIENTS; c++) {
        expect(row1[c] ?? 0).toBeCloseTo(row2[c] ?? 0, 10);
      }
    }
  });

  it("returns empty array for empty input", () => {
    expect(extractMfcc(new Float32Array(0), sampleRate)).toEqual([]);
  });

  it("returns empty array for very short audio (< 1 frame)", () => {
    // Less than one frame length
    const tooShort = new Float32Array(100); // 100 samples at 16kHz = 6.25ms < 25ms frame
    expect(extractMfcc(tooShort, sampleRate)).toEqual([]);
  });

  it("produces different features for different signals", () => {
    const sine440 = new Float32Array(numSamples);
    const sine2000 = new Float32Array(numSamples);
    for (let i = 0; i < numSamples; i++) {
      sine440[i] = Math.sin(2 * Math.PI * 440 * i / sampleRate) * 0.3;
      sine2000[i] = Math.sin(2 * Math.PI * 2000 * i / sampleRate) * 0.3;
    }

    const mfcc1 = extractMfcc(sine440, sampleRate);
    const mfcc2 = extractMfcc(sine2000, sampleRate);

    // At least some coefficients should differ significantly
    let totalDiff = 0;
    const minLen = Math.min(mfcc1.length, mfcc2.length);
    for (let f = 0; f < minLen; f++) {
      const row1 = mfcc1[f];
      const row2 = mfcc2[f];
      if (!row1 || !row2) continue;
      for (let c = 0; c < NUM_MFCC_COEFFICIENTS; c++) {
        totalDiff += Math.abs((row1[c] ?? 0) - (row2[c] ?? 0));
      }
    }
    expect(totalDiff).toBeGreaterThan(0);
  });

  it("handles silence (all zeros) without errors", () => {
    const silence = new Float32Array(numSamples); // all zeros
    const mfccs = extractMfcc(silence, sampleRate);
    expect(mfccs.length).toBeGreaterThan(0);
    // All values should be finite
    for (const frame of mfccs) {
      for (const c of frame) {
        expect(isFinite(c)).toBe(true);
      }
    }
  });

  it("handles loud audio (clipping range) without errors", () => {
    const loud = new Float32Array(numSamples);
    for (let i = 0; i < numSamples; i++) loud[i] = Math.sin(2 * Math.PI * 440 * i / sampleRate);
    const mfccs = extractMfcc(loud, sampleRate);
    expect(mfccs.length).toBeGreaterThan(0);
    for (const frame of mfccs) {
      for (const c of frame) {
        expect(isFinite(c)).toBe(true);
      }
    }
  });
});
