import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";

// Must set up speechSynthesis BEFORE the hook module is imported,
// because the hook evaluates `browserSupported` at module load time.
const mockSynthSpeak = vi.fn();
const mockSynthCancel = vi.fn();
const mockSynthPause = vi.fn();
const mockSynthResume = vi.fn();
const mockSynthGetVoices = vi.fn().mockReturnValue([]);

Object.defineProperty(window, "speechSynthesis", {
  value: {
    speak: mockSynthSpeak,
    cancel: mockSynthCancel,
    pause: mockSynthPause,
    resume: mockSynthResume,
    getVoices: mockSynthGetVoices,
    speaking: false,
    paused: false,
    onvoiceschanged: null,
  },
  writable: true,
  configurable: true,
});

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

Object.defineProperty(globalThis, "SpeechSynthesisUtterance", {
  value: FakeUtterance,
  writable: true,
  configurable: true,
});

// Mock api module — include the cached wrapper the hook actually imports
const { _mockFetchCaps } = vi.hoisted(() => ({
  _mockFetchCaps: vi.fn(),
}));
vi.mock("../api/capabilities", () => ({
  fetchCapabilitiesLiveness: _mockFetchCaps,
  fetchCapabilitiesLivenessCached: (...args: unknown[]) => _mockFetchCaps(...args) as unknown,
  _resetCapabilitiesCache: vi.fn(),
}));
vi.mock("../api/tts", () => ({
  fetchCachedTTS: vi.fn(),
  getTTSVoices: vi.fn(),
  synthesizeTTS: vi.fn(),
  reportTTSEvent: vi.fn(),
}));

import { fetchCapabilitiesLiveness } from "../api/capabilities";
import { fetchCachedTTS, getTTSVoices, synthesizeTTS } from "../api/tts";
import { useTextToSpeech, type TTSSettings } from "../hooks/useTextToSpeech";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";

const mockFetchCaps = fetchCapabilitiesLiveness as ReturnType<typeof vi.fn>;
const mockFetchCachedTTS = fetchCachedTTS as ReturnType<typeof vi.fn>;
const mockGetVoices = getTTSVoices as ReturnType<typeof vi.fn>;
const mockSynthesizeTTS = synthesizeTTS as ReturnType<typeof vi.fn>;

const defaultSettings: TTSSettings = {
  voice: "",
  rate: 1.0,
  pitch: 1.0,
  kokoroVoice: "af_heart",
  kokoroSpeed: 1.0,
  backendPreference: "auto",
};

beforeEach(() => {
  vi.clearAllMocks();
  useWorkspaceStore.setState({ startMutedOnLoad: true });
});

afterEach(() => {
  vi.clearAllMocks();
  vi.useRealTimers();
  // Re-setup speechSynthesis mock after clearing (vi.clearAllMocks resets fn implementations)
  mockSynthGetVoices.mockReturnValue([]);
});

