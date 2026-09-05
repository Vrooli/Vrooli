// Phase 6 (streaming tail-durability): TTS playback must survive the pane that
// started it being unmounted (workspace warm-set eviction), and a remount must
// re-adopt the same live provider rather than spawning a duplicate. These tests
// drive the real useTextToSpeech adapter + core through jsdom and assert against
// the process-wide playback registry.

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";

const mockSynthSpeak = vi.fn();
const mockSynthCancel = vi.fn();
const mockSynthGetVoices = vi.fn().mockReturnValue([]);
Object.defineProperty(window, "speechSynthesis", {
  value: {
    speak: mockSynthSpeak,
    cancel: mockSynthCancel,
    pause: vi.fn(),
    resume: vi.fn(),
    getVoices: mockSynthGetVoices,
    speaking: false,
    paused: false,
    onvoiceschanged: null,
  },
  writable: true,
  configurable: true,
});

const { _mockFetchCaps } = vi.hoisted(() => ({ _mockFetchCaps: vi.fn() }));
vi.mock("../api/capabilities", () => ({
  fetchCapabilitiesLiveness: _mockFetchCaps,
  fetchCapabilitiesLivenessCached: (...args: unknown[]) => _mockFetchCaps(...args) as unknown,
  _resetCapabilitiesCache: vi.fn(),
}));

const sharedTTSMocks = vi.hoisted(() => ({
  fetchCachedTTS: vi.fn(),
  getTTSVoices: vi.fn(),
  synthesizeTTS: vi.fn(),
  synthesizeTTSWithMetrics: vi.fn(),
}));
vi.mock("../audio-integration", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../audio-integration")>();
  return {
    ...actual,
    fetchCachedTTS: sharedTTSMocks.fetchCachedTTS,
    getTTSVoices: sharedTTSMocks.getTTSVoices,
    synthesizeTTS: sharedTTSMocks.synthesizeTTS,
    synthesizeTTSWithMetrics: sharedTTSMocks.synthesizeTTSWithMetrics,
  };
});
vi.mock("../audio-integration/api/tts", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../audio-integration/api/tts")>();
  return {
    ...actual,
    fetchCachedTTS: sharedTTSMocks.fetchCachedTTS,
    getTTSVoices: sharedTTSMocks.getTTSVoices,
    synthesizeTTS: sharedTTSMocks.synthesizeTTS,
    synthesizeTTSWithMetrics: sharedTTSMocks.synthesizeTTSWithMetrics,
  };
});
vi.mock("../api/ttsHook", () => ({
  recordTTSPlaybackEvent: vi.fn().mockResolvedValue(undefined),
}));

import { fetchCapabilitiesLiveness } from "../api/capabilities";
import { getTTSVoices, synthesizeTTS, ttsPlaybackRegistry } from "../audio-integration";
import { useTextToSpeech, type TTSSettings } from "../hooks/useTextToSpeech";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";

const mockFetchCaps = fetchCapabilitiesLiveness as ReturnType<typeof vi.fn>;
const mockGetVoices = getTTSVoices as ReturnType<typeof vi.fn>;
const mockSynthesizeTTS = synthesizeTTS as ReturnType<typeof vi.fn>;

const settings: TTSSettings = {
  voice: "",
  rate: 1.0,
  pitch: 1.0,
  kokoroVoice: "af_heart",
  kokoroSpeed: 1.0,
  backendPreference: "kokoro",
};

// A controllable HTMLAudioElement stand-in whose play() resolves but does NOT
// auto-end, so the provider stays "speaking" until we choose to end it.
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
    Object.defineProperty(this, "duration", { value: 5, writable: true, configurable: true });
  });
  pause = vi.fn(() => { this.paused = true; });
  load = vi.fn(() => { this.currentTime = 0; });
  removeAttribute = vi.fn();
}

let fakeAudio: FakeAudio;

