/**
 * Unit tests for KokoroProvider.
 *
 * Audio playback is driven by an HTMLAudioElement whose `play()` method
 * jsdom doesn't implement.  We replace `window.Audio` with a FakeAudio that
 * (a) extends EventTarget so event listeners wired in the constructor work,
 * (b) exposes vi.fn() spies for every method the provider calls, and
 * (c) saves the last created instance so individual tests can dispatch events
 * to drive the internal state machine.
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// Mock ttsApi so the default (no-synthesize) constructor path can be exercised
// without a real network connection.
const synthesizeTTSMock = vi.fn();
vi.mock("../../api/tts", () => ({
  synthesizeTTS: (...args: unknown[]) => synthesizeTTSMock(...args),
}));

import { KokoroProvider } from "./KokoroProvider";

// ─── FakeAudio ────────────────────────────────────────────────────────────────

class FakeAudio extends EventTarget {
  /** Tracks the most recently constructed FakeAudio instance. */
  static last: FakeAudio | null = null;

  src = "";
  muted = false;
  paused = true;
  currentTime = 0;
  duration = NaN;
  playbackRate = 1;
  volume = 1;
  error: { message?: string } | null = null;

  play = vi.fn().mockResolvedValue(undefined);
  pause = vi.fn().mockImplementation(() => {
    this.paused = true;
  });
  load = vi.fn();
  removeAttribute = vi.fn().mockImplementation(() => {
    this.src = "";
  });

  constructor() {
    super();
    // Use a static property so we don't need a module-level `this` alias.
    FakeAudio.last = this;
  }
}

// ─── helpers ──────────────────────────────────────────────────────────────────

/** Flush all pending microtasks so mocked Promises settle. */
const flushMicrotasks = () => new Promise<void>((r) => setTimeout(r, 0));

/** Minimal non-empty Blob that looks like audio. */
const makeBlob = (size = 4) => new Blob([new Uint8Array(size)], { type: "audio/mpeg" });
const emptyBlob = () => new Blob([], { type: "audio/mpeg" });

// ─── suite ────────────────────────────────────────────────────────────────────

