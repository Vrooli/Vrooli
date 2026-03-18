import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { KokoroProvider } from "../KokoroProvider";

vi.mock("../../../lib/api", () => ({
  synthesizeTTS: vi.fn(),
}));

import { synthesizeTTS } from "../../../lib/api";

const mockSynthesizeTTS = synthesizeTTS as ReturnType<typeof vi.fn>;

/**
 * Minimal mock that simulates an HTMLAudioElement for unit testing.
 *
 * Vitest/jsdom does not provide a working Audio constructor, so we
 * intercept `new Audio()` with this fake that exposes the subset of
 * properties and events the provider relies on.
 */
class FakeHTMLAudioElement extends EventTarget {
  src = "";
  currentTime = 0;
  duration = NaN;
  paused = true;
  playbackRate = 1;
  volume = 1;
  error: MediaError | null = null;

  play = vi.fn(async () => {
    this.paused = false;
    // Simulate loadedmetadata → set a duration
    Object.defineProperty(this, "duration", { value: 5.0, writable: true, configurable: true });
    // Fire ended asynchronously (simulates playback finishing)
    setTimeout(() => this.dispatchEvent(new Event("ended")), 0);
  });
  pause = vi.fn(() => {
    this.paused = true;
  });
  load = vi.fn(() => {
    this.currentTime = 0;
  });
  removeAttribute = vi.fn();

  addEventListener = vi.fn((type: string, handler: EventListenerOrEventListenerObject) => {
    super.addEventListener(type, handler);
  });
  removeEventListener = vi.fn((type: string, handler: EventListenerOrEventListenerObject) => {
    super.removeEventListener(type, handler);
  });
}

let fakeAudio: FakeHTMLAudioElement;

