import { describe, expect, it } from "vitest";

import { concatInt16, encodeWavFromPcm16, floatTo16BitPCM, frameToCanonicalPcm16, pcm16ToWavBuffer, TARGET_SAMPLE_RATE } from "./pcm";

describe("floatTo16BitPCM", () => {
  it("maps full-scale samples to the int16 range", () => {
    const out = floatTo16BitPCM(new Float32Array([0, 1, -1, 0.5, -0.5]));
    expect(out[0]).toBe(0);
    expect(out[1]).toBe(0x7fff); // +1.0 -> max positive
    expect(out[2]).toBe(-0x8000); // -1.0 -> max negative
    expect(out[3]).toBe(Math.round(0.5 * 0x7fff));
    expect(out[4]).toBe(Math.round(-0.5 * 0x8000));
  });

  it("clamps out-of-range samples", () => {
    const out = floatTo16BitPCM(new Float32Array([2, -2]));
    expect(out[0]).toBe(0x7fff);
    expect(out[1]).toBe(-0x8000);
  });

  it("preserves length", () => {
    expect(floatTo16BitPCM(new Float32Array(320)).length).toBe(320);
  });
});

describe("frameToCanonicalPcm16", () => {
  it("is an identity (no resample) when capture rate equals the target", () => {
    const frame = new Float32Array([0.25, -0.25, 0.75]);
    const out = frameToCanonicalPcm16(frame, TARGET_SAMPLE_RATE);
    expect(out.length).toBe(3);
    expect(out[0]).toBe(Math.round(0.25 * 0x7fff));
  });

  it("downsamples 48 kHz to 16 kHz (~1/3 the samples)", () => {
    const frame = new Float32Array(48_000).fill(0.1); // 1s at 48k
    const out = frameToCanonicalPcm16(frame, 48_000);
    // downsample uses ceil(len / (from/to)); 48000 / 3 = 16000.
    expect(out.length).toBe(16_000);
  });
});

describe("concatInt16", () => {
  it("concatenates chunks in order", () => {
    const a = Int16Array.from([1, 2]);
    const b = Int16Array.from([3, 4, 5]);
    expect(Array.from(concatInt16([a, b]))).toEqual([1, 2, 3, 4, 5]);
  });

  it("returns an empty buffer for no chunks", () => {
    expect(concatInt16([]).length).toBe(0);
  });
});

describe("encodeWavFromPcm16", () => {
  it("produces a canonical 44-byte-header mono s16le WAV", () => {
    const pcm = Int16Array.from([0, 1000, -1000, 32767, -32768]);
    const blob = encodeWavFromPcm16(pcm, TARGET_SAMPLE_RATE);
    expect(blob.type).toBe("audio/wav");
    expect(blob.size).toBe(44 + pcm.length * 2);
  });

  it("writes a correct RIFF/WAVE header and sample rate", () => {
    const pcm = Int16Array.from([0, 256, -256]);
    const buf = pcm16ToWavBuffer(pcm, TARGET_SAMPLE_RATE);
    const view = new DataView(buf);
    const ascii = (off: number, len: number) =>
      String.fromCharCode(...new Uint8Array(buf.slice(off, off + len)));
    expect(ascii(0, 4)).toBe("RIFF");
    expect(ascii(8, 4)).toBe("WAVE");
    expect(ascii(12, 4)).toBe("fmt ");
    expect(view.getUint16(20, true)).toBe(1); // PCM
    expect(view.getUint16(22, true)).toBe(1); // mono
    expect(view.getUint32(24, true)).toBe(TARGET_SAMPLE_RATE);
    expect(view.getUint16(34, true)).toBe(16); // bits/sample
    expect(ascii(36, 4)).toBe("data");
    expect(view.getUint32(40, true)).toBe(pcm.length * 2);
    // First sample round-trips little-endian.
    expect(view.getInt16(44, true)).toBe(0);
    expect(view.getInt16(46, true)).toBe(256);
    expect(view.getInt16(48, true)).toBe(-256);
  });
});