describe("KokoroProvider", () => {
  let createObjectURLMock: ReturnType<typeof vi.fn>;
  let revokeObjectURLMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    FakeAudio.last = null;
    vi.clearAllMocks();
    // Replace global Audio with our FakeAudio before each test.
    vi.stubGlobal("Audio", FakeAudio);
    // URL.createObjectURL / revokeObjectURL are not implemented in jsdom.
    // Define them directly (configurable so afterEach can clean up).
    createObjectURLMock = vi.fn().mockReturnValue("blob:fake-url");
    revokeObjectURLMock = vi.fn();
    Object.defineProperty(URL, "createObjectURL", {
      value: createObjectURLMock,
      writable: true,
      configurable: true,
    });
    Object.defineProperty(URL, "revokeObjectURL", {
      value: revokeObjectURLMock,
      writable: true,
      configurable: true,
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
    // Remove properties we added so they don't leak into other test files.
    // @ts-expect-error — URL.createObjectURL is not normally on URL in jsdom
    delete URL.createObjectURL;
    // @ts-expect-error — URL.revokeObjectURL is not normally on URL in jsdom
    delete URL.revokeObjectURL;
  });

  // ── capabilities ────────────────────────────────────────────────────────────

  it("declares correct capabilities", () => {
    const p = new KokoroProvider({});
    expect(p.capabilities).toEqual({
      canPause: true,
      canSeek: true,
      canAdjustSpeed: true,
      canAdjustVolume: true,
    });
  });

  // ── unlock ──────────────────────────────────────────────────────────────────

  it("unlock() resolves true on first call and sets unlocked", async () => {
    const p = new KokoroProvider({});
    const result = await p.unlock();
    expect(result).toBe(true);
    expect(p.isUnlocked()).toBe(true);
  });

  it("unlock() skips re-play when already unlocked (not forced)", async () => {
    const p = new KokoroProvider({});
    await p.unlock();
    const audio = FakeAudio.last!;
    const callCount = audio.play.mock.calls.length;
    await p.unlock(); // second call, not forced
    expect(audio.play.mock.calls.length).toBe(callCount); // no extra play()
  });

  it("unlock(force=true) re-plays even if already unlocked", async () => {
    const p = new KokoroProvider({});
    await p.unlock();
    const audio = FakeAudio.last!;
    const callCount = audio.play.mock.calls.length;
    await p.unlock(true);
    expect(audio.play.mock.calls.length).toBeGreaterThan(callCount);
  });

  it("unlock() returns false when play() rejects", async () => {
    const p = new KokoroProvider({});
    FakeAudio.last!.play.mockRejectedValueOnce(new Error("NotAllowedError"));
    const result = await p.unlock();
    expect(result).toBe(false);
  });

  it("unlock() skips silent play when already speaking and not paused", async () => {
    const p = new KokoroProvider({});
    // Simulate active playback by directly setting private fields.
    // @ts-expect-error — accessing private for test setup
    p._isSpeaking = true;
    FakeAudio.last!.paused = false;
    const callCount = FakeAudio.last!.play.mock.calls.length;
    await p.unlock(); // should take the early-return path
    expect(p.isUnlocked()).toBe(true);
    expect(FakeAudio.last!.play.mock.calls.length).toBe(callCount);
  });

  // ── speak – success ─────────────────────────────────────────────────────────

  it("speak() resolves when audio 'ended' fires", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });

    const speakPromise = p.speak("hello");
    await flushMicrotasks();
    FakeAudio.last!.dispatchEvent(new Event("ended"));
    await speakPromise;

    expect(synthFn).toHaveBeenCalledWith("hello", undefined, undefined, expect.any(AbortSignal));
    expect(createObjectURLMock).toHaveBeenCalledTimes(1);
    expect(p.isSpeaking).toBe(false);
  });

  it("speak() passes voice and rate opts to synthesize", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    const speakPromise = p.speak("hi", { voice: "en-v1", rate: 1.2 });
    await flushMicrotasks();
    FakeAudio.last!.dispatchEvent(new Event("ended"));
    await speakPromise;
    expect(synthFn).toHaveBeenCalledWith("hi", "en-v1", 1.2, expect.any(AbortSignal));
  });

  it("speak() silently resolves and skips playback for 0-byte blob", async () => {
    const synthFn = vi.fn().mockResolvedValue(emptyBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    await p.speak("---");
    expect(FakeAudio.last!.play).not.toHaveBeenCalled();
    expect(p.isSpeaking).toBe(false);
  });

  it("speak() rejects when audio 'error' fires", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });

    const speakPromise = p.speak("hello");
    await flushMicrotasks();
    FakeAudio.last!.error = { message: "decode error" };
    FakeAudio.last!.dispatchEvent(new Event("error"));
    await expect(speakPromise).rejects.toThrow("decode error");
    expect(p.isSpeaking).toBe(false);
  });

  it("speak() rejects with generic message when audio.error is null", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });

    const speakPromise = p.speak("hello");
    await flushMicrotasks();
    FakeAudio.last!.error = null; // no MediaError set
    FakeAudio.last!.dispatchEvent(new Event("error"));
    await expect(speakPromise).rejects.toThrow("Audio playback error");
  });

  it("speak() rejects when play() rejects", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    FakeAudio.last!.play.mockRejectedValueOnce(new Error("NotAllowedError"));

    await expect(p.speak("hello")).rejects.toThrow("NotAllowedError");
    expect(p.isSpeaking).toBe(false);
  });

  it("speak() rejects with Error wrapping non-Error play rejection", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    FakeAudio.last!.play.mockRejectedValueOnce("string-rejection");

    await expect(p.speak("hello")).rejects.toThrow("string-rejection");
  });

  it("speak() throws when synthesize rejects", async () => {
    const synthFn = vi.fn().mockRejectedValue(new Error("network error"));
    const p = new KokoroProvider({ synthesize: synthFn });
    await expect(p.speak("hello")).rejects.toThrow("network error");
    expect(p.isSpeaking).toBe(false);
  });

  it("speak() revokes previous blob URL before creating new one", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });

    // First speak
    const p1 = p.speak("first");
    await flushMicrotasks();
    FakeAudio.last!.dispatchEvent(new Event("ended"));
    await p1;

    // Second speak
    const p2 = p.speak("second");
    await flushMicrotasks();
    FakeAudio.last!.dispatchEvent(new Event("ended"));
    await p2;

    // First URL should have been revoked when second speak started
    expect(revokeObjectURLMock).toHaveBeenCalledWith("blob:fake-url");
  });

  it("speak() aborts prior speak when called again", async () => {
    // First speak resolves synthesize immediately so it reaches the "wait for ended" stage.
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });

    const first = p.speak("one");
    await flushMicrotasks(); // let first speak set up playbackReject
    // Second speak calls stop() internally, which rejects the first speak.
    const second = p.speak("two");
    await expect(first).rejects.toMatchObject({ name: "AbortError" });
    // Clean up second speak.
    await flushMicrotasks();
    FakeAudio.last!.dispatchEvent(new Event("ended"));
    await second;
  });

  // ── speakSequence ───────────────────────────────────────────────────────────

  it("speakSequence() returns immediately for empty array", async () => {
    const synthFn = vi.fn();
    const p = new KokoroProvider({ synthesize: synthFn });
    await p.speakSequence([]);
    expect(synthFn).not.toHaveBeenCalled();
  });

  it("speakSequence() delegates single-item to speak()", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    const seq = p.speakSequence(["only one"]);
    await flushMicrotasks();
    FakeAudio.last!.dispatchEvent(new Event("ended"));
    await seq;
    expect(synthFn).toHaveBeenCalledTimes(1);
  });

  it("speakSequence() concatenates multiple blobs and plays once", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    const seq = p.speakSequence(["one", "two", "three"]);
    await flushMicrotasks();
    FakeAudio.last!.dispatchEvent(new Event("ended"));
    await seq;
    expect(synthFn).toHaveBeenCalledTimes(3);
    // play() called once for the combined blob
    expect(FakeAudio.last!.play).toHaveBeenCalledTimes(1);
  });

  it("speakSequence() passes voice and rate opts to synthesize for each segment", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    const seq = p.speakSequence(["a", "b"], { voice: "en-v1", rate: 1.5 });
    await flushMicrotasks();
    FakeAudio.last!.dispatchEvent(new Event("ended"));
    await seq;
    expect(synthFn).toHaveBeenCalledWith("a", "en-v1", 1.5, expect.any(AbortSignal));
    expect(synthFn).toHaveBeenCalledWith("b", "en-v1", 1.5, expect.any(AbortSignal));
  });

  it("speakSequence() skips empty blobs but plays non-empty ones", async () => {
    const synthFn = vi
      .fn()
      .mockResolvedValueOnce(emptyBlob())
      .mockResolvedValueOnce(makeBlob())
      .mockResolvedValueOnce(emptyBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    const seq = p.speakSequence(["a", "b", "c"]);
    await flushMicrotasks();
    FakeAudio.last!.dispatchEvent(new Event("ended"));
    await seq;
    expect(FakeAudio.last!.play).toHaveBeenCalledTimes(1);
  });

  it("speakSequence() silently resolves when all blobs are empty", async () => {
    const synthFn = vi.fn().mockResolvedValue(emptyBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    await p.speakSequence(["a", "b"]);
    expect(FakeAudio.last!.play).not.toHaveBeenCalled();
  });

  it("speakSequence() rejects when synthesize throws", async () => {
    const synthFn = vi.fn().mockRejectedValue(new Error("synth fail"));
    const p = new KokoroProvider({ synthesize: synthFn });
    await expect(p.speakSequence(["x", "y"])).rejects.toThrow("synth fail");
    expect(p.isSpeaking).toBe(false);
  });

  it("speakSequence() rejects when audio 'error' fires", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    const seq = p.speakSequence(["a", "b"]);
    await flushMicrotasks();
    FakeAudio.last!.dispatchEvent(new Event("error"));
    await expect(seq).rejects.toThrow("Audio playback error");
  });

  // ── speakFromBlob ────────────────────────────────────────────────────────────

  it("speakFromBlob() plays a pre-fetched blob", async () => {
    const p = new KokoroProvider({});
    const promise = p.speakFromBlob(makeBlob());
    await flushMicrotasks();
    FakeAudio.last!.dispatchEvent(new Event("ended"));
    await promise;
    expect(createObjectURLMock).toHaveBeenCalled();
  });

  it("speakFromBlob() returns immediately for empty blob", async () => {
    const p = new KokoroProvider({});
    await p.speakFromBlob(emptyBlob());
    expect(FakeAudio.last!.play).not.toHaveBeenCalled();
    expect(p.isSpeaking).toBe(false);
  });

  it("speakFromBlob() rejects when play() rejects with Error", async () => {
    const p = new KokoroProvider({});
    FakeAudio.last!.play.mockRejectedValueOnce(new Error("blocked"));
    await expect(p.speakFromBlob(makeBlob())).rejects.toThrow("blocked");
  });

  it("speakFromBlob() wraps non-Error play rejection in an Error", async () => {
    const p = new KokoroProvider({});
    FakeAudio.last!.play.mockRejectedValueOnce("string-fail");
    await expect(p.speakFromBlob(makeBlob())).rejects.toThrow("string-fail");
  });

  // ── stop ────────────────────────────────────────────────────────────────────

  it("stop() rejects a pending speak promise", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    const speakPromise = p.speak("hello");
    await flushMicrotasks(); // let speak get to the await-for-ended stage

    p.stop();
    await expect(speakPromise).rejects.toMatchObject({ name: "AbortError" });
    expect(p.isSpeaking).toBe(false);
  });

  it("stop() pauses audio when not already paused", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    const sp = p.speak("hi").catch(() => {});
    await flushMicrotasks();
    FakeAudio.last!.paused = false; // simulate playing
    p.stop();
    await sp;
    expect(FakeAudio.last!.pause).toHaveBeenCalled();
  });

  it("stop() skips pause when audio is already paused", () => {
    const p = new KokoroProvider({});
    FakeAudio.last!.paused = true;
    p.stop();
    expect(FakeAudio.last!.pause).not.toHaveBeenCalled();
  });

  it("stop() revokes blobUrl", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    const sp = p.speak("hi").catch(() => {});
    await flushMicrotasks();
    p.stop();
    await sp;
    expect(revokeObjectURLMock).toHaveBeenCalledWith("blob:fake-url");
  });

  // ── isSpeaking ───────────────────────────────────────────────────────────────

  it("isSpeaking is false initially", () => {
    const p = new KokoroProvider({});
    expect(p.isSpeaking).toBe(false);
  });

  it("isSpeaking is true while pending speak and false after ended", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    const sp = p.speak("hello");
    await flushMicrotasks();
    expect(p.isSpeaking).toBe(true);
    FakeAudio.last!.dispatchEvent(new Event("ended"));
    await sp;
    expect(p.isSpeaking).toBe(false);
  });

  // ── pause / resume ───────────────────────────────────────────────────────────

  it("pause() pauses audio when speaking and not already paused", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    const sp = p.speak("hi").catch(() => {});
    await flushMicrotasks();
    p.pause();
    expect(FakeAudio.last!.pause).toHaveBeenCalled();
    p.stop();
    await sp;
  });

  it("pause() is a no-op when not speaking", () => {
    const p = new KokoroProvider({});
    p.pause();
    expect(FakeAudio.last!.pause).not.toHaveBeenCalled();
  });

  it("pause() is a no-op when already paused", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    const sp = p.speak("hi").catch(() => {});
    await flushMicrotasks();
    p.pause(); // first pause
    const pauseCallCount = FakeAudio.last!.pause.mock.calls.length;
    p.pause(); // should be no-op
    expect(FakeAudio.last!.pause.mock.calls.length).toBe(pauseCallCount);
    p.stop();
    await sp;
  });

  it("resume() calls audio.play() when speaking and paused", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    const sp = p.speak("hi").catch(() => {});
    await flushMicrotasks();
    p.pause();
    const playCallsBefore = FakeAudio.last!.play.mock.calls.length;
    p.resume();
    expect(FakeAudio.last!.play.mock.calls.length).toBe(playCallsBefore + 1);
    p.stop();
    await sp;
  });

  it("resume() is a no-op when not speaking", () => {
    const p = new KokoroProvider({});
    const playBefore = FakeAudio.last!.play.mock.calls.length;
    p.resume();
    expect(FakeAudio.last!.play.mock.calls.length).toBe(playBefore);
  });

  it("resume() is a no-op when speaking but not paused", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    const sp = p.speak("hi").catch(() => {});
    await flushMicrotasks();
    const playBefore = FakeAudio.last!.play.mock.calls.length;
    p.resume(); // not paused, no-op
    expect(FakeAudio.last!.play.mock.calls.length).toBe(playBefore);
    p.stop();
    await sp;
  });

  // ── seek ─────────────────────────────────────────────────────────────────────

  it("seek() clamps to [0, duration] and sets currentTime", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    const sp = p.speak("hi").catch(() => {});
    await flushMicrotasks();
    FakeAudio.last!.duration = 10;
    p.seek(5);
    expect(FakeAudio.last!.currentTime).toBe(5);
    p.seek(-1); // below 0 → clamped to 0
    expect(FakeAudio.last!.currentTime).toBe(0);
    p.seek(100); // above duration → clamped to 10
    expect(FakeAudio.last!.currentTime).toBe(10);
    p.stop();
    await sp;
  });

  it("seek() is a no-op when not speaking", () => {
    const p = new KokoroProvider({});
    FakeAudio.last!.duration = 10;
    p.seek(5);
    // currentTime should stay at 0 (initial jsdom value)
    expect(FakeAudio.last!.currentTime).toBe(0);
  });

  it("seek() is a no-op when duration is not finite (NaN)", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    const sp = p.speak("hi").catch(() => {});
    await flushMicrotasks();
    // duration is NaN by default in FakeAudio
    p.seek(5);
    expect(FakeAudio.last!.currentTime).toBe(0);
    p.stop();
    await sp;
  });

  // ── setPlaybackRate / setVolume ──────────────────────────────────────────────

  it("setPlaybackRate() sets playbackRate on audio", () => {
    const p = new KokoroProvider({});
    p.setPlaybackRate(1.5);
    expect(FakeAudio.last!.playbackRate).toBe(1.5);
  });

  it("setVolume() clamps to [0,1]", () => {
    const p = new KokoroProvider({});
    p.setVolume(0.7);
    expect(FakeAudio.last!.volume).toBe(0.7);
    p.setVolume(2);
    expect(FakeAudio.last!.volume).toBe(1);
    p.setVolume(-0.5);
    expect(FakeAudio.last!.volume).toBe(0);
  });

  // ── getPlaybackState ─────────────────────────────────────────────────────────

  it("getPlaybackState() returns current snapshot", () => {
    const p = new KokoroProvider({});
    FakeAudio.last!.currentTime = 3;
    FakeAudio.last!.duration = 10;
    FakeAudio.last!.playbackRate = 1.5;
    FakeAudio.last!.volume = 0.8;
    const state = p.getPlaybackState();
    expect(state.currentTime).toBe(3);
    expect(state.duration).toBe(10);
    expect(state.isPaused).toBe(false);
    expect(state.playbackRate).toBe(1.5);
    expect(state.volume).toBe(0.8);
    expect(state.isMuted).toBe(false);
    expect(state.capabilities).toBe(p.capabilities);
  });

  it("getPlaybackState() returns null duration when NaN", () => {
    const p = new KokoroProvider({});
    // duration is NaN (default FakeAudio)
    expect(p.getPlaybackState().duration).toBeNull();
  });

  // ── onProgress / handleTimeUpdate ────────────────────────────────────────────

  it("onProgress() registers callback invoked by timeupdate when duration is finite", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    const cb = vi.fn();
    p.onProgress(cb);
    const sp = p.speak("hi").catch(() => {});
    await flushMicrotasks();
    FakeAudio.last!.currentTime = 2;
    FakeAudio.last!.duration = 10;
    FakeAudio.last!.dispatchEvent(new Event("timeupdate"));
    expect(cb).toHaveBeenCalledWith(2, 10);
    p.stop();
    await sp;
  });

  it("onProgress callback is not called when duration is NaN", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    const cb = vi.fn();
    p.onProgress(cb);
    const sp = p.speak("hi").catch(() => {});
    await flushMicrotasks();
    FakeAudio.last!.duration = NaN;
    FakeAudio.last!.dispatchEvent(new Event("timeupdate"));
    expect(cb).not.toHaveBeenCalled();
    p.stop();
    await sp;
  });

  it("onProgress(null) clears the callback", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    const cb = vi.fn();
    p.onProgress(cb);
    p.onProgress(null);
    const sp = p.speak("hi").catch(() => {});
    await flushMicrotasks();
    FakeAudio.last!.duration = 10;
    FakeAudio.last!.dispatchEvent(new Event("timeupdate"));
    expect(cb).not.toHaveBeenCalled();
    p.stop();
    await sp;
  });

  // ── dispose ──────────────────────────────────────────────────────────────────

  it("dispose() stops playback and removes event listeners", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    const sp = p.speak("hi").catch(() => {});
    await flushMicrotasks();
    p.dispose();
    await sp;
    expect(p.isSpeaking).toBe(false);
    // After dispose, dispatching events should not throw (listeners removed).
    FakeAudio.last!.dispatchEvent(new Event("ended"));
  });

  // ── throwIfAborted (line 344-345) ─────────────────────────────────────────

  it("throwIfAborted fires when signal is aborted mid-synthesize (stop() called during await)", async () => {
    let resolveBlob!: (b: Blob) => void;
    const synthFn = vi.fn().mockReturnValue(
      new Promise<Blob>((resolve) => {
        resolveBlob = resolve;
      }),
    );
    const p = new KokoroProvider({ synthesize: synthFn });

    const first = p.speak("one"); // synthesize pending
    p.stop(); // abort the signal; abortController set to null
    // Now resolve synthesize — throwIfAborted will throw
    resolveBlob(makeBlob());
    await expect(first).rejects.toMatchObject({ name: "AbortError" });
    expect(p.isSpeaking).toBe(false);
  });

  // ── speakSequence play().catch() branch (lines 187-190) ───────────────────

  it("speakSequence() rejects with Error when play() rejects with Error", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    FakeAudio.last!.play.mockRejectedValueOnce(new Error("play blocked"));
    await expect(p.speakSequence(["a", "b"])).rejects.toThrow("play blocked");
    expect(p.isSpeaking).toBe(false);
  });

  it("speakSequence() wraps non-Error play rejection in an Error", async () => {
    const synthFn = vi.fn().mockResolvedValue(makeBlob());
    const p = new KokoroProvider({ synthesize: synthFn });
    FakeAudio.last!.play.mockRejectedValueOnce("string-rejection");
    await expect(p.speakSequence(["a", "b"])).rejects.toThrow("string-rejection");
    expect(p.isSpeaking).toBe(false);
  });

  // ── default synthesize fallback (line 60) ────────────────────────────────────

  it("uses ttsApi.synthesizeTTS as default when no synthesize option is provided", async () => {
    synthesizeTTSMock.mockResolvedValueOnce(makeBlob());
    const p = new KokoroProvider(); // uses default synthesize
    const sp = p.speak("default path");
    await flushMicrotasks();
    FakeAudio.last!.dispatchEvent(new Event("ended"));
    await sp;
    expect(synthesizeTTSMock).toHaveBeenCalledWith(
      "default path",
      undefined,
      undefined,
      expect.any(AbortSignal),
    );
  });
});
