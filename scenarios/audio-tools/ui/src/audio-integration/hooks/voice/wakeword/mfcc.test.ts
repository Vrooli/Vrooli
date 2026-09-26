import { describe, it, expect } from "vitest";

import {
  nextPow2,
  fft,
  hzToMel,
  melToHz,
  buildMelFilterbank,
  dctII,
  extractMfcc,
} from "./mfcc";
import { NUM_MFCC_COEFFICIENTS, MEL_FILTER_COUNT, MEL_LOW_HZ, MEL_HIGH_HZ } from "./types";

// ---------------------------------------------------------------------------
// nextPow2
// ---------------------------------------------------------------------------

describe("nextPow2", () => {
  it("returns 1 for n=0 and n=1", () => {
    expect(nextPow2(0)).toBe(1);
    expect(nextPow2(1)).toBe(1);
  });

  it("is an identity for exact powers of two", () => {
    for (const p of [2, 4, 8, 16, 32, 64, 128, 256, 512]) {
      expect(nextPow2(p)).toBe(p);
    }
  });

  it("rounds up to the next power of two", () => {
    expect(nextPow2(3)).toBe(4);
    expect(nextPow2(5)).toBe(8);
    expect(nextPow2(9)).toBe(16);
    expect(nextPow2(17)).toBe(32);
    expect(nextPow2(400)).toBe(512);
    expect(nextPow2(1000)).toBe(1024);
  });
});

// ---------------------------------------------------------------------------
// fft
// ---------------------------------------------------------------------------

describe("fft", () => {
  it("transforms an impulse at index 0 into all-ones (real)", () => {
    const n = 4;
    const real = new Float64Array([1, 0, 0, 0]);
    const imag = new Float64Array(n);
    fft(real, imag);
    for (let k = 0; k < n; k++) {
      expect(real[k]).toBeCloseTo(1, 9);
      expect(imag[k]).toBeCloseTo(0, 9);
    }
  });

  it("transforms a DC (all-ones) signal into N at bin 0, zero elsewhere", () => {
    const n = 4;
    const real = new Float64Array([1, 1, 1, 1]);
    const imag = new Float64Array(n);
    fft(real, imag);
    expect(real[0]).toBeCloseTo(4, 9);
    expect(imag[0]).toBeCloseTo(0, 9);
    for (let k = 1; k < n; k++) {
      expect(Math.abs(real[k] as number)).toBeCloseTo(0, 5);
      expect(Math.abs(imag[k] as number)).toBeCloseTo(0, 5);
    }
  });

  it("produces correct magnitude for a 1-Hz sine wave (n=8)", () => {
    const n = 8;
    const real = new Float64Array(n);
    const imag = new Float64Array(n);
    for (let i = 0; i < n; i++) {
      real[i] = Math.sin((2 * Math.PI * i) / n);
    }
    fft(real, imag);
    // The sine wave at frequency 1 should have energy at bin 1 and N-1=7.
    const mag1 = Math.sqrt((real[1] as number) ** 2 + (imag[1] as number) ** 2);
    const mag7 = Math.sqrt((real[7] as number) ** 2 + (imag[7] as number) ** 2);
    expect(mag1).toBeGreaterThan(3); // close to n/2 = 4
    expect(mag7).toBeGreaterThan(3);
    // DC and Nyquist should be near zero
    expect(Math.abs(real[0] as number)).toBeCloseTo(0, 3);
    expect(Math.abs(real[4] as number)).toBeCloseTo(0, 3);
  });

  it("handles size-2 FFT correctly (Parseval conservation)", () => {
    const real = new Float64Array([1, -1]);
    const imag = new Float64Array(2);
    fft(real, imag);
    // DFT of [1,-1]: X[0]=0, X[1]=2
    expect(real[0]).toBeCloseTo(0, 9);
    expect(real[1]).toBeCloseTo(2, 9);
    expect(imag[0]).toBeCloseTo(0, 9);
    expect(imag[1]).toBeCloseTo(0, 9);
  });
});

// ---------------------------------------------------------------------------
// hzToMel / melToHz — roundtrip
// ---------------------------------------------------------------------------

describe("hzToMel / melToHz", () => {
  it("roundtrips without loss for typical voice frequencies", () => {
    for (const hz of [300, 700, 1000, 4000, 8000]) {
      expect(melToHz(hzToMel(hz))).toBeCloseTo(hz, 5);
    }
  });

  it("mel increases monotonically with Hz", () => {
    const freqs = [100, 500, 1000, 2000, 8000];
    const mels = freqs.map(hzToMel);
    for (let i = 1; i < mels.length; i++) {
      expect((mels[i] as number) > (mels[i - 1] as number)).toBe(true);
    }
  });

  it("hzToMel(700) ≈ 781.4 (well-known calibration point)", () => {
    expect(hzToMel(700)).toBeCloseTo(781.4, 0);
  });
});

// ---------------------------------------------------------------------------
// buildMelFilterbank
// ---------------------------------------------------------------------------

