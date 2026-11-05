import { afterEach, describe, expect, it, vi } from "vitest";
import { audioWindow, decodeWaveform, formatAudioTime, MAX_AUDIO_ANALYSIS_BYTES, readAudioBytes, waveformPeaks } from "../audioAnalysis";

afterEach(() => { vi.unstubAllGlobals(); });

describe("audio presentation math", () => {
  it("formats hour-long recordings and rejects unknown/invalid times", () => {
    expect(formatAudioTime(119.9)).toBe("1:59");
    expect(formatAudioTime(3661)).toBe("1:01:01");
    expect(formatAudioTime(5.25, true)).toBe("0:05.2");
    for (const time of [NaN, Infinity, -1]) expect(formatAudioTime(time)).toBe("–:––");
  });
  it("clamps a zoomed window at either end of the track", () => {
    expect(audioWindow(120, 4, 115)).toEqual({ start: 90, end: 120, span: 30 });
    expect(audioWindow(120, 4, -15)).toEqual({ start: 0, end: 30, span: 30 });
    expect(audioWindow(120, 1, 50)).toEqual({ start: 0, end: 120, span: 120 });
  });
  it("captures transients from either channel and represents silence honestly", () => {
    const left = new Float32Array(4096);
    const right = new Float32Array(4096);
    left[1] = 0.5;
    right[4095] = -1;
    const peaks = waveformPeaks({ length: 4096, numberOfChannels: 2, getChannelData: (channel) => channel === 0 ? left : right });
    expect(peaks).toHaveLength(2048);
    expect(peaks[0]).toBe(0.5);
    expect(peaks[2047]).toBe(1);
    expect(peaks[100]).toBe(0);
    expect(waveformPeaks({ length: 10, numberOfChannels: 1, getChannelData: () => new Float32Array(10) })).toEqual(Array.from({ length: 10 }, () => 0));
  });
});

describe("bounded optional analysis", () => {
  it("reads bytes with same-origin credentials and an abort signal", async () => {
    const request = vi.fn().mockResolvedValue(new Response(new Uint8Array([1, 2, 3])));
    vi.stubGlobal("fetch", request);
    const signal = new AbortController().signal;
    expect(new Uint8Array(await readAudioBytes("/track", signal))).toEqual(new Uint8Array([1, 2, 3]));
    expect(request).toHaveBeenCalledWith("/track", { signal, credentials: "same-origin" });
  });
  it("rejects a failed or oversized response before decoding", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response(null, { status: 409 }))
      .mockResolvedValueOnce(new Response(new Uint8Array([1]), { headers: { "Content-Length": String(MAX_AUDIO_ANALYSIS_BYTES + 1) } })));
    const signal = new AbortController().signal;
    await expect(readAudioBytes("/stale", signal)).rejects.toThrow();
    await expect(readAudioBytes("/large", signal)).rejects.toThrow("size limit");
  });
  it("enforces the byte limit even when the server omits Content-Length", async () => {
    let cancelled = false;
    const body = new ReadableStream({ start(controller) { controller.enqueue(new Uint8Array(MAX_AUDIO_ANALYSIS_BYTES)); controller.enqueue(new Uint8Array([1])); }, cancel() { cancelled = true; } });
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(body)));
    await expect(readAudioBytes("/large", new AbortController().signal)).rejects.toThrow("oversized");
    expect(cancelled).toBe(true);
  });
  it("leaves waveform analysis optional when the decoder is absent or rejects the codec", async () => {
    vi.stubGlobal("OfflineAudioContext", undefined);
    await expect(decodeWaveform(new ArrayBuffer(1))).resolves.toEqual([]);
    vi.stubGlobal("OfflineAudioContext", class { decodeAudioData() { return Promise.reject(new Error("Unsupported codec")); } });
    await expect(decodeWaveform(new ArrayBuffer(1))).rejects.toThrow("Unsupported codec");
  });
});