describe("useTextToSpeech", () => {
  function unlockBrowserAudio() {
    act(() => {
      window.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter" }));
    });
  }

  it("selects kokoro backend when capability is available", async () => {
    mockFetchCaps.mockResolvedValue({
      capabilities: [{ id: "kokoro-tts", status: "available" }],
      timestamp: new Date().toISOString(),
    });
    mockGetVoices.mockResolvedValue([
      { id: "af_heart", name: "af_heart" },
      { id: "bf_emma", name: "bf_emma" },
    ]);

    const { result } = renderHook(() => useTextToSpeech(defaultSettings));

    await waitFor(() => {
      expect(result.current.backend).toBe("kokoro");
    });
    expect(result.current.supported).toBe(true);
    expect(result.current.voices).toHaveLength(2);
  });

  it("falls back to browser when kokoro is unavailable", async () => {
    mockFetchCaps.mockResolvedValue({
      capabilities: [{ id: "kokoro-tts", status: "unavailable" }],
      timestamp: new Date().toISOString(),
    });

    const { result } = renderHook(() => useTextToSpeech(defaultSettings));

    await waitFor(() => {
      expect(result.current.backend).toBe("browser");
    });
    expect(result.current.supported).toBe(true);
  });

  it("falls back to browser when capabilities fetch fails", async () => {
    mockFetchCaps.mockRejectedValue(new Error("Network error"));

    const { result } = renderHook(() => useTextToSpeech(defaultSettings));

    await waitFor(() => {
      expect(result.current.backend).toBe("browser");
    });
    expect(result.current.supported).toBe(true);
  });

  it("uses browser directly when backendPreference is browser", async () => {
    const settings = { ...defaultSettings, backendPreference: "browser" as const };
    const { result } = renderHook(() => useTextToSpeech(settings));

    await waitFor(() => {
      expect(result.current.backend).toBe("browser");
    });
    // Should not have called capabilities check
    expect(mockFetchCaps).not.toHaveBeenCalled();
  });

  it("stop resets isSpeaking state", async () => {
    mockFetchCaps.mockResolvedValue({
      capabilities: [],
      timestamp: new Date().toISOString(),
    });

    const { result } = renderHook(() => useTextToSpeech(defaultSettings));

    await waitFor(() => {
      expect(result.current.backend).toBe("browser");
    });

    act(() => result.current.stop());
    expect(result.current.isSpeaking).toBe(false);
  });

  it("speak triggers provider and sets isSpeaking", async () => {
    mockFetchCaps.mockResolvedValue({
      capabilities: [],
      timestamp: new Date().toISOString(),
    });

    const { result } = renderHook(() => useTextToSpeech(defaultSettings));

    await waitFor(() => {
      expect(result.current.backend).toBe("browser");
    });
    const browserUtterances: FakeUtterance[] = [];
    mockSynthSpeak.mockImplementation((u: FakeUtterance) => {
      browserUtterances.push(u);
    });

    unlockBrowserAudio();
    act(() => result.current.speak("hello world"));
    expect(result.current.isSpeaking).toBe(true);

    await waitFor(() => {
      expect(browserUtterances).toHaveLength(1);
    });
    await act(async () => {
      browserUtterances.shift()?.onend?.();
      await Promise.resolve();
    });

    await waitFor(() => {
      expect(result.current.isSpeaking).toBe(false);
    });
  });

  it("speakParagraphs speaks multiple paragraphs sequentially", async () => {
    mockFetchCaps.mockResolvedValue({
      capabilities: [],
      timestamp: new Date().toISOString(),
    });

    const { result } = renderHook(() => useTextToSpeech(defaultSettings));

    await waitFor(() => {
      expect(result.current.backend).toBe("browser");
    });
    const browserUtterances: FakeUtterance[] = [];
    mockSynthSpeak.mockImplementation((u: FakeUtterance) => {
      browserUtterances.push(u);
    });

    unlockBrowserAudio();
    let playback!: Promise<unknown>;
    await act(async () => {
      playback = result.current.speakParagraphs(["para 1", "para 2"]);
      await Promise.resolve();
    });
    await waitFor(() => {
      expect(browserUtterances).toHaveLength(1);
    });
    await act(async () => {
      browserUtterances.shift()?.onend?.();
      await Promise.resolve();
    });
    await waitFor(() => {
      expect(browserUtterances).toHaveLength(1);
    });
    await act(async () => {
      browserUtterances.shift()?.onend?.();
      await Promise.resolve();
    });
    await act(async () => {
      await playback;
    });

    await waitFor(() => {
      expect(result.current.isSpeaking).toBe(false);
    });

    // Both paragraphs should have been spoken (cancel + speak for each)
    expect(mockSynthSpeak).toHaveBeenCalledTimes(2);
  });

  it("speak() falls back to browser when kokoro synthesis fails at runtime", async () => {
    mockFetchCaps.mockResolvedValue({
      capabilities: [{ id: "kokoro-tts", status: "available" }],
      timestamp: new Date().toISOString(),
    });
    mockGetVoices.mockResolvedValue([
      { id: "af_heart", name: "af_heart" },
    ]);
    mockSynthesizeTTS.mockRejectedValue(new Error("Kokoro synthesis failed"));

    // Browser fallback should complete via onend
    const { result } = renderHook(() => useTextToSpeech(defaultSettings));

    await waitFor(() => {
      expect(result.current.backend).toBe("kokoro");
    });
    const browserUtterances: FakeUtterance[] = [];
    mockSynthSpeak.mockImplementation((u: FakeUtterance) => {
      browserUtterances.push(u);
    });

    unlockBrowserAudio();
    act(() => result.current.speak("hello"));

    // The kokoro provider rejects, triggering browser fallback
    await waitFor(() => {
      expect(mockSynthSpeak).toHaveBeenCalled();
    });
    await waitFor(() => {
      expect(browserUtterances).toHaveLength(1);
    });

    await act(async () => {
      browserUtterances.shift()?.onend?.();
      await Promise.resolve();
    });

    // isSpeaking resets after the error handler runs (kokoro path sets it false on rejection)
    await waitFor(() => {
      expect(result.current.isSpeaking).toBe(false);
    });
  });

  it("runtime fallback does not crash when both backends fail", async () => {
    mockFetchCaps.mockResolvedValue({
      capabilities: [{ id: "kokoro-tts", status: "available" }],
      timestamp: new Date().toISOString(),
    });
    mockGetVoices.mockResolvedValue([
      { id: "af_heart", name: "af_heart" },
    ]);
    mockSynthesizeTTS.mockRejectedValue(new Error("Kokoro synthesis failed"));

    // Browser fallback also fails via onerror
    const { result } = renderHook(() => useTextToSpeech(defaultSettings));

    await waitFor(() => {
      expect(result.current.backend).toBe("kokoro");
    });
    const browserUtterances: FakeUtterance[] = [];
    mockSynthSpeak.mockImplementation((u: FakeUtterance) => {
      browserUtterances.push(u);
    });

    unlockBrowserAudio();
    // Should not throw an unhandled rejection
    act(() => result.current.speak("hello"));

    await waitFor(() => {
      expect(mockSynthSpeak).toHaveBeenCalled();
    });
    await waitFor(() => {
      expect(browserUtterances).toHaveLength(1);
    });

    await act(async () => {
      browserUtterances.shift()?.onerror?.(new Error("Browser speech failed"));
      await Promise.resolve();
    });

    await waitFor(() => {
      expect(result.current.isSpeaking).toBe(false);
    });

    // Verify both backends were attempted
    expect(mockSynthesizeTTS).toHaveBeenCalled();
    expect(mockSynthSpeak).toHaveBeenCalled();
  });

  it("defaults kokoro voices to af_heart when voice fetch fails", async () => {
    mockFetchCaps.mockResolvedValue({
      capabilities: [{ id: "kokoro-tts", status: "available" }],
      timestamp: new Date().toISOString(),
    });
    mockGetVoices.mockRejectedValue(new Error("Failed"));

    const { result } = renderHook(() => useTextToSpeech(defaultSettings));

    await waitFor(() => {
      expect(result.current.backend).toBe("kokoro");
    });
    expect(result.current.voices).toEqual([{ id: "af_heart", name: "af_heart" }]);
  });

  it("uses strict kokoro mode without silently falling back to browser", async () => {
    mockFetchCaps.mockResolvedValue({
      capabilities: [{ id: "kokoro-tts", status: "unavailable" }],
      timestamp: new Date().toISOString(),
    });

    const { result } = renderHook(() => useTextToSpeech({
      ...defaultSettings,
      backendPreference: "kokoro",
    }));

    await waitFor(() => {
      expect(result.current.backend).toBe("none");
      expect(result.current.backendReason).toContain("Kokoro backend was selected explicitly");
    });
  });

  it("speakParagraphs falls back to browser in auto mode when kokoro fails", async () => {
    mockFetchCaps.mockResolvedValue({
      capabilities: [{ id: "kokoro-tts", status: "available" }],
      timestamp: new Date().toISOString(),
    });
    mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
    mockSynthesizeTTS.mockRejectedValue(new Error("Kokoro synthesis failed"));
    const { result } = renderHook(() => useTextToSpeech(defaultSettings));

    await waitFor(() => {
      expect(result.current.backend).toBe("kokoro");
    });
    const browserUtterances: FakeUtterance[] = [];
    mockSynthSpeak.mockImplementation((u: FakeUtterance) => {
      browserUtterances.push(u);
    });

    unlockBrowserAudio();
    let playback!: Promise<unknown>;
    await act(async () => {
      playback = result.current.speakParagraphs(["para 1", "para 2"]);
      await Promise.resolve();
    });
    await waitFor(() => {
      expect(browserUtterances).toHaveLength(1);
    });
    await act(async () => {
      browserUtterances.shift()?.onend?.();
      await Promise.resolve();
    });
    await act(async () => {
      await playback;
    });

    await waitFor(() => {
      expect(mockSynthSpeak).toHaveBeenCalled();
    });
    await waitFor(() => {
      expect(result.current.isSpeaking).toBe(false);
    });
    expect(result.current.backendReason).toContain("Browser handled playback");
  });

  it("pause and resume control the active fallback browser provider after runtime kokoro failure", async () => {
    mockFetchCaps.mockResolvedValue({
      capabilities: [{ id: "kokoro-tts", status: "available" }],
      timestamp: new Date().toISOString(),
    });
    mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
    mockSynthesizeTTS.mockRejectedValue(new Error("Kokoro synthesis failed"));

    const { result } = renderHook(() => useTextToSpeech(defaultSettings));

    await waitFor(() => {
      expect(result.current.backend).toBe("kokoro");
    });

    const browserUtterances: FakeUtterance[] = [];
    mockSynthSpeak.mockImplementation((u: FakeUtterance) => {
      browserUtterances.push(u);
    });

    unlockBrowserAudio();
    act(() => result.current.speak("fallback speech"));

    await waitFor(() => {
      expect(browserUtterances).toHaveLength(1);
    });

    act(() => result.current.pause());
    expect(mockSynthPause).toHaveBeenCalledTimes(1);
    expect(result.current.isPaused).toBe(true);

    act(() => result.current.resume());
    expect(mockSynthResume).toHaveBeenCalledTimes(1);
    expect(result.current.isPaused).toBe(false);

    await act(async () => {
      browserUtterances.shift()?.onend?.();
      await Promise.resolve();
    });

    await waitFor(() => {
      expect(result.current.isSpeaking).toBe(false);
    });
  });

  it("playback state snapshots come from the active fallback browser provider", async () => {
    mockFetchCaps.mockResolvedValue({
      capabilities: [{ id: "kokoro-tts", status: "available" }],
      timestamp: new Date().toISOString(),
    });
    mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
    mockSynthesizeTTS.mockRejectedValue(new Error("Kokoro synthesis failed"));

    const { result } = renderHook(() => useTextToSpeech(defaultSettings));

    await waitFor(() => {
      expect(result.current.backend).toBe("kokoro");
    });

    const browserUtterances: FakeUtterance[] = [];
    mockSynthSpeak.mockImplementation((u: FakeUtterance) => {
      browserUtterances.push(u);
    });

    unlockBrowserAudio();
    act(() => result.current.speak("fallback snapshot"));

    await waitFor(() => {
      expect(browserUtterances).toHaveLength(1);
    });

    let playback = result.current.getPlaybackState();
    expect(playback?.capabilities.canPause).toBe(true);
    expect(playback?.capabilities.canSeek).toBe(false);

    act(() => result.current.pause());
    playback = result.current.getPlaybackState();
    expect(playback?.isPaused).toBe(true);

    await act(async () => {
      browserUtterances.shift()?.onend?.();
      await Promise.resolve();
    });
  });

  it("reports browser audio readiness after user interaction", async () => {
    mockFetchCaps.mockResolvedValue({
      capabilities: [],
      timestamp: new Date().toISOString(),
    });

    const { result } = renderHook(() => useTextToSpeech(defaultSettings));

    await waitFor(() => {
      expect(result.current.backend).toBe("browser");
    });
    expect(result.current.browserAudioReady).toBe(false);

    unlockBrowserAudio();

    await waitFor(() => {
      expect(result.current.browserAudioReady).toBe(true);
    });
  });

  it("tracks the start-muted preference before the user explicitly changes mute", async () => {
    useWorkspaceStore.setState({ startMutedOnLoad: false });
    mockFetchCaps.mockResolvedValue({
      capabilities: [],
      timestamp: new Date().toISOString(),
    });

    const { result } = renderHook(() => useTextToSpeech(defaultSettings));

    await waitFor(() => {
      expect(result.current.backend).toBe("browser");
    });
    expect(result.current.isMuted).toBe(false);

    act(() => {
      useWorkspaceStore.setState({ startMutedOnLoad: true });
    });

    await waitFor(() => {
      expect(result.current.isMuted).toBe(true);
    });
  });

  describe("autoplay unlock", () => {
    // Minimal HTMLAudioElement stand-in used when we need to drive the Kokoro
    // path through jsdom. Only the handful of properties KokoroProvider calls
    // are implemented; the rest are no-ops that mirror real element behavior.
    class FakeAudio extends EventTarget {
      src = "";
      currentTime = 0;
      duration = NaN;
      paused = true;
      playbackRate = 1;
      volume = 1;
      muted = false;
      play = vi.fn(async () => {
        this.paused = false;
        Object.defineProperty(this, "duration", { value: 1, writable: true, configurable: true });
        setTimeout(() => this.dispatchEvent(new Event("ended")), 0);
      });
      pause = vi.fn(() => { this.paused = true; });
      load = vi.fn(() => { this.currentTime = 0; });
      removeAttribute = vi.fn();
    }

    let fakeAudio: FakeAudio;
    let createObjectURLSpy: ReturnType<typeof vi.fn>;
    let revokeObjectURLSpy: ReturnType<typeof vi.fn>;
    beforeEach(() => {
      fakeAudio = new FakeAudio();
      vi.stubGlobal("Audio", vi.fn(() => fakeAudio));
      createObjectURLSpy = vi.fn(() => "blob:fake-url");
      revokeObjectURLSpy = vi.fn();
      (globalThis.URL as unknown as { createObjectURL: unknown }).createObjectURL = createObjectURLSpy;
      (globalThis.URL as unknown as { revokeObjectURL: unknown }).revokeObjectURL = revokeObjectURLSpy;
    });
    afterEach(() => {
      vi.unstubAllGlobals();
    });

    it("needsUnlock starts false", async () => {
      mockFetchCaps.mockResolvedValue({
        capabilities: [{ id: "kokoro-tts", status: "available" }],
        timestamp: new Date().toISOString(),
      });
      mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);

      const { result } = renderHook(() => useTextToSpeech(defaultSettings));
      await waitFor(() => expect(result.current.backend).toBe("kokoro"));
      expect(result.current.needsUnlock).toBe(false);
    });

    it("speakParagraphs sets needsUnlock=true (not error) on NotAllowedError", async () => {
      mockFetchCaps.mockResolvedValue({
        capabilities: [{ id: "kokoro-tts", status: "available" }],
        timestamp: new Date().toISOString(),
      });
      mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
      const blob = new Blob(["audio"], { type: "audio/mp3" });
      mockSynthesizeTTS.mockResolvedValue(blob);
      fakeAudio.play = vi.fn().mockRejectedValue(
        Object.assign(new Error("not allowed by the user agent"), { name: "NotAllowedError" }),
      );

      const { result } = renderHook(() => useTextToSpeech({
        ...defaultSettings,
        backendPreference: "kokoro",
      }));
      await waitFor(() => expect(result.current.backend).toBe("kokoro"));

      await act(async () => {
        await result.current.speakParagraphs(["paragraph one"]);
      });

      await waitFor(() => expect(result.current.needsUnlock).toBe(true));
      expect(result.current.error).toBeNull();
    });

    it("generic non-autoplay failure sets error, not needsUnlock", async () => {
      mockFetchCaps.mockResolvedValue({
        capabilities: [{ id: "kokoro-tts", status: "available" }],
        timestamp: new Date().toISOString(),
      });
      mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
      mockSynthesizeTTS.mockRejectedValue(new Error("Kokoro synthesis failed"));

      const { result } = renderHook(() => useTextToSpeech({
        ...defaultSettings,
        backendPreference: "kokoro",
      }));
      await waitFor(() => expect(result.current.backend).toBe("kokoro"));

      await act(async () => {
        try {
          await result.current.speakParagraphs(["paragraph one"]);
        } catch {
          // expected
        }
      });

      await waitFor(() => expect(result.current.error).toBe("Kokoro synthesis failed"));
      expect(result.current.needsUnlock).toBe(false);
    });

    it("unlockAudio() calls provider.unlock() and clears needsUnlock on success", async () => {
      mockFetchCaps.mockResolvedValue({
        capabilities: [{ id: "kokoro-tts", status: "available" }],
        timestamp: new Date().toISOString(),
      });
      mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
      const blob = new Blob(["audio"], { type: "audio/mp3" });
      mockSynthesizeTTS.mockResolvedValue(blob);
      fakeAudio.play = vi.fn().mockRejectedValue(
        Object.assign(new Error("not allowed"), { name: "NotAllowedError" }),
      );

      const { result } = renderHook(() => useTextToSpeech({
        ...defaultSettings,
        backendPreference: "kokoro",
      }));
      await waitFor(() => expect(result.current.backend).toBe("kokoro"));
      await act(async () => { await result.current.speakParagraphs(["x"]); });
      await waitFor(() => expect(result.current.needsUnlock).toBe(true));

      // Reconfigure play to succeed so unlock() resolves true.
      fakeAudio.play = vi.fn(async () => { fakeAudio.paused = false; });

      let unlockResult: boolean | undefined;
      await act(async () => { unlockResult = await result.current.unlockAudio(); });

      expect(unlockResult).toBe(true);
      expect(result.current.needsUnlock).toBe(false);
      expect(result.current.browserAudioReady).toBe(true);
    });

    it("dismissNeedsUnlock clears the flag without touching the provider", async () => {
      mockFetchCaps.mockResolvedValue({
        capabilities: [{ id: "kokoro-tts", status: "available" }],
        timestamp: new Date().toISOString(),
      });
      mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
      const blob = new Blob(["audio"], { type: "audio/mp3" });
      mockSynthesizeTTS.mockResolvedValue(blob);
      fakeAudio.play = vi.fn().mockRejectedValue(
        Object.assign(new Error("not allowed"), { name: "NotAllowedError" }),
      );

      const { result } = renderHook(() => useTextToSpeech({
        ...defaultSettings,
        backendPreference: "kokoro",
      }));
      await waitFor(() => expect(result.current.backend).toBe("kokoro"));
      await act(async () => { await result.current.speakParagraphs(["x"]); });
      await waitFor(() => expect(result.current.needsUnlock).toBe(true));

      act(() => result.current.dismissNeedsUnlock());
      expect(result.current.needsUnlock).toBe(false);
    });

    it("pause controls cached Kokoro blob playback from the active provider", async () => {
      mockFetchCaps.mockResolvedValue({
        capabilities: [{ id: "kokoro-tts", status: "available" }],
        timestamp: new Date().toISOString(),
      });
      mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
      mockFetchCachedTTS.mockResolvedValue(new Blob(["audio"], { type: "audio/mp3" }));

      const { result } = renderHook(() => useTextToSpeech({
        ...defaultSettings,
        backendPreference: "kokoro",
      }));
      await waitFor(() => expect(result.current.backend).toBe("kokoro"));

      let playback!: Promise<unknown>;
      await act(async () => {
        playback = result.current.speakParagraphs(["cached paragraph"], {
          eventId: "evt-cached",
          version: "active",
        });
        await Promise.resolve();
      });

      act(() => result.current.pause());
      expect(fakeAudio.pause).toHaveBeenCalled();
      expect(result.current.isPaused).toBe(true);

      await act(async () => {
        fakeAudio.dispatchEvent(new Event("ended"));
        await playback;
      });
    });
  });
});
