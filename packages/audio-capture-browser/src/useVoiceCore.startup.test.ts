import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useVoiceCore } from "./useVoiceCore";
import type { VoiceCoreServices } from "./voice/services";
import type { TranscriptionProvider } from "./voice/types";

class StartFailProvider implements TranscriptionProvider {
  readonly markStartRequested = vi.fn();
  readonly start = vi.fn(async () => {
    throw new Error("capture device disappeared");
  });
  readonly stop = vi.fn();
  readonly dispose = vi.fn();
  readonly getStream = vi.fn(() => null);
  readonly getLastTurnAudio = vi.fn(() => null);
  readonly disposeLastTurn = vi.fn();
  readonly dropTail = vi.fn();
  onResult: ((text: string) => void) | null = null;
  onError: ((error: string) => void) | null = null;
  onPartial: ((text: string) => void) | null = null;
  onStatus: ((status: { code: string; message: string }) => void) | null = null;
  onDiagnostic = null;
}

function services(provider: StartFailProvider): VoiceCoreServices {
  return {
    PcmVoiceStreamProvider: class {
      constructor() {
        return provider as unknown as InstanceType<VoiceCoreServices["PcmVoiceStreamProvider"]>;
      }
    } as unknown as VoiceCoreServices["PcmVoiceStreamProvider"],
    getVoiceStreamConfig: vi.fn(async () => ({})),
    getWakeWordConfig: vi.fn(async () => ({ configured: false, template: null })),
    transcribeAudio: vi.fn(async () => ""),
    transcribeAudioBypassFilter: vi.fn(async () => ""),
    createAudioFilterChain: vi.fn(() => ({ analyser: {} as AnalyserNode, filteredStream: {} as MediaStream, nodes: [] })),
    playRecordingStartCue: vi.fn(),
    playRecordingStopCue: vi.fn(),
    WhisperProvider: class {
      start = vi.fn();
      stop = vi.fn();
      dispose = vi.fn();
      getStream = vi.fn(() => null);
      getLastTurnAudio = vi.fn(() => null);
      disposeLastTurn = vi.fn();
      dropTail = vi.fn();
      onResult = null;
      onError = null;
    },
    bytesToFeatures: vi.fn(async () => ({ frames: [], sampleRate: 16_000 })),
    createWakeWordEngine: vi.fn(() => ({ sampleRate: 16_000 } as never)),
    PassiveListener: class {
      start = vi.fn(async () => {});
      dispose = vi.fn();
    },
  };
}

describe("useVoiceCore provider startup", () => {
  it("starts capture before a slow capability probe resolves", async () => {
    const provider = new StartFailProvider();
    let resolveProbe!: (value: { whisperHealthy: boolean; streamingAvailable: boolean }) => void;
    const probe = new Promise<{ whisperHealthy: boolean; streamingAvailable: boolean }>((resolve) => {
      resolveProbe = resolve;
    });
    const { result } = renderHook(() => useVoiceCore({
      services: services(provider),
      capabilityCheck: () => probe,
      voiceEnabled: true,
      voiceLanguage: "en",
      vadSilenceTimeoutMs: 900,
      persistentMode: false,
      wakeWordEnabled: false,
      segmentSilenceMs: 900,
      onTranscript: vi.fn(),
    }));

    await waitFor(() => expect(result.current.supported).toBe(true));
    let start: Promise<void> | undefined;
    await act(async () => {
      start = result.current.startRecording();
      await Promise.resolve();
    });
    expect(provider.start).toHaveBeenCalledOnce();
    expect(provider.markStartRequested).toHaveBeenCalledOnce();
    resolveProbe({ whisperHealthy: true, streamingAvailable: true });
    await act(async () => { await start; });
    await waitFor(() => expect(result.current.voiceState).toBe("idle"));
  });

  it("returns to idle with an explicit error when capture startup rejects", async () => {
    const provider = new StartFailProvider();
    const { result } = renderHook(() => useVoiceCore({
      services: services(provider),
      voiceEnabled: true,
      voiceLanguage: "en",
      vadSilenceTimeoutMs: 900,
      persistentMode: false,
      wakeWordEnabled: false,
      segmentSilenceMs: 900,
      onTranscript: vi.fn(),
    }));

    await waitFor(() => expect(result.current.supported).toBe(true));
    await act(async () => {
      await result.current.startRecording();
    });

    await waitFor(() => expect(result.current.voiceState).toBe("idle"));
    expect(result.current.error).toBe("capture device disappeared");
    expect(provider.start).toHaveBeenCalledOnce();
    expect(provider.dispose).toHaveBeenCalledOnce();
  });

  it("ignores an unhealthy capability result from a stopped turn after a newer turn starts", async () => {
    const provider = new StartFailProvider();
    let resolveOldProbe!: (value: { whisperHealthy: boolean; streamingAvailable: boolean }) => void;
    let resolveNewProbe!: (value: { whisperHealthy: boolean; streamingAvailable: boolean }) => void;
    let probeCalls = 0;
    const capabilityCheck = vi.fn(() => {
      probeCalls++;
      if (probeCalls === 1) return Promise.resolve({ whisperHealthy: true, streamingAvailable: true });
      if (probeCalls === 2) return new Promise<{ whisperHealthy: boolean; streamingAvailable: boolean }>((resolve) => { resolveOldProbe = resolve; });
      return new Promise<{ whisperHealthy: boolean; streamingAvailable: boolean }>((resolve) => { resolveNewProbe = resolve; });
    });
    let rejectSecondStart!: (error: Error) => void;
    provider.start.mockImplementationOnce(async () => { throw new Error("first start failed"); })
      .mockImplementationOnce(() => new Promise<void>((_resolve, reject) => { rejectSecondStart = reject; }));
    const { result } = renderHook(() => useVoiceCore({
      services: services(provider),
      capabilityCheck,
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
    await waitFor(() => expect(result.current.voiceState).toBe("idle"));
    expect(provider.dispose).toHaveBeenCalledOnce();

    let secondStart!: Promise<void>;
    await act(async () => { secondStart = result.current.startRecording(); await Promise.resolve(); });
    await waitFor(() => expect(result.current.voiceState).toBe("preparing"));
    resolveOldProbe({ whisperHealthy: false, streamingAvailable: false });
    await act(async () => { await Promise.resolve(); });

    expect(provider.dispose).toHaveBeenCalledOnce();
    expect(result.current.voiceState).toBe("preparing");
    expect(result.current.supported).toBe(true);

    rejectSecondStart(new Error("second start failed"));
    await act(async () => { await secondStart; });
    resolveNewProbe({ whisperHealthy: true, streamingAvailable: true });
  });
});
