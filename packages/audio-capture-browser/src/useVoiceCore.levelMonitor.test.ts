import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useVoiceCore } from "./useVoiceCore";
import type { VoiceCoreServices } from "./voice/services";
import type { TranscriptionProvider } from "./voice/types";

const audioContext = vi.hoisted(() => ({
  ensureRunning: undefined as (() => Promise<AudioContext>) | undefined,
}));

vi.mock("./voice/sharedAudioContext", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./voice/sharedAudioContext")>();
  return {
    ...actual,
    getSharedAudioContext: () => ({ state: "running", resume: vi.fn(async () => {}) }),
    ensureRunningSharedAudioContext: () => audioContext.ensureRunning!(),
  };
});

class StreamingProvider implements TranscriptionProvider {
  onResult = null;
  onError = null;
  onPartial = null;
  onStatus = null;
  onDiagnostic = null;
  start = vi.fn(async () => {});
  stop = vi.fn();
  dispose = vi.fn();
  getStream = vi.fn(() => ({
    active: true,
    getTracks: () => [{ kind: "audio", readyState: "live", muted: false }],
    getAudioTracks: () => [{ kind: "audio", readyState: "live", muted: false }],
  } as unknown as MediaStream));
  getLastTurnAudio = vi.fn(() => null);
  disposeLastTurn = vi.fn();
  dropTail = vi.fn();
}

afterEach(() => {
  audioContext.ensureRunning = undefined;
  vi.unstubAllGlobals();
});

describe("useVoiceCore level monitor lifecycle", () => {
  it("does not install nodes or RAF after teardown wins a pending context resume", async () => {
    let resolveContext!: (context: AudioContext) => void;
    audioContext.ensureRunning = () => new Promise((resolve) => { resolveContext = resolve; });

    const provider = new StreamingProvider();
    const createAudioFilterChain = vi.fn(() => ({
      analyser: { fftSize: 32, getByteTimeDomainData: vi.fn() } as unknown as AnalyserNode,
      filteredStream: {} as MediaStream,
      nodes: [],
    }));
    const services: VoiceCoreServices = {
      PcmVoiceStreamProvider: class {
        constructor() { return provider as unknown as InstanceType<VoiceCoreServices["PcmVoiceStreamProvider"]>; }
      } as unknown as VoiceCoreServices["PcmVoiceStreamProvider"],
      getVoiceStreamConfig: vi.fn(async () => ({})),
      getWakeWordConfig: vi.fn(async () => ({ configured: false, template: null })),
      transcribeAudio: vi.fn(async () => ""),
      transcribeAudioBypassFilter: vi.fn(async () => ""),
      createAudioFilterChain,
      playRecordingStartCue: vi.fn(),
      playRecordingStopCue: vi.fn(),
      WhisperProvider: class {
        start = vi.fn(); stop = vi.fn(); dispose = vi.fn();
        getStream = vi.fn(() => null); getLastTurnAudio = vi.fn(() => null);
        disposeLastTurn = vi.fn(); dropTail = vi.fn();
      },
      bytesToFeatures: vi.fn(async () => ({ frames: [], sampleRate: 16_000 })),
      createWakeWordEngine: vi.fn(() => ({ sampleRate: 16_000 } as never)),
      PassiveListener: class { start = vi.fn(async () => {}); dispose = vi.fn(); },
    };
    const requestAnimationFrame = vi.fn(() => 1);
    vi.stubGlobal("requestAnimationFrame", requestAnimationFrame);
    vi.stubGlobal("cancelAnimationFrame", vi.fn());

    const { result } = renderHook(() => useVoiceCore({
      services,
      voiceEnabled: true,
      voiceLanguage: "en",
      vadSilenceTimeoutMs: 900,
      persistentMode: false,
      wakeWordEnabled: false,
      segmentSilenceMs: 900,
      onTranscript: vi.fn(),
    }));

    await waitFor(() => expect(result.current.supported).toBe(true));
    await act(async () => { await result.current.startRecording(); });
    expect(provider.getStream).toHaveBeenCalledOnce();

    await act(async () => { result.current.releaseMicrophone(); });
    resolveContext({ state: "running" } as AudioContext);
    await act(async () => { await Promise.resolve(); });

    expect(createAudioFilterChain).not.toHaveBeenCalled();
    expect(requestAnimationFrame).not.toHaveBeenCalled();
  });

  it("disconnects the media source when filter-chain setup throws", async () => {
    const provider = new StreamingProvider();
    const source = { disconnect: vi.fn() };
    const createAudioFilterChain = vi.fn(() => { throw new Error("filter setup failed"); });
    const services: VoiceCoreServices = {
      PcmVoiceStreamProvider: class {
        constructor() { return provider as unknown as InstanceType<VoiceCoreServices["PcmVoiceStreamProvider"]>; }
      } as unknown as VoiceCoreServices["PcmVoiceStreamProvider"],
      getVoiceStreamConfig: vi.fn(async () => ({})),
      getWakeWordConfig: vi.fn(async () => ({ configured: false, template: null })),
      transcribeAudio: vi.fn(async () => ""),
      transcribeAudioBypassFilter: vi.fn(async () => ""),
      createAudioFilterChain,
      playRecordingStartCue: vi.fn(),
      playRecordingStopCue: vi.fn(),
      playRecordingFaultCue: vi.fn(),
      WhisperProvider: class {
        start = vi.fn(); stop = vi.fn(); dispose = vi.fn();
        getStream = vi.fn(() => null); getLastTurnAudio = vi.fn(() => null);
        disposeLastTurn = vi.fn(); dropTail = vi.fn();
      },
      bytesToFeatures: vi.fn(async () => ({ frames: [], sampleRate: 16_000 })),
      createWakeWordEngine: vi.fn(() => ({ sampleRate: 16_000 } as never)),
      PassiveListener: class { start = vi.fn(async () => {}); dispose = vi.fn(); },
    };
    const context = {
      state: "running",
      createMediaStreamSource: vi.fn(() => source),
    } as unknown as AudioContext;
    audioContext.ensureRunning = async () => context;

    const { result } = renderHook(() => useVoiceCore({
      services,
      voiceEnabled: true,
      voiceLanguage: "en",
      vadSilenceTimeoutMs: 900,
      persistentMode: false,
      wakeWordEnabled: false,
      segmentSilenceMs: 900,
      onTranscript: vi.fn(),
    }));

    await waitFor(() => expect(result.current.supported).toBe(true));
    await act(async () => { await result.current.startRecording(); });
    await waitFor(() => expect(createAudioFilterChain).toHaveBeenCalledOnce());
    expect(source.disconnect).toHaveBeenCalledOnce();
  });
});
