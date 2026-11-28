import { describe, expect, it } from "vitest";
import { StreamingPcm16Resampler } from "./pcm";

describe("StreamingPcm16Resampler", () => {
  const legacyProcess = (state: { captureRate: number; ratio: number; pending: Float32Array; position: number }, frame: Float32Array, captureRate: number) => {
    if (captureRate === 16_000) {
      state.captureRate = 16_000;
      state.ratio = 1;
      state.pending = new Float32Array(0);
      state.position = 0;
      return convertToPcm(frame);
    }
    if (captureRate !== state.captureRate) {
      state.captureRate = captureRate;
      state.ratio = captureRate / 16_000;
      state.pending = new Float32Array(0);
      state.position = 0;
    }
    const joined = new Float32Array(state.pending.length + frame.length);
    joined.set(state.pending);
    joined.set(frame, state.pending.length);
    const samples: number[] = [];
    while (state.position + 1 < joined.length) {
      const floor = Math.floor(state.position);
      const fraction = state.position - floor;
      samples.push(joined[floor]! * (1 - fraction) + joined[floor + 1]! * fraction);
      state.position += state.ratio;
    }
    const consumed = Math.min(Math.floor(state.position), Math.max(0, joined.length - 1));
    if (consumed > 0) {
      state.pending = joined.slice(consumed);
      state.position -= consumed;
    } else state.pending = joined;
    return convertToPcm(Float32Array.from(samples));
  };

  const convertToPcm = (samples: Float32Array) => {
    const output = new Int16Array(samples.length);
    for (let index = 0; index < samples.length; index++) {
      let sample = samples[index] ?? 0;
      if (sample > 1) sample = 1;
      else if (sample < -1) sample = -1;
      output[index] = sample < 0 ? Math.round(sample * 0x8000) : Math.round(sample * 0x7fff);
    }
    return output;
  };

  it("keeps resampling phase continuous across callback boundaries", () => {
    const resampler = new StreamingPcm16Resampler();
    const first = resampler.process(Float32Array.from({ length: 5 }, (_, i) => i / 5), 48_000);
    const second = resampler.process(Float32Array.from({ length: 5 }, (_, i) => (i + 5) / 5), 48_000);
    expect(first.length + second.length).toBe(3);
    expect(second[0]).toBeGreaterThan(first.at(-1) ?? -1);
  });

  it("does not reset when a callback ends mid-ratio", () => {
    const resampler = new StreamingPcm16Resampler();
    const first = resampler.process(Float32Array.from({ length: 4 }, (_, i) => i / 4), 44_100);
    const second = resampler.process(Float32Array.from({ length: 4 }, (_, i) => (i + 4) / 4), 44_100);
    expect(first.length + second.length).toBe(3);
    expect(second[0]).toBeGreaterThan(first[0] ?? -1);
  });

  it("preserves the 48 kHz tail before switching to 44.1 kHz", () => {
    const resampler = new StreamingPcm16Resampler();
    const firstRate = Float32Array.from({ length: 4 }, (_, i) => (i + 1) / 10);
    const secondRate = Float32Array.from({ length: 8 }, (_, i) => (i + 5) / 10);
    const before = resampler.process(firstRate, 48_000);
    const after = resampler.process(secondRate, 44_100);
    expect(before.length).toBe(1);
    expect(after.length).toBe(4);
    expect(after[0]).toBe(13107); // The old 48 kHz endpoint precedes new-rate output.
  });

  it("flushes the resampled tail before direct 16 kHz samples", () => {
    const resampler = new StreamingPcm16Resampler();
    resampler.process(Float32Array.of(0.1, 0.2, 0.3, 0.4), 48_000);
    const switched = resampler.process(Float32Array.of(0.5, 0.6, 0.7), 16_000);
    expect(Array.from(switched)).toEqual([13107, 16384, 19660, 22937]);
  });

  it("keeps same-rate callback boundaries continuous without flushing", () => {
    const resampler = new StreamingPcm16Resampler();
    const first = resampler.process(Float32Array.of(0, 0.3, 0.6, 0.9), 48_000);
    const second = resampler.process(Float32Array.of(0.2, 0.5, 0.8), 48_000);
    expect(first.length + second.length).toBe(2);
    expect(Array.from([...first, ...second])).toEqual([0, 29490]);
  });

  it.each([16_000, 22_050, 44_100, 48_000, 96_000])("matches the prior sample sequence exactly at %i Hz across uneven callback boundaries", (rate) => {
    const actual = new StreamingPcm16Resampler();
    const expected = { captureRate: 16_000, ratio: 1, pending: new Float32Array(0), position: 0 };
    const source = Float32Array.from({ length: 997 }, (_, index) => Math.sin(index * 0.17) * 1.4);
    const chunks = [1, 2, 7, 31, 3, 64, 5, 127, 11, 256, 490];
    let offset = 0;
    for (const chunkSize of chunks) {
      const frame = source.slice(offset, offset + chunkSize);
      offset += frame.length;
      expect(actual.process(frame, rate)).toEqual(legacyProcess(expected, frame, rate));
    }
    expect(offset).toBe(source.length);
  });

  it("preserves saturation and resets interpolation phase for canonical-rate callbacks", () => {
    const resampler = new StreamingPcm16Resampler();
    expect(Array.from(resampler.process(Float32Array.of(-2, -1, 0, 1, 2), 16_000))).toEqual([-32768, -32768, 0, 32767, 32767]);
    resampler.process(Float32Array.of(0, 1), 48_000);
    const canonical = resampler.process(Float32Array.of(0.25, -0.25), 16_000);
    expect(Array.from(canonical)).toEqual([8192, -8192]);
  });
});
