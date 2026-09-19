// ── Audio Cue Contract Regression Tests ──
//
// These tests enforce the contract that audio cues (start/stop chimes) play
// ONLY during recording sessions — never during mic pre-warm, visibility
// lifecycle, cleanup/dispose, error recovery, or transcription cancellation.
//
// Each test is named to make the contract explicit. Future agents modifying
// voice code should run these tests to verify cue behavior isn't broken.
//
// DOC: docs/internal/VOICE-LATENCY.md#audio-cue-contract

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { apiBaseMock } from "../../test-utils";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";

vi.mock("@vrooli/api-base", () => apiBaseMock());

// Capabilities surface is consumed by the web-console useVoiceInput adapter
// (via the audio-integration core's capabilityCheck hook). Mock the four
// entry points the adapter calls so tests don't hit the real Connect-RPC
// transport. Individual tests can override `mockCapabilities()` to set the
// resolved value via `globalThis.fetch` — that mock is honored too because
// fetchCapabilities falls back to it when the synchronous snapshot is null.
const fetchCapabilitiesMock = vi.fn().mockResolvedValue({
  capabilities: [{ id: "audio-tools", status: "unavailable", features: [] }],
  timestamp: new Date().toISOString(),
});
const refreshCapabilitiesLivenessMock = vi.fn().mockResolvedValue(undefined);
const getCapabilitiesLivenessSnapshotMock = vi.fn(() => null);
vi.mock("../../api/capabilities", () => ({
  fetchCapabilities: fetchCapabilitiesMock,
  fetchCapabilitiesLiveness: fetchCapabilitiesMock,
  fetchCapabilitiesLivenessCached: fetchCapabilitiesMock,
  refreshCapabilitiesLiveness: refreshCapabilitiesLivenessMock,
  getCapabilitiesLivenessSnapshot: getCapabilitiesLivenessSnapshotMock,
  _resetCapabilitiesCache: vi.fn(),
}));

// Mock the audio cues module so we can track calls without actual audio.
// useVoiceCore (inside audio-integration) imports cues via "../index" — the
// barrel mock does not intercept those internal imports, so we also mock the
// underlying audioCues / sharedAudioContext / vad source
// files directly. Spy instances are hoisted via vi.hoisted so they're safe
// to reference inside the lifted vi.mock factories.
const audioHoisted = vi.hoisted(() => ({
  startCue: vi.fn(),
  stopCue: vi.fn(),
  createAudioContextStub: () => ({
    state: "running",
    createMediaStreamSource: () => ({ connect: () => {}, disconnect: () => {} }),
    createBiquadFilter: () => ({ type: "lowpass", frequency: { value: 0 }, Q: { value: 0 }, connect: () => {}, disconnect: () => {} }),
    createAnalyser: () => ({ fftSize: 0, frequencyBinCount: 64, getByteFrequencyData: () => {}, getByteTimeDomainData: () => {}, connect: () => {}, disconnect: () => {} }),
    createGain: () => ({ gain: { value: 1 }, connect: () => {}, disconnect: () => {} }),
    destination: { connect: () => {}, disconnect: () => {} },
    resume: () => Promise.resolve(),
  }),
}));
const startCueSpy = audioHoisted.startCue;
const stopCueSpy = audioHoisted.stopCue;
vi.mock("../../audio-integration/hooks/voice/audioCues", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../audio-integration/hooks/voice/audioCues")>();
  return {
    ...actual,
    playRecordingStartCue: audioHoisted.startCue,
    playRecordingStopCue: audioHoisted.stopCue,
  };
});
vi.mock("../../audio-integration/hooks/voice/sharedAudioContext", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../audio-integration/hooks/voice/sharedAudioContext")>();
  return {
    ...actual,
    getSharedAudioContext: () => audioHoisted.createAudioContextStub(),
    closeSharedAudioContext: vi.fn(),
  };
});
vi.mock("../../audio-integration/hooks/voice/vad", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../audio-integration/hooks/voice/vad")>();
  return {
    ...actual,
    createVadRefs: () => ({ state: "idle", silenceThreshold: 0, speechThreshold: 0, cachedFloorBaseline: null }),
    createVadRefsFromCache: vi.fn(),
    extractCacheableFloor: vi.fn().mockReturnValue({ silenceThreshold: 0.01, speechThreshold: 0.02, timestamp: Date.now() }),
    saveNoiseFloorCache: vi.fn(),
    loadNoiseFloorCache: vi.fn().mockReturnValue(null),
    vadTick: vi.fn().mockReturnValue(null),
    VAD_FLOOR_CACHE_MAX_AGE_MS: 86400000,
  };
});
vi.mock("../../audio-integration/api/voice", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../audio-integration/api/voice")>();
  return {
    ...actual,
    getVoiceStreamConfig: vi.fn().mockResolvedValue({
      flushIntervalMs: 0, minDeltaBytes: 0, overlapBytes: 0,
      persistentMode: false, wakeWordEnabled: false, wakeWordThreshold: 0, segmentSilenceMs: 0,
    }),
    getWakeWordConfig: vi.fn().mockResolvedValue({ configured: false, template: null }),
    transcribeAudioBypassFilter: vi.fn().mockResolvedValue(""),
    transcribeAudio: vi.fn().mockResolvedValue(""),
    transcribeAudioWithRetry: vi.fn().mockResolvedValue(""),
    buildVoiceStreamWsUrl: vi.fn().mockReturnValue("ws://localhost:0"),
  };
});
vi.mock("../../audio-integration/hooks/voice/wakeword", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../audio-integration/hooks/voice/wakeword")>();
  return {
    ...actual,
    createWakeWordEngine: vi.fn(),
    PassiveListener: vi.fn(),
  };
});

