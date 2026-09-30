import { afterEach, describe, expect, it, vi } from "vitest";

import { bytesToFeatures, MFCC_SAMPLE_RATE } from "./extractFromBytes";
import { createWakeWordEngine } from "./engine";

function sine(lengthSamples: number, freqHz: number, sampleRate: number): Float32Array {
  const out = new Float32Array(lengthSamples);
  for (let i = 0; i < lengthSamples; i++) out[i] = Math.sin((2 * Math.PI * freqHz * i) / sampleRate) * 0.5;
  return out;
}

const SIGNAL = sine(MFCC_SAMPLE_RATE, 440, MFCC_SAMPLE_RATE);

class FakeAudioContext {
  constructor(public sampleRate = 44_100) {}
  decodeAudioData(_buffer: ArrayBuffer): Promise<{ getChannelData: () => Float32Array }> {
    return Promise.resolve({ getChannelData: () => SIGNAL });
  }
  close(): Promise<void> { return Promise.resolve(); }
}

describe("bytesToFeatures", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("decodes bytes and extracts mfcc-v1 features at 16 kHz", async () => {
    vi.stubGlobal("AudioContext", FakeAudioContext);
    const features = await bytesToFeatures(new Uint8Array([1, 2, 3, 4]), createWakeWordEngine());
    expect(features.kind).toBe("mfcc-v1");
    expect(features.sampleRate).toBe(MFCC_SAMPLE_RATE);
    expect(features.data.length).toBeGreaterThan(0);
  });

  it("re-derives identical features that self-match after persistence", async () => {
    vi.stubGlobal("AudioContext", FakeAudioContext);
    const engine = createWakeWordEngine();
    const recorded = await bytesToFeatures(new Uint8Array([1, 2, 3, 4]), engine);
    const reloaded = await bytesToFeatures(new Uint8Array([1, 2, 3, 4]), engine);
    const result = engine.compareBest(reloaded, [recorded], 0.65);
    expect(result.isMatch).toBe(true);
    expect(result.score).toBeGreaterThan(0.65);
  });
});
