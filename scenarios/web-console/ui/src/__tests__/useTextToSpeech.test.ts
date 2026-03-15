import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useTextToSpeech, type SpeechSynthesisAdapter } from "../hooks/useTextToSpeech";

// SpeechSynthesisUtterance is not available in the test environment (jsdom/happy-dom),
// so we provide a minimal stub that the hook's speak() function can construct.
class FakeUtterance {
  text: string;
  rate = 1;
  pitch = 1;
  voice: SpeechSynthesisVoice | null = null;
  onstart: (() => void) | null = null;
  onend: (() => void) | null = null;
  onerror: ((e: unknown) => void) | null = null;
  onpause: (() => void) | null = null;
  onresume: (() => void) | null = null;
  constructor(text: string) { this.text = text; }
}

beforeEach(() => {
  Object.defineProperty(globalThis, "SpeechSynthesisUtterance", {
    value: FakeUtterance,
    writable: true,
    configurable: true,
  });
});
afterEach(() => {
  Object.defineProperty(globalThis, "SpeechSynthesisUtterance", {
    value: undefined,
    writable: true,
    configurable: true,
  });
});

function createFakeAdapter(overrides: Partial<SpeechSynthesisAdapter> = {}): SpeechSynthesisAdapter {
  return {
    getVoices: vi.fn().mockReturnValue([]),
    speak: vi.fn(),
    cancel: vi.fn(),
    pause: vi.fn(),
    resume: vi.fn(),
    speaking: false,
    paused: false,
    onvoiceschanged: null,
    ...overrides,
  };
}

const defaultSettings = { voice: "", rate: 1.0, pitch: 1.0 };

describe("useTextToSpeech", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("reports supported when adapter is available", () => {
    const adapter = createFakeAdapter();
    const { result } = renderHook(() => useTextToSpeech(defaultSettings, adapter));
    expect(result.current.supported).toBe(true);
  });

  it("reports unsupported when adapter is null", () => {
    const { result } = renderHook(() => useTextToSpeech(defaultSettings, null));
    expect(result.current.supported).toBe(false);
  });

  it("speak calls adapter.cancel then adapter.speak", () => {
    const adapter = createFakeAdapter();
    const { result } = renderHook(() => useTextToSpeech(defaultSettings, adapter));

    act(() => result.current.speak("hello"));

    expect(adapter.cancel).toHaveBeenCalledTimes(1);
    expect(adapter.speak).toHaveBeenCalledTimes(1);
    const calls = (adapter.speak as ReturnType<typeof vi.fn>).mock.calls;
    const utterance = (calls[0] as [FakeUtterance])[0];
    expect(utterance.text).toBe("hello");
    expect(utterance.rate).toBe(1.0);
    expect(utterance.pitch).toBe(1.0);
  });

  it("applies custom rate and pitch to utterance", () => {
    const adapter = createFakeAdapter();
    const settings = { voice: "", rate: 1.5, pitch: 0.8 };
    const { result } = renderHook(() => useTextToSpeech(settings, adapter));

    act(() => result.current.speak("test"));

    const calls = (adapter.speak as ReturnType<typeof vi.fn>).mock.calls;
    const utterance = (calls[0] as [FakeUtterance])[0];
    expect(utterance.rate).toBe(1.5);
    expect(utterance.pitch).toBe(0.8);
  });

  it("stop calls adapter.cancel and resets state", () => {
    const adapter = createFakeAdapter();
    const { result } = renderHook(() => useTextToSpeech(defaultSettings, adapter));

    act(() => result.current.stop());

    expect(adapter.cancel).toHaveBeenCalled();
    expect(result.current.isSpeaking).toBe(false);
    expect(result.current.isPaused).toBe(false);
  });

  it("pause and resume delegate to adapter", () => {
    const adapter = createFakeAdapter();
    const { result } = renderHook(() => useTextToSpeech(defaultSettings, adapter));

    act(() => result.current.pause());
    expect(adapter.pause).toHaveBeenCalledTimes(1);

    act(() => result.current.resume());
    expect(adapter.resume).toHaveBeenCalledTimes(1);
  });

  it("loads voices on voiceschanged event", () => {
    const mockVoices = [{ name: "English", lang: "en-US" }] as SpeechSynthesisVoice[];
    let voicesChangedCb: (() => void) | null = null;
    const getVoices = vi.fn().mockReturnValue([]);
    const adapter: SpeechSynthesisAdapter = {
      getVoices,
      speak: vi.fn(),
      cancel: vi.fn(),
      pause: vi.fn(),
      resume: vi.fn(),
      speaking: false,
      paused: false,
      set onvoiceschanged(fn: (() => void) | null) { voicesChangedCb = fn; },
      get onvoiceschanged() { return voicesChangedCb; },
    };

    const { result } = renderHook(() => useTextToSpeech(defaultSettings, adapter));
    expect(result.current.voices).toHaveLength(0);

    // Simulate voices loading
    getVoices.mockReturnValue(mockVoices);
    act(() => voicesChangedCb?.());

    expect(result.current.voices).toHaveLength(1);
    expect(result.current.voices[0]?.name).toBe("English");
  });

  it("cancels speech on unmount", () => {
    const adapter = createFakeAdapter();
    const { unmount } = renderHook(() => useTextToSpeech(defaultSettings, adapter));

    unmount();

    expect(adapter.cancel).toHaveBeenCalled();
  });

  it("is a no-op when adapter unavailable", () => {
    const { result } = renderHook(() => useTextToSpeech(defaultSettings, null));

    // Should not throw
    act(() => result.current.speak("hello"));
    act(() => result.current.stop());
    act(() => result.current.pause());
    act(() => result.current.resume());

    expect(result.current.isSpeaking).toBe(false);
  });
});
