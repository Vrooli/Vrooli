// Tests for KokoroProvider's Phase-3 pipelined speakSequence behavior.
//
// jsdom does not implement HTMLAudioElement.play() as a Promise, so we
// monkey-patch the prototype here. URL.createObjectURL is also stubbed
// because jsdom returns "" by default.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { KokoroProvider, type KokoroSynthesizeWithMetricsFn } from "./KokoroProvider";

type AudioCtl = {
  triggerEnded: () => void;
  triggerError: () => void;
};

function installAudioStubs(): AudioCtl {
  const ctl: AudioCtl = {
    triggerEnded: () => {
      throw new Error("triggerEnded called before audio created");
    },
    triggerError: () => {
      throw new Error("triggerError called before audio created");
    },
  };
  vi.spyOn(HTMLMediaElement.prototype, "play").mockImplementation(function (this: HTMLMediaElement) {
    // Capture latest element so the test can dispatch 'ended' on it.
    ctl.triggerEnded = () => this.dispatchEvent(new Event("ended"));
    ctl.triggerError = () => this.dispatchEvent(new Event("error"));
    return Promise.resolve();
  });
  vi.spyOn(HTMLMediaElement.prototype, "pause").mockImplementation(() => undefined);
  vi.spyOn(HTMLMediaElement.prototype, "load").mockImplementation(() => undefined);

  if (typeof URL.createObjectURL !== "function") {
    Object.defineProperty(URL, "createObjectURL", { value: vi.fn(), configurable: true });
  }
  if (typeof URL.revokeObjectURL !== "function") {
    Object.defineProperty(URL, "revokeObjectURL", { value: vi.fn(), configurable: true });
  }
  vi.spyOn(URL, "createObjectURL").mockImplementation(() => "blob:fake-" + Math.random().toString(36).slice(2));
  vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => undefined);

  return ctl;
}

function deferred<T>() {
  let resolve!: (v: T) => void;
  let reject!: (e: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

describe("KokoroProvider.speakSequence (pipelined)", () => {
  let ctl: AudioCtl;
  beforeEach(() => { ctl = installAudioStubs(); });
  afterEach(() => { vi.restoreAllMocks(); });

  it("plays paragraphs in order even when synth resolves out of order", async () => {
    const order: number[] = [];
    const d1 = deferred<{ blob: Blob; metrics: { requestId: string; synthStartMs: number; totalChars: number } }>();
    const d2 = deferred<typeof d1.promise extends Promise<infer T> ? T : never>();
    const synthesizeWithMetrics: KokoroSynthesizeWithMetricsFn = vi.fn(async (text: string) => {
      order.push(text === "one" ? 1 : 2);
      return text === "one" ? d1.promise : d2.promise;
    });
    const provider = new KokoroProvider({ synthesizeWithMetrics });

    const playPromise = provider.speakSequence(["one", "two"]);

    // Resolve paragraph 2 BEFORE paragraph 1 to test order preservation.
    d2.resolve({ blob: new Blob([new Uint8Array([1, 2, 3])], { type: "audio/mpeg" }), metrics: { requestId: "r2", synthStartMs: 0, totalChars: 3 } });
    await Promise.resolve();
    d1.resolve({ blob: new Blob([new Uint8Array([4, 5, 6])], { type: "audio/mpeg" }), metrics: { requestId: "r1", synthStartMs: 0, totalChars: 3 } });

    // Drain microtasks then fire the two 'ended' events sequentially.
    await new Promise((r) => setTimeout(r, 0));
    ctl.triggerEnded();
    await new Promise((r) => setTimeout(r, 0));
    ctl.triggerEnded();

    await playPromise;
    expect(order).toEqual([1, 2]);
    expect(synthesizeWithMetrics).toHaveBeenCalledTimes(2);
  });

  it("starts playback as soon as paragraph 1 is ready (does not wait for paragraph N)", async () => {
    const d1 = deferred<{ blob: Blob; metrics: { requestId: string; synthStartMs: number; totalChars: number } }>();
    const dNever = deferred<typeof d1.promise extends Promise<infer T> ? T : never>();
    const synthesizeWithMetrics: KokoroSynthesizeWithMetricsFn = vi.fn(async (text: string) => {
      if (text === "first") return d1.promise;
      // Paragraph 2 never resolves until after playback of 1 starts.
      return dNever.promise;
    });
    const playSpy = vi.spyOn(HTMLMediaElement.prototype, "play");
    const provider = new KokoroProvider({ synthesizeWithMetrics });
    const playPromise = provider.speakSequence(["first", "second"]);

    // Resolve only paragraph 1 — paragraph 2 still pending.
    d1.resolve({ blob: new Blob([new Uint8Array([1])], { type: "audio/mpeg" }), metrics: { requestId: "r1", synthStartMs: 0, totalChars: 5 } });
    await new Promise((r) => setTimeout(r, 0));

    expect(playSpy).toHaveBeenCalledTimes(1); // playback of paragraph 1 started before 2 resolved

    // Now resolve paragraph 2 and let the sequence drain.
    dNever.resolve({ blob: new Blob([new Uint8Array([2])], { type: "audio/mpeg" }), metrics: { requestId: "r2", synthStartMs: 0, totalChars: 6 } });
    ctl.triggerEnded(); // end paragraph 1
    await new Promise((r) => setTimeout(r, 0));
    ctl.triggerEnded(); // end paragraph 2
    await playPromise;
  });

  it("skips 0-byte paragraphs without interrupting playback of neighbors", async () => {
    const synthesizeWithMetrics: KokoroSynthesizeWithMetricsFn = vi.fn(async (text: string) => ({
      blob: new Blob([text === "silent" ? new Uint8Array() : new Uint8Array([1])], { type: "audio/mpeg" }),
      metrics: { requestId: text, synthStartMs: 0, totalChars: text.length },
    }));
    const playSpy = vi.spyOn(HTMLMediaElement.prototype, "play");
    const provider = new KokoroProvider({ synthesizeWithMetrics });

    const playPromise = provider.speakSequence(["a", "silent", "b"]);
    // Drain so all three synths resolve and playback starts on "a".
    await new Promise((r) => setTimeout(r, 0));
    ctl.triggerEnded(); // end "a"
    await new Promise((r) => setTimeout(r, 0));
    // "silent" is skipped (0 bytes); next ended fires for "b".
    ctl.triggerEnded(); // end "b"
    await playPromise;

    // Only "a" and "b" should have triggered play() — "silent" was skipped.
    expect(playSpy).toHaveBeenCalledTimes(2);
  });

  it("abort cancels all pending synth promises", async () => {
    const aborted: AbortSignal[] = [];
    type Result = { blob: Blob; metrics: { requestId: string; synthStartMs: number; totalChars: number } };
    const synthesizeWithMetrics: KokoroSynthesizeWithMetricsFn = async (_text, _voice, _speed, signal) => {
      if (signal) aborted.push(signal);
      return new Promise<Result>((_resolve, reject) => {
        signal?.addEventListener("abort", () => reject(new DOMException("aborted", "AbortError")));
      });
    };
    const provider = new KokoroProvider({ synthesizeWithMetrics });

    const playPromise = provider.speakSequence(["a", "b", "c"]);
    // Allow synths to be kicked off (concurrency window).
    await new Promise((r) => setTimeout(r, 0));
    provider.stop();

    await expect(playPromise).rejects.toThrow();
    // Every pending synth signal must have been aborted.
    expect(aborted.length).toBeGreaterThan(0);
    for (const s of aborted) expect(s.aborted).toBe(true);
  });
});
