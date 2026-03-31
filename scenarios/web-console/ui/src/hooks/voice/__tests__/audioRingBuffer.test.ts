import { describe, it, expect } from "vitest";
import { AudioRingBuffer, downsample } from "../audioUtils";

describe("AudioRingBuffer", () => {
  const rate = 16000;

  it("starts empty", () => {
    const buf = new AudioRingBuffer(1, rate);
    expect(buf.totalWritten).toBe(0);
    expect(buf.extractLast(100)).toEqual(new Float32Array(0));
  });

  it("writes and reads back data correctly", () => {
    const buf = new AudioRingBuffer(1, rate);
    const data = new Float32Array([1, 2, 3, 4, 5]);
    buf.write(data);
    expect(buf.totalWritten).toBe(5);
    const result = buf.extractLast(5);
    expect(Array.from(result)).toEqual([1, 2, 3, 4, 5]);
  });

  it("extractLast returns only available data when requesting more than written", () => {
    const buf = new AudioRingBuffer(1, rate);
    buf.write(new Float32Array([10, 20, 30]));
    const result = buf.extractLast(100);
    expect(result.length).toBe(3);
    expect(Array.from(result)).toEqual([10, 20, 30]);
  });

  it("wraps around correctly when buffer is full", () => {
    // Small buffer: 5 samples capacity
    const buf = new AudioRingBuffer(5 / rate, rate);
    expect(buf.capacity).toBe(5);

    // Write 3 samples
    buf.write(new Float32Array([1, 2, 3]));
    // Write 4 more (total 7, wraps around)
    buf.write(new Float32Array([4, 5, 6, 7]));

    expect(buf.totalWritten).toBe(7);
    // Last 5 samples should be 3,4,5,6,7
    const result = buf.extractLast(5);
    expect(Array.from(result)).toEqual([3, 4, 5, 6, 7]);
  });

  it("handles write larger than capacity", () => {
    const buf = new AudioRingBuffer(3 / rate, rate);
    expect(buf.capacity).toBe(3);
    buf.write(new Float32Array([1, 2, 3, 4, 5, 6]));
    const result = buf.extractLast(3);
    expect(Array.from(result)).toEqual([4, 5, 6]);
  });

  it("extractLast works after multiple wrap-arounds", () => {
    const buf = new AudioRingBuffer(4 / rate, rate);
    for (let i = 0; i < 10; i++) {
      buf.write(new Float32Array([i]));
    }
    expect(buf.totalWritten).toBe(10);
    const result = buf.extractLast(4);
    expect(Array.from(result)).toEqual([6, 7, 8, 9]);
  });

  describe("mark / extractSinceMark", () => {
    it("extracts samples written after the mark", () => {
      const buf = new AudioRingBuffer(1, rate);
      buf.write(new Float32Array([1, 2, 3]));
      const m = buf.mark();
      buf.write(new Float32Array([4, 5, 6]));
      const result = buf.extractSinceMark(m);
      expect(Array.from(result)).toEqual([4, 5, 6]);
    });

    it("returns empty when no samples written after mark", () => {
      const buf = new AudioRingBuffer(1, rate);
      buf.write(new Float32Array([1, 2, 3]));
      const m = buf.mark();
      expect(buf.extractSinceMark(m)).toEqual(new Float32Array(0));
    });

    it("caps at capacity when mark is very old", () => {
      const buf = new AudioRingBuffer(4 / rate, rate);
      const m = buf.mark(); // mark at 0
      buf.write(new Float32Array([1, 2, 3, 4, 5, 6, 7, 8])); // way more than capacity
      const result = buf.extractSinceMark(m);
      expect(result.length).toBe(4); // limited by capacity
      expect(Array.from(result)).toEqual([5, 6, 7, 8]);
    });
  });

  it("reset clears the buffer", () => {
    const buf = new AudioRingBuffer(1, rate);
    buf.write(new Float32Array([1, 2, 3, 4, 5]));
    buf.reset();
    expect(buf.totalWritten).toBe(0);
    expect(buf.extractLast(5)).toEqual(new Float32Array(0));
  });

  it("capacity is correct for given duration", () => {
    const buf = new AudioRingBuffer(3, 48000);
    expect(buf.capacity).toBe(144000); // 3s * 48kHz
  });
});

describe("downsample", () => {
  it("returns same buffer when rates are equal", () => {
    const input = new Float32Array([1, 2, 3, 4]);
    const result = downsample(input, 16000, 16000);
    expect(result).toBe(input); // same reference
  });

  it("halves length for 2:1 downsampling", () => {
    const input = new Float32Array(1000);
    for (let i = 0; i < 1000; i++) input[i] = Math.sin(2 * Math.PI * 100 * i / 48000);
    const result = downsample(input, 48000, 24000);
    expect(result.length).toBe(500);
  });

  it("produces correct output length for 48kHz -> 16kHz", () => {
    const numSamples = 48000; // 1 second at 48kHz
    const input = new Float32Array(numSamples);
    const result = downsample(input, 48000, 16000);
    expect(result.length).toBe(16000);
  });

  it("preserves low-frequency content", () => {
    // 100Hz sine at 48kHz -> 16kHz should still look like a sine
    const fromRate = 48000;
    const toRate = 16000;
    const freq = 100;
    const duration = 0.1; // 100ms
    const input = new Float32Array(Math.round(fromRate * duration));
    for (let i = 0; i < input.length; i++) {
      input[i] = Math.sin(2 * Math.PI * freq * i / fromRate);
    }

    const output = downsample(input, fromRate, toRate);

    // Check that output samples approximate the expected sine values
    for (let i = 0; i < output.length; i++) {
      const expected = Math.sin(2 * Math.PI * freq * i / toRate);
      expect(output[i]).toBeCloseTo(expected, 1);
    }
  });

  it("throws on upsample attempt", () => {
    expect(() => downsample(new Float32Array(100), 16000, 48000)).toThrow("Cannot upsample");
  });

  it("handles empty input", () => {
    const result = downsample(new Float32Array(0), 48000, 16000);
    expect(result.length).toBe(0);
  });
});