// Single consolidated mock for the audio-integration surface. vi.mock
// overwrites duplicates so the previous one-mock-per-concern split (audio
// cues, shared audio context, mic readiness, VAD, wake-word) is fused here.
vi.mock("../../audio-integration", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../audio-integration")>();
  return {
    ...actual,
    playRecordingStartCue: audioHoisted.startCue,
    playRecordingStopCue: audioHoisted.stopCue,
    getSharedAudioContext: () => ({
      state: "running",
      createMediaStreamSource: () => ({ connect: vi.fn(), disconnect: vi.fn() }),
      createBiquadFilter: () => ({ type: "lowpass", frequency: { value: 0 }, Q: { value: 0 }, connect: vi.fn(), disconnect: vi.fn() }),
      createMediaStreamDestination: () => ({ stream: mockStream(), connect: vi.fn(), disconnect: vi.fn() }),
      createAnalyser: () => ({ fftSize: 0, frequencyBinCount: 64, getByteFrequencyData: vi.fn(), getByteTimeDomainData: vi.fn(), connect: vi.fn(), disconnect: vi.fn() }),
      createGain: () => ({ gain: { value: 1 }, connect: vi.fn(), disconnect: vi.fn() }),
      destination: { connect: vi.fn(), disconnect: vi.fn() },
      resume: vi.fn().mockResolvedValue(undefined),
    }),
    closeSharedAudioContext: vi.fn(),
    createVadRefs: () => ({ state: "idle", silenceThreshold: 0, speechThreshold: 0, cachedFloorBaseline: null }),
    createVadRefsFromCache: vi.fn(),
    extractCacheableFloor: vi.fn().mockReturnValue({ silenceThreshold: 0.01, speechThreshold: 0.02, timestamp: Date.now() }),
    saveNoiseFloorCache: vi.fn(),
    loadNoiseFloorCache: vi.fn().mockReturnValue(null),
    vadTick: vi.fn().mockReturnValue(null),
    VAD_FLOOR_CACHE_MAX_AGE_MS: 86400000,
    createWakeWordEngine: vi.fn(),
    PassiveListener: vi.fn(),
    // API surface — useVoiceInput calls these at mount via the embed lazy
    // client; intercept so tests don't hit the network.
    getVoiceStreamConfig: vi.fn().mockResolvedValue({
      flushIntervalMs: 0, minDeltaBytes: 0, overlapBytes: 0,
      persistentMode: false, wakeWordEnabled: false, wakeWordThreshold: 0, segmentSilenceMs: 0,
    }),
    getWakeWordConfig: vi.fn().mockResolvedValue({ configured: false, template: null }),
    transcribeAudioBypassFilter: vi.fn().mockResolvedValue(""),
    transcribeAudio: vi.fn().mockResolvedValue(""),
    transcribeAudioWithRetry: vi.fn().mockResolvedValue(""),
    buildVoiceStreamWsUrl: vi.fn().mockReturnValue("ws://localhost:0"),
  };
});

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function mockStream(): MediaStream {
  const track = {
    stop: vi.fn(),
    readyState: "live" as MediaStreamTrack["readyState"],
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  };
  return {
    getTracks: () => [track],
    getAudioTracks: () => [track],
  } as unknown as MediaStream;
}