beforeEach(() => {
  fakeAudio = new FakeHTMLAudioElement();
  vi.stubGlobal("Audio", vi.fn(() => fakeAudio));

  // Mock URL.createObjectURL / revokeObjectURL
  vi.stubGlobal("URL", {
    ...globalThis.URL,
    createObjectURL: vi.fn(() => "blob:fake-url"),
    revokeObjectURL: vi.fn(),
  });
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("KokoroProvider", () => {
  it("synthesizes, creates blob URL, and plays via HTMLAudioElement", async () => {
    const blob = new Blob(["audio-data"], { type: "audio/mp3" });
    mockSynthesizeTTS.mockResolvedValue(blob);

    const provider = new KokoroProvider();
    await provider.speak("hello world", { voice: "af_heart", rate: 1.2 });

    expect(mockSynthesizeTTS).toHaveBeenCalledWith(
      "hello world",
      "af_heart",
      1.2,
      expect.any(AbortSignal),
    );

    expect(URL.createObjectURL).toHaveBeenCalledWith(blob);
    expect(fakeAudio.src).toBe("blob:fake-url");
    expect(fakeAudio.play).toHaveBeenCalledTimes(1);
    expect(provider.isSpeaking).toBe(false); // resolved after 'ended'
  });

  it("stop() pauses audio, revokes blob URL, and rejects pending speak", async () => {
    const blob = new Blob(["audio-data"], { type: "audio/mp3" });
    // Never resolve play → keep speak promise pending
    fakeAudio.play = vi.fn(async () => {
      fakeAudio.paused = false;
      Object.defineProperty(fakeAudio, "duration", { value: 10, writable: true, configurable: true });
      // Don't fire 'ended' — playback stays in progress
    });
    mockSynthesizeTTS.mockResolvedValue(blob);

    const provider = new KokoroProvider();
    const speakPromise = provider.speak("test");

    // Wait for play() to be called
    await vi.waitFor(() => {
      expect(fakeAudio.play).toHaveBeenCalledTimes(1);
    });

    provider.stop();

    expect(fakeAudio.pause).toHaveBeenCalled();
    expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:fake-url");
    await expect(speakPromise).rejects.toThrow("The operation was aborted.");
    expect(provider.isSpeaking).toBe(false);
  });

  it("pause() and resume() delegate to audio element", async () => {
    const blob = new Blob(["audio-data"], { type: "audio/mp3" });
    fakeAudio.play = vi.fn(async () => {
      fakeAudio.paused = false;
      Object.defineProperty(fakeAudio, "duration", { value: 10, writable: true, configurable: true });
    });
    mockSynthesizeTTS.mockResolvedValue(blob);

    const provider = new KokoroProvider();
    const speakPromise = provider.speak("test");
    await vi.waitFor(() => expect(fakeAudio.play).toHaveBeenCalledTimes(1));

    provider.pause();
    expect(fakeAudio.pause).toHaveBeenCalled();
    expect(provider.isSpeaking).toBe(true); // still in a speak session

    const state = provider.getPlaybackState();
    expect(state.isPaused).toBe(true);

    fakeAudio.play.mockImplementation(async () => {
      fakeAudio.paused = false;
      // Fire ended to resolve the speak promise
      setTimeout(() => fakeAudio.dispatchEvent(new Event("ended")), 0);
    });
    provider.resume();
    expect(fakeAudio.play).toHaveBeenCalled();

    await speakPromise;
    expect(provider.isSpeaking).toBe(false);
  });

  it("seek() sets audio.currentTime within duration bounds", async () => {
    const blob = new Blob(["audio-data"], { type: "audio/mp3" });
    fakeAudio.play = vi.fn(async () => {
      fakeAudio.paused = false;
      Object.defineProperty(fakeAudio, "duration", { value: 10, writable: true, configurable: true });
    });
    mockSynthesizeTTS.mockResolvedValue(blob);

    const provider = new KokoroProvider();
    const speakPromise = provider.speak("test").catch(() => {});
    await vi.waitFor(() => expect(fakeAudio.play).toHaveBeenCalledTimes(1));

    provider.seek(5);
    expect(fakeAudio.currentTime).toBe(5);

    // Clamp to duration
    provider.seek(999);
    expect(fakeAudio.currentTime).toBe(10);

    // Clamp to 0
    provider.seek(-5);
    expect(fakeAudio.currentTime).toBe(0);

    provider.stop();
    await speakPromise;
  });

  it("setPlaybackRate() and setVolume() update audio element properties", () => {
    const provider = new KokoroProvider();

    provider.setPlaybackRate(1.5);
    expect(fakeAudio.playbackRate).toBe(1.5);

    provider.setVolume(0.3);
    expect(fakeAudio.volume).toBe(0.3);

    // Volume clamped to [0, 1]
    provider.setVolume(2);
    expect(fakeAudio.volume).toBe(1);

    provider.setVolume(-1);
    expect(fakeAudio.volume).toBe(0);
  });

  it("progress callback fires on timeupdate events", async () => {
    const blob = new Blob(["audio-data"], { type: "audio/mp3" });
    fakeAudio.play = vi.fn(async () => {
      fakeAudio.paused = false;
      Object.defineProperty(fakeAudio, "duration", { value: 10, writable: true, configurable: true });
    });
    mockSynthesizeTTS.mockResolvedValue(blob);

    const provider = new KokoroProvider();
    const progressFn = vi.fn();
    provider.onProgress(progressFn);

    const speakPromise = provider.speak("test").catch(() => {});
    await vi.waitFor(() => expect(fakeAudio.play).toHaveBeenCalledTimes(1));

    // Simulate timeupdate
    fakeAudio.currentTime = 3.5;
    fakeAudio.dispatchEvent(new Event("timeupdate"));

    expect(progressFn).toHaveBeenCalledWith(3.5, 10);

    // Unregister
    provider.onProgress(null);
    fakeAudio.currentTime = 5;
    fakeAudio.dispatchEvent(new Event("timeupdate"));
    expect(progressFn).toHaveBeenCalledTimes(1); // not called again

    provider.stop();
    await speakPromise;
  });

  it("handles 0-byte blob gracefully (non-speakable input)", async () => {
    const emptyBlob = new Blob([], { type: "audio/mp3" });
    mockSynthesizeTTS.mockResolvedValue(emptyBlob);

    const provider = new KokoroProvider();
    await provider.speak("---");

    expect(fakeAudio.play).not.toHaveBeenCalled();
    expect(provider.isSpeaking).toBe(false);
  });

  it("capabilities returns all true", () => {
    const provider = new KokoroProvider();
    expect(provider.capabilities).toEqual({
      canPause: true,
      canSeek: true,
      canAdjustSpeed: true,
      canAdjustVolume: true,
    });
  });

  it("dispose() removes event listeners", () => {
    const provider = new KokoroProvider();
    provider.dispose();

    expect(fakeAudio.removeEventListener).toHaveBeenCalledWith("timeupdate", expect.any(Function));
    expect(fakeAudio.removeEventListener).toHaveBeenCalledWith("ended", expect.any(Function));
    expect(fakeAudio.removeEventListener).toHaveBeenCalledWith("error", expect.any(Function));
  });

  it("reports isSpeaking=false initially", () => {
    const provider = new KokoroProvider();
    expect(provider.isSpeaking).toBe(false);
  });

  it("cleans up on fetch error", async () => {
    mockSynthesizeTTS.mockRejectedValue(new Error("Network error"));

    const provider = new KokoroProvider();
    await expect(provider.speak("test")).rejects.toThrow("Network error");

    expect(provider.isSpeaking).toBe(false);
  });

  it("revokes previous blob URL when speak is called again", async () => {
    const blob1 = new Blob(["audio-1"], { type: "audio/mp3" });
    const blob2 = new Blob(["audio-2"], { type: "audio/mp3" });
    mockSynthesizeTTS.mockResolvedValueOnce(blob1).mockResolvedValueOnce(blob2);

    const provider = new KokoroProvider();
    await provider.speak("first");

    // The first blob URL was created; on second speak, stop() revokes it
    await provider.speak("second");

    expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:fake-url");
  });

  describe("speakSequence", () => {
    it("synthesizes all texts and plays concatenated blob as a single track", async () => {
      const blob1 = new Blob(["part1"], { type: "audio/mpeg" });
      const blob2 = new Blob(["part2"], { type: "audio/mpeg" });
      mockSynthesizeTTS
        .mockResolvedValueOnce(blob1)
        .mockResolvedValueOnce(blob2);

      const provider = new KokoroProvider();
      await provider.speakSequence(["hello", "world"], { voice: "af_heart" });

      // Both segments synthesized
      expect(mockSynthesizeTTS).toHaveBeenCalledTimes(2);
      expect(mockSynthesizeTTS).toHaveBeenCalledWith("hello", "af_heart", undefined, expect.any(AbortSignal));
      expect(mockSynthesizeTTS).toHaveBeenCalledWith("world", "af_heart", undefined, expect.any(AbortSignal));

      // Single blob URL created (concatenated), single play() call
      expect(URL.createObjectURL).toHaveBeenCalledTimes(1);
      expect(fakeAudio.play).toHaveBeenCalledTimes(1);
      expect(provider.isSpeaking).toBe(false); // resolved after 'ended'
    });

    it("skips 0-byte segments but still plays the rest", async () => {
      const emptyBlob = new Blob([], { type: "audio/mpeg" });
      const goodBlob = new Blob(["audio"], { type: "audio/mpeg" });
      mockSynthesizeTTS
        .mockResolvedValueOnce(emptyBlob)
        .mockResolvedValueOnce(goodBlob);

      const provider = new KokoroProvider();
      await provider.speakSequence(["---", "real text"]);

      expect(URL.createObjectURL).toHaveBeenCalledTimes(1);
      expect(fakeAudio.play).toHaveBeenCalledTimes(1);
    });

    it("returns early without playing when all segments are empty", async () => {
      const emptyBlob = new Blob([], { type: "audio/mpeg" });
      mockSynthesizeTTS.mockResolvedValue(emptyBlob);

      const provider = new KokoroProvider();
      await provider.speakSequence(["---", "..."]);

      expect(fakeAudio.play).not.toHaveBeenCalled();
      expect(provider.isSpeaking).toBe(false);
    });

    it("delegates to speak() for a single-element array", async () => {
      const blob = new Blob(["audio"], { type: "audio/mpeg" });
      mockSynthesizeTTS.mockResolvedValue(blob);

      const provider = new KokoroProvider();
      const speakSpy = vi.spyOn(provider, "speak");
      await provider.speakSequence(["only one"], { voice: "af_heart" });

      expect(speakSpy).toHaveBeenCalledWith("only one", { voice: "af_heart" });
    });

    it("returns immediately for an empty array", async () => {
      const provider = new KokoroProvider();
      await provider.speakSequence([]);

      expect(mockSynthesizeTTS).not.toHaveBeenCalled();
      expect(fakeAudio.play).not.toHaveBeenCalled();
    });

    it("stop() during synthesis aborts and rejects", async () => {
      let resolveFirst: ((blob: Blob) => void) | undefined;
      mockSynthesizeTTS.mockImplementationOnce(
        () => new Promise<Blob>((resolve) => { resolveFirst = resolve; }),
      );

      const provider = new KokoroProvider();
      const promise = provider.speakSequence(["hello", "world"]);

      // Stop before the first synthesis completes
      provider.stop();

      // Resolve the blocked synthesis — should be ignored due to abort
      resolveFirst?.(new Blob(["audio"], { type: "audio/mpeg" }));

      await expect(promise).rejects.toThrow("The operation was aborted.");
      expect(provider.isSpeaking).toBe(false);
    });
  });
});
