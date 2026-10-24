import { describe, expect, it } from "vitest";
import { StreamingPcm16Resampler } from "./pcm";

describe("StreamingPcm16Resampler", () => {
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
});
