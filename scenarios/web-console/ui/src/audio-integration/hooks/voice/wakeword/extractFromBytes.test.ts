// Verifies the shared decode→extract helper that the wake-word feature uses on
// both the record path and the load path. Persistence is RAW audio; features
// are re-derived from it, so this is the seam that guarantees load-time
// features match record-time features (round-trip identity).
//
// jsdom has no Web Audio, so AudioContext is stubbed to return a synthetic PCM
// signal. The real MFCC+DTW engine then runs on it — proving the decode wiring
// (16 kHz mono → extractFeatures) and that identical audio self-matches.

import { afterEach, describe, expect, it, vi } from "vitest";

import { bytesToFeatures, MFCC_SAMPLE_RATE } from "./extractFromBytes";
import { createWakeWordEngine } from "./engine";

function sine(lengthSamples: number, freqHz: number, sampleRate: number): Float32Array {
  const out = new Float32Array(lengthSamples);
  for (let i = 0; i < lengthSamples; i++) {
    out[i] = Math.sin((2 * Math.PI * freqHz * i) / sampleRate) * 0.5;
  }
  return out;
}

const SIGNAL = sine(MFCC_SAMPLE_RATE, 440, MFCC_SAMPLE_RATE); // 1s @ 16 kHz

class FakeAudioContext {
  sampleRate: number;
  constructor(opts?: { sampleRate?: number }) {
    this.sampleRate = opts?.sampleRate ?? 44100;
  }
  decodeAudioData(_buf: ArrayBuffer): Promise<{ getChannelData: () => Float32Array }> {
    return Promise.resolve({ getChannelData: () => SIGNAL });
  }
  close(): Promise<void> {
    return Promise.resolve();
  }
}

describe("bytesToFeatures", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("decodes bytes to PCM and extracts mfcc-v1 features at 16 kHz", async () => {
    vi.stubGlobal("AudioContext", FakeAudioContext);
    const engine = createWakeWordEngine();
    const features = await bytesToFeatures(new Uint8Array([1, 2, 3, 4]), engine);

    expect(features.kind).toBe("mfcc-v1");
    expect(features.sampleRate).toBe(MFCC_SAMPLE_RATE);
    expect(Array.isArray(features.data)).toBe(true);
    expect(features.data.length).toBeGreaterThan(0);
  });

  it("yields features that self-match (round-trip identity for re-extraction on load)", async () => {
    vi.stubGlobal("AudioContext", FakeAudioContext);
    const engine = createWakeWordEngine();
    // Two independent decodes of the "same" audio — mirrors record-time vs
    // load-time extraction. They must compare as a confident match.
    const recorded = await bytesToFeatures(new Uint8Array([1, 2, 3, 4]), engine);
    const reloaded = await bytesToFeatures(new Uint8Array([1, 2, 3, 4]), engine);

    const result = engine.compareBest(reloaded, [recorded], 0.65);
    expect(result.isMatch).toBe(true);
    expect(result.score).toBeGreaterThan(0.65);
  });
});