function mockCapabilities(whisperAvailable: boolean, streaming = false) {
  const features = whisperAvailable
    ? (streaming ? ["voice-input", "voice-streaming"] : ["voice-input"])
    : [];
  const resp = {
    capabilities: [
      {
        id: "audio-tools",
        status: whisperAvailable ? "available" : "unavailable",
        features,
      },
    ],
    timestamp: new Date().toISOString(),
  };
  fetchCapabilitiesMock.mockResolvedValue(resp);
  refreshCapabilitiesLivenessMock.mockResolvedValue(resp);
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve(resp),
  }) as typeof fetch;
}

function mockMediaDevices() {
  Object.defineProperty(navigator, "mediaDevices", {
    value: {
      getUserMedia: vi.fn().mockResolvedValue(mockStream()),
    },
    configurable: true,
  });
}

async function settle() {
  await new Promise((r) => setTimeout(r, 50));
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("Audio Cue Contract", () => {
  let originalFetch: typeof globalThis.fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    vi.clearAllMocks();
    startCueSpy.mockClear();
    stopCueSpy.mockClear();
    const speechWindow = window as { SpeechRecognition?: unknown; webkitSpeechRecognition?: unknown };
    delete speechWindow.SpeechRecognition;
    delete speechWindow.webkitSpeechRecognition;
    useWorkspaceStore.setState({ voiceEnabled: true });
    mockMediaDevices();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("does not play a start cue when the durable backend is unavailable", async () => {
    mockCapabilities(false);

    const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
    const { result } = renderHook(() => useVoiceInput(vi.fn()));

    await act(async () => { await settle(); });
    expect(result.current.backend).toBe("none");

    await act(async () => { await result.current.startRecording(); });

    expect(startCueSpy).not.toHaveBeenCalled();
    expect(stopCueSpy).not.toHaveBeenCalled();
  });

  it("does not play a stop cue when a refused recording is stopped", async () => {
    mockCapabilities(false);

    const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
    const { result } = renderHook(() => useVoiceInput(vi.fn()));

    await act(async () => { await settle(); });
    await act(async () => { await result.current.startRecording(); });
    act(() => { result.current.stopRecording(); });

    expect(stopCueSpy).not.toHaveBeenCalled();
  });

  it("does not play cues on component unmount", async () => {
    mockCapabilities(false);

    const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
    const { unmount } = renderHook(() => useVoiceInput(vi.fn()));

    await act(async () => { await settle(); });
    unmount();

    expect(startCueSpy).not.toHaveBeenCalled();
    expect(stopCueSpy).not.toHaveBeenCalled();
  });

  it("does NOT play any cue on mount", async () => {
    mockCapabilities(true, true);

    const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
    renderHook(() => useVoiceInput(vi.fn()));

    await act(async () => { await settle(); });

    // No cues should play on mount — mic pre-warm is not a recording
    expect(startCueSpy).not.toHaveBeenCalled();
    expect(stopCueSpy).not.toHaveBeenCalled();
  });

  it("does not play cues on transcription cancellation without a provider", async () => {
    mockCapabilities(false);

    const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
    const { result } = renderHook(() => useVoiceInput(vi.fn()));

    await act(async () => { await settle(); });
    act(() => { result.current.cancelTranscription(); });

    expect(startCueSpy).not.toHaveBeenCalled();
    expect(stopCueSpy).not.toHaveBeenCalled();
  });

  it("does NOT play stop cue when stopRecording is called without active recording", async () => {
    mockCapabilities(false);
    const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
    const { result } = renderHook(() => useVoiceInput(vi.fn()));

    await act(async () => { await settle(); });

    // Call stopRecording without ever starting — no cues should play
    act(() => { result.current.stopRecording(); });

    expect(startCueSpy).not.toHaveBeenCalled();
    expect(stopCueSpy).not.toHaveBeenCalled();
  });

});
