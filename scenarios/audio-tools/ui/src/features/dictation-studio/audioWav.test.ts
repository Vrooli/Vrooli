import { describe, it, expect } from "vitest";

import { extractPcm16FromWav, pcm16DurationMs } from "./audioWav";

function buildWav(pcm: Uint8Array, sampleRate: number): ArrayBuffer {
  const buffer = new ArrayBuffer(44 + pcm.byteLength);
  const view = new DataView(buffer);
  const writeStr = (off: number, s: string) => {
    for (let i = 0; i < s.length; i++) view.setUint8(off + i, s.charCodeAt(i));
  };
  writeStr(0, "RIFF");
  view.setUint32(4, 36 + pcm.byteLength, true);
  writeStr(8, "WAVE");
  writeStr(12, "fmt ");
  view.setUint32(16, 16, true);
  view.setUint16(20, 1, true);
  view.setUint16(22, 1, true);
  view.setUint32(24, sampleRate, true);
  view.setUint32(28, sampleRate * 2, true);
  view.setUint16(32, 2, true);
  view.setUint16(34, 16, true);
  writeStr(36, "data");
  view.setUint32(40, pcm.byteLength, true);
  new Uint8Array(buffer, 44).set(pcm);
  return buffer;
}

describe("extractPcm16FromWav", () => {
  it("strips the RIFF container back to the raw PCM data chunk", () => {
    const pcm = new Uint8Array([1, 2, 3, 4, 5, 6, 7, 8]);
    const wav = buildWav(pcm, 16_000);
    const out = extractPcm16FromWav(wav);
    expect(Array.from(out.pcm)).toEqual([1, 2, 3, 4, 5, 6, 7, 8]);
    expect(out.sampleRateHz).toBe(16_000);
  });

  it("reads a non-default sample rate from the fmt chunk", () => {
    const out = extractPcm16FromWav(buildWav(new Uint8Array([0, 0]), 8_000));
    expect(out.sampleRateHz).toBe(8_000);
  });

  it("rejects a buffer that is not RIFF/WAVE", () => {
    const buf = new ArrayBuffer(16);
    expect(() => extractPcm16FromWav(buf)).toThrow();
  });

  it("rejects a buffer too small to be a WAV", () => {
    expect(() => extractPcm16FromWav(new ArrayBuffer(4))).toThrow();
  });

  it("throws when the data chunk is absent", () => {
    // RIFF/WAVE header with a fmt chunk but no data chunk.
    const buffer = new ArrayBuffer(36);
    const view = new DataView(buffer);
    const writeStr = (off: number, s: string) => {
      for (let i = 0; i < s.length; i++) view.setUint8(off + i, s.charCodeAt(i));
    };
    writeStr(0, "RIFF");
    view.setUint32(4, 28, true);
    writeStr(8, "WAVE");
    writeStr(12, "fmt ");
    view.setUint32(16, 16, true);
    view.setUint32(24, 16_000, true);
    expect(() => extractPcm16FromWav(buffer)).toThrow();
  });
});

describe("pcm16DurationMs", () => {
  it("derives duration from byte length and sample rate", () => {
    // 16000 samples * 2 bytes = 32000 bytes at 16 kHz = 1000 ms.
    expect(pcm16DurationMs(32_000, 16_000)).toBe(1_000);
  });

  it("returns 0 for a non-positive sample rate", () => {
    expect(pcm16DurationMs(1_000, 0)).toBe(0);
  });
});
