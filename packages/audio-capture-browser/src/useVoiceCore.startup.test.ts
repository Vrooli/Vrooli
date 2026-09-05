import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useVoiceCore } from "./useVoiceCore";
import type { VoiceCoreServices } from "./voice/services";
import type { TranscriptionProvider } from "./voice/types";

class StartFailProvider implements TranscriptionProvider {
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
});
