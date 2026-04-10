import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { BrowserTTSProvider } from "../BrowserTTSProvider";

class FakeUtterance {
  text: string;
  rate = 1;
  pitch = 1;
  voice: SpeechSynthesisVoice | null = null;
  onend: (() => void) | null = null;
  onerror: ((e: unknown) => void) | null = null;
  constructor(text: string) {
    this.text = text;
  }
}

const mockSpeak = vi.fn();
const mockCancel = vi.fn();
const mockGetVoices = vi.fn().mockReturnValue([]);
const mockPause = vi.fn();
const mockResume = vi.fn();

beforeEach(() => {
  Object.defineProperty(globalThis, "SpeechSynthesisUtterance", {
    value: FakeUtterance,
    writable: true,
    configurable: true,
  });
  Object.defineProperty(window, "speechSynthesis", {
    value: {
      speak: mockSpeak,
      cancel: mockCancel,
      getVoices: mockGetVoices,
      pause: mockPause,
      resume: mockResume,
    },
    writable: true,
    configurable: true,
  });
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("BrowserTTSProvider", () => {
  it("creates utterance with correct options and calls speechSynthesis.speak", () => {
    const provider = new BrowserTTSProvider();
    provider.speak("hello", { rate: 1.5, pitch: 0.8 });

    expect(mockCancel).toHaveBeenCalled();
    expect(mockSpeak).toHaveBeenCalledTimes(1);

    const utterance = mockSpeak.mock.calls[0]?.[0] as FakeUtterance;
    expect(utterance.text).toBe("hello");
    expect(utterance.rate).toBe(1.5);
    expect(utterance.pitch).toBe(0.8);
  });

  it("resolves when utterance ends", async () => {
    mockSpeak.mockImplementation((u: FakeUtterance) => {
      // Simulate immediate end
      setTimeout(() => u.onend?.(), 0);
    });

    const provider = new BrowserTTSProvider();
    await provider.speak("test");

    expect(provider.isSpeaking).toBe(false);
  });

  it("rejects on utterance error", async () => {
    mockSpeak.mockImplementation((u: FakeUtterance) => {
      setTimeout(() => u.onerror?.({ error: "not-allowed" }), 0);
    });

    const provider = new BrowserTTSProvider();
    await expect(provider.speak("test")).rejects.toThrow("not-allowed");

    expect(provider.isSpeaking).toBe(false);
  });

  it("stop() calls cancel and resets isSpeaking", () => {
    const provider = new BrowserTTSProvider();
    // Manually set speaking state
    provider.speak("test");
    expect(provider.isSpeaking).toBe(true);

    provider.stop();

    expect(mockCancel).toHaveBeenCalled();
    expect(provider.isSpeaking).toBe(false);
  });

  it("dispose() calls stop()", () => {
    const provider = new BrowserTTSProvider();
    const stopSpy = vi.spyOn(provider, "stop");

    provider.dispose();

    expect(stopSpy).toHaveBeenCalled();
  });

  it("sets voice when matching voice found", () => {
    const fakeVoice = { name: "English", lang: "en-US" } as SpeechSynthesisVoice;
    mockGetVoices.mockReturnValue([fakeVoice]);

    const provider = new BrowserTTSProvider();
    provider.speak("hi", { voice: "English" });

    const utterance = mockSpeak.mock.calls[0]?.[0] as FakeUtterance;
    expect(utterance.voice).toBe(fakeVoice);
  });

  it("pause() delegates to speechSynthesis.pause()", () => {
    const provider = new BrowserTTSProvider();
    provider.speak("test");
    expect(provider.isSpeaking).toBe(true);

    provider.pause();
    expect(mockPause).toHaveBeenCalledTimes(1);
    expect(provider.getPlaybackState().isPaused).toBe(true);
    // Still in a speak session
    expect(provider.isSpeaking).toBe(true);
  });

  it("resume() delegates to speechSynthesis.resume()", () => {
    const provider = new BrowserTTSProvider();
    provider.speak("test");
    provider.pause();

    provider.resume();
    expect(mockResume).toHaveBeenCalledTimes(1);
    expect(provider.getPlaybackState().isPaused).toBe(false);
  });

  it("capabilities returns correct shape", () => {
    const provider = new BrowserTTSProvider();
    expect(provider.capabilities).toEqual({
      canPause: true,
      canSeek: false,
      canAdjustSpeed: false,
      canAdjustVolume: false,
    });
  });

  it("getPlaybackState() returns minimal timing data", () => {
    const provider = new BrowserTTSProvider();
    const state = provider.getPlaybackState();
    expect(state.currentTime).toBe(0);
    expect(state.duration).toBeNull();
    expect(state.isPaused).toBe(false);
    expect(state.playbackRate).toBe(1);
    expect(state.volume).toBe(1);
    expect(state.capabilities).toBeDefined();
  });

  it("onProgress is a no-op", () => {
    const provider = new BrowserTTSProvider();
    // Should not throw
    expect(() => provider.onProgress(vi.fn())).not.toThrow();
    expect(() => provider.onProgress(null)).not.toThrow();
  });
});
