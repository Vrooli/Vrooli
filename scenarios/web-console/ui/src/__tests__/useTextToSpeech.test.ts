import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";

// Must set up speechSynthesis BEFORE the hook module is imported,
// because the hook evaluates `browserSupported` at module load time.
const mockSynthSpeak = vi.fn();
const mockSynthCancel = vi.fn();
const mockSynthGetVoices = vi.fn().mockReturnValue([]);

Object.defineProperty(window, "speechSynthesis", {
  value: {
    speak: mockSynthSpeak,
    cancel: mockSynthCancel,
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
vi.mock("../lib/api", () => ({
  fetchCapabilitiesLiveness: _mockFetchCaps,
  fetchCapabilitiesLivenessCached: (...args: unknown[]) => _mockFetchCaps(...args) as unknown,
  getTTSVoices: vi.fn(),
  synthesizeTTS: vi.fn(),
  _resetCapabilitiesCache: vi.fn(),
}));

import { fetchCapabilitiesLiveness, getTTSVoices, synthesizeTTS } from "../lib/api";
import { useTextToSpeech, type TTSSettings } from "../hooks/useTextToSpeech";

const mockFetchCaps = fetchCapabilitiesLiveness as ReturnType<typeof vi.fn>;
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
});

afterEach(() => {
  vi.clearAllMocks();
  // Re-setup speechSynthesis mock after clearing (vi.clearAllMocks resets fn implementations)
  mockSynthGetVoices.mockReturnValue([]);
});

describe("useTextToSpeech", () => {
  function unlockBrowserAudio() {
    window.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter" }));
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

    mockSynthSpeak.mockImplementation((u: FakeUtterance) => {
      setTimeout(() => u.onend?.(), 10);
    });

    const { result } = renderHook(() => useTextToSpeech(defaultSettings));

    await waitFor(() => {
      expect(result.current.backend).toBe("browser");
    });

    unlockBrowserAudio();
    act(() => result.current.speak("hello world"));
    expect(result.current.isSpeaking).toBe(true);

    await waitFor(() => {
      expect(result.current.isSpeaking).toBe(false);
    });
  });

  it("speakParagraphs speaks multiple paragraphs sequentially", async () => {
    mockFetchCaps.mockResolvedValue({
      capabilities: [],
      timestamp: new Date().toISOString(),
    });

    mockSynthSpeak.mockImplementation((u: FakeUtterance) => {
      setTimeout(() => u.onend?.(), 5);
    });

    const { result } = renderHook(() => useTextToSpeech(defaultSettings));

    await waitFor(() => {
      expect(result.current.backend).toBe("browser");
    });

    unlockBrowserAudio();
    act(() => result.current.speakParagraphs(["para 1", "para 2"]));
    expect(result.current.isSpeaking).toBe(true);

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
    mockSynthSpeak.mockImplementation((u: FakeUtterance) => {
      setTimeout(() => u.onend?.(), 10);
    });

    const { result } = renderHook(() => useTextToSpeech(defaultSettings));

    await waitFor(() => {
      expect(result.current.backend).toBe("kokoro");
    });

    unlockBrowserAudio();
    act(() => result.current.speak("hello"));

    // The kokoro provider rejects, triggering browser fallback
    await waitFor(() => {
      expect(mockSynthSpeak).toHaveBeenCalled();
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
    mockSynthSpeak.mockImplementation((u: FakeUtterance) => {
      setTimeout(() => u.onerror?.(new Error("Browser speech failed")), 10);
    });

    const { result } = renderHook(() => useTextToSpeech(defaultSettings));

    await waitFor(() => {
      expect(result.current.backend).toBe("kokoro");
    });

    unlockBrowserAudio();
    // Should not throw an unhandled rejection
    act(() => result.current.speak("hello"));

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
    });
    expect(result.current.backendReason).toContain("Kokoro backend was selected explicitly");
  });

  it("speakParagraphs falls back to browser in auto mode when kokoro fails", async () => {
    mockFetchCaps.mockResolvedValue({
      capabilities: [{ id: "kokoro-tts", status: "available" }],
      timestamp: new Date().toISOString(),
    });
    mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
    mockSynthesizeTTS.mockRejectedValue(new Error("Kokoro synthesis failed"));
    mockSynthSpeak.mockImplementation((u: FakeUtterance) => {
      setTimeout(() => u.onend?.(), 10);
    });

    const { result } = renderHook(() => useTextToSpeech(defaultSettings));

    await waitFor(() => {
      expect(result.current.backend).toBe("kokoro");
    });

    unlockBrowserAudio();
    act(() => result.current.speakParagraphs(["para 1", "para 2"]));

    await waitFor(() => {
      expect(mockSynthSpeak).toHaveBeenCalled();
    });
    await waitFor(() => {
      expect(result.current.isSpeaking).toBe(false);
    });
    expect(result.current.backendReason).toContain("Browser handled playback");
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
});