beforeEach(() => {
  vi.clearAllMocks();
  ttsPlaybackRegistry._resetForTests();
  useWorkspaceStore.setState({ startMutedOnLoad: false });
  mockSynthGetVoices.mockReturnValue([]);
  mockFetchCaps.mockResolvedValue({
    capabilities: [{ id: "audio-tools", status: "available", featureStatus: { "voice-output": "available" } }],
    timestamp: new Date().toISOString(),
  });
  mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
  mockSynthesizeTTS.mockResolvedValue(new Blob(["audio"], { type: "audio/mp3" }));
  sharedTTSMocks.synthesizeTTSWithMetrics.mockResolvedValue({
    blob: new Blob(["audio"], { type: "audio/mp3" }),
    metrics: { requestId: "test-request", synthStartMs: 0, totalChars: 11 },
  });

  fakeAudio = new FakeAudio();
  vi.stubGlobal("Audio", vi.fn(() => fakeAudio));
  (globalThis.URL as unknown as { createObjectURL: unknown }).createObjectURL = vi.fn(() => "blob:fake");
  (globalThis.URL as unknown as { revokeObjectURL: unknown }).revokeObjectURL = vi.fn();
});

afterEach(() => {
  ttsPlaybackRegistry._resetForTests();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

async function mountSpeaking(sessionId: string) {
  const hook = renderHook(() =>
    useTextToSpeech(settings, { source: "terminal_auto", sessionId }),
  );
  await waitFor(() => expect(hook.result.current.backend).toBe("kokoro"));
  await act(async () => {
    void hook.result.current.speakParagraphs(["hello there"]);
    await Promise.resolve();
  });
  await waitFor(() => expect(hook.result.current.isSpeaking).toBe(true));
  return hook;
}

describe("TTS playback persistence across pane unmount (Phase 6)", () => {
  it("keeps a speaking provider alive when its pane unmounts (warm-set eviction)", async () => {
    const hook = await mountSpeaking("s1");
    expect(ttsPlaybackRegistry.has("s1")).toBe(true);
    expect(ttsPlaybackRegistry.isOrphaned("s1")).toBe(false);
    fakeAudio.pause.mockClear();

    // Warm-set eviction: the pane unmounts mid-playback.
    act(() => hook.unmount());

    // Provider is handed off, not disposed — audio was never paused/stopped.
    expect(ttsPlaybackRegistry.has("s1")).toBe(true);
    expect(ttsPlaybackRegistry.isOrphaned("s1")).toBe(true);
    expect(fakeAudio.pause).not.toHaveBeenCalled();
  });

  it("re-adopts the same live provider on remount (single owner, no duplicate)", async () => {
    const first = await mountSpeaking("s1");
    act(() => first.unmount());
    expect(ttsPlaybackRegistry.isOrphaned("s1")).toBe(true);
    fakeAudio.pause.mockClear();

    // User switches back → the pane remounts for the same session.
    const second = renderHook(() =>
      useTextToSpeech(settings, { source: "terminal_auto", sessionId: "s1" }),
    );
    await waitFor(() => expect(second.result.current.backend).toBe("kokoro"));

    // Exactly one owner, re-adopted (not orphaned, not recreated) and the hook
    // reflects that playback is still in progress.
    expect(ttsPlaybackRegistry.size).toBe(1);
    expect(ttsPlaybackRegistry.isOrphaned("s1")).toBe(false);
    await waitFor(() => expect(second.result.current.isSpeaking).toBe(true));
    // A recreate would have replaced+disposed the old provider, pausing audio.
    expect(fakeAudio.pause).not.toHaveBeenCalled();
  });

  it("disposes the provider on unmount when it is idle (no leak)", async () => {
    const hook = renderHook(() =>
      useTextToSpeech(settings, { source: "terminal_auto", sessionId: "s1" }),
    );
    await waitFor(() => expect(hook.result.current.backend).toBe("kokoro"));
    expect(ttsPlaybackRegistry.has("s1")).toBe(true);

    act(() => hook.unmount());
    expect(ttsPlaybackRegistry.has("s1")).toBe(false);
    expect(ttsPlaybackRegistry.size).toBe(0);
  });

  it("explicit stop halts playback and a later idle unmount tears the provider down", async () => {
    const hook = await mountSpeaking("s1");

    act(() => hook.result.current.stop());
    expect(fakeAudio.pause).toHaveBeenCalled();
    await waitFor(() => expect(hook.result.current.isSpeaking).toBe(false));
    // Still installed for reuse while mounted (not orphaned).
    expect(ttsPlaybackRegistry.has("s1")).toBe(true);
    expect(ttsPlaybackRegistry.isOrphaned("s1")).toBe(false);

    act(() => hook.unmount());
    expect(ttsPlaybackRegistry.has("s1")).toBe(false);
  });
});