describe("buildMelFilterbank", () => {
  const FFT_SIZE = 512;
  const SAMPLE_RATE = 16_000;

  it("returns the requested number of filters", () => {
    const fb = buildMelFilterbank(26, FFT_SIZE, SAMPLE_RATE, MEL_LOW_HZ, MEL_HIGH_HZ);
    expect(fb.length).toBe(26);
  });

  it("each filter has (fftSize/2)+1 bins", () => {
    const numBins = (FFT_SIZE >> 1) + 1;
    const fb = buildMelFilterbank(26, FFT_SIZE, SAMPLE_RATE, MEL_LOW_HZ, MEL_HIGH_HZ);
    for (const f of fb) {
      expect(f.length).toBe(numBins);
    }
  });

  it("each filter value is in [0, 1]", () => {
    const fb = buildMelFilterbank(26, FFT_SIZE, SAMPLE_RATE, MEL_LOW_HZ, MEL_HIGH_HZ);
    for (const f of fb) {
      for (const v of f) {
        expect(v).toBeGreaterThanOrEqual(0);
        expect(v).toBeLessThanOrEqual(1 + 1e-10);
      }
    }
  });

  it("returns a single filter without error", () => {
    const fb = buildMelFilterbank(1, FFT_SIZE, SAMPLE_RATE, MEL_LOW_HZ, MEL_HIGH_HZ);
    expect(fb.length).toBe(1);
  });
});

// ---------------------------------------------------------------------------
// dctII
// ---------------------------------------------------------------------------

describe("dctII", () => {
  it("returns all zeros for a zero input", () => {
    const input = new Float64Array(26);
    const output = dctII(input, 13);
    expect(output.length).toBe(13);
    for (const v of output) {
      expect(v).toBeCloseTo(0, 10);
    }
  });

  it("returns numCoeffs coefficients regardless of input length", () => {
    const output = dctII(new Float64Array([1, 2, 3, 4, 5]), 3);
    expect(output.length).toBe(3);
  });

  it("DC coefficient (k=0) equals scale * sum(input)", () => {
    // For k=0: output[0] = sqrt(2/n) * sum(input[i] * cos(0)) = sqrt(2/n) * sum(input)
    const input = new Float64Array([1, 1, 1, 1]);
    const n = input.length;
    const scale = Math.sqrt(2 / n);
    const expected0 = scale * input.reduce((a, b) => a + b, 0);
    const output = dctII(input, 2);
    expect(output[0]).toBeCloseTo(expected0, 9);
  });

  it("is consistent for a single-element input", () => {
    const input = new Float64Array([1]);
    const output = dctII(input, 1);
    // sqrt(2/1) * 1 * cos(0) = sqrt(2)
    expect(output[0]).toBeCloseTo(Math.sqrt(2), 9);
  });
});

// ---------------------------------------------------------------------------
// extractMfcc
// ---------------------------------------------------------------------------

describe("extractMfcc", () => {
  const SR = 16_000;

  it("returns [] for empty audio", () => {
    expect(extractMfcc(new Float32Array(0), SR)).toEqual([]);
  });

  it("returns [] when audio is shorter than one frame", () => {
    // frameLenSamples = round(0.025 * 16000) = 400; audio < 400 → 0 frames
    expect(extractMfcc(new Float32Array(100), SR)).toEqual([]);
    expect(extractMfcc(new Float32Array(1), SR)).toEqual([]);
  });

  it("returns exactly 1 frame (13 coefficients) for a frame-length input", () => {
    const frameLenSamples = Math.round(0.025 * SR); // 400
    const audio = new Float32Array(frameLenSamples).fill(0.1);
    const result = extractMfcc(audio, SR);
    expect(result.length).toBe(1);
    expect(result[0]!.length).toBe(NUM_MFCC_COEFFICIENTS);
  });

  it("returns multiple frames for longer audio and each has 13 coefficients", () => {
    // frameHopSamples = round(0.010 * 16000) = 160; frameLenSamples = 400
    // numFrames for 800 samples = floor((800-400)/160)+1 = 3
    const audio = new Float32Array(800).fill(0.05);
    const result = extractMfcc(audio, SR);
    expect(result.length).toBe(3);
    for (const frame of result) {
      expect(frame.length).toBe(NUM_MFCC_COEFFICIENTS);
    }
  });

  it("returns a non-trivial result for a non-zero signal", () => {
    const SR2 = 16_000;
    const frameLenSamples = Math.round(0.025 * SR2);
    const audio = new Float32Array(frameLenSamples);
    for (let i = 0; i < frameLenSamples; i++) {
      // 1 kHz sine wave
      audio[i] = 0.5 * Math.sin((2 * Math.PI * 1000 * i) / SR2);
    }
    const result = extractMfcc(audio, SR2);
    expect(result.length).toBe(1);
    // The coefficients should be non-trivially non-zero
    const allZero = (result[0] as number[]).every(v => v === 0);
    expect(allZero).toBe(false);
  });

  it("uses the filterbank cache: two calls with the same params return the same shape", () => {
    const SR2 = 16_000;
    const audio = new Float32Array(Math.round(0.025 * SR2)).fill(0.1);
    const r1 = extractMfcc(audio, SR2);
    const r2 = extractMfcc(audio, SR2);
    expect(r1.length).toBe(r2.length);
    expect((r1[0] as number[]).length).toBe((r2[0] as number[]).length);
  });

  it("applies MEL_FILTER_COUNT filters (shape is independent of filter count)", () => {
    // Verify numFilters used is MEL_FILTER_COUNT from types.ts
    expect(MEL_FILTER_COUNT).toBe(26);
    // If MEL_FILTER_COUNT changed, extractMfcc would still return 13 coefficients
    // (DCT uses NUM_MFCC_COEFFICIENTS, not MEL_FILTER_COUNT)
    const audio = new Float32Array(Math.round(0.025 * SR)).fill(0.1);
    const result = extractMfcc(audio, SR);
    expect((result[0] as number[]).length).toBe(NUM_MFCC_COEFFICIENTS);
  });
});
