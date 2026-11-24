// Unit tests for the StreamConfig codec: decodeStreamConfig must surface
// the five advanced fields and updateVoiceStreamConfig must build a
// FieldMask that includes every patched advanced path.

import { describe, expect, it, vi } from "vitest";
import { create, toBinary } from "@bufbuild/protobuf";

import {
  StreamingMode,
  StrategyPreference,
} from "@vrooli/proto-types/audio-tools/v1/stt/stt_pb";
import { TranscribeStreamEventSchema } from "@vrooli/proto-types/audio-tools/v1/stt/stt_pb";

import {
  streamingModeLabel,
  strategyPreferenceLabel,
} from "@vrooli/audio-capture-browser";
import { createVoiceApi } from "./voice";
import type { AudioToolsClient } from "../client";

function makeFakeClient(opts: {
  getStreamConfig?: ReturnType<typeof vi.fn>;
  updateStreamConfig?: ReturnType<typeof vi.fn>;
}): AudioToolsClient {
  return {
    baseUrl: "http://test",
    stt: { transcribe: vi.fn() } as never,
    sttAdmin: {
      getStreamConfig: opts.getStreamConfig ?? vi.fn(),
      updateStreamConfig: opts.updateStreamConfig ?? vi.fn(),
      getWakeWordConfig: vi.fn(),
    } as never,
    tts: {} as never,
    summarize: {} as never,
  };
}

describe("streamingModeLabel", () => {
  it("maps each enum value to its CLI label", () => {
    expect(streamingModeLabel(StreamingMode.AUTO)).toBe("auto");
    expect(streamingModeLabel(StreamingMode.OFF)).toBe("off");
    expect(streamingModeLabel(StreamingMode.UNSPECIFIED)).toBe("unspecified");
    expect(streamingModeLabel(undefined)).toBe("unspecified");
  });
});

describe("browser voice WebSocket routing", () => {
  it("uses the canonical same-origin upgrade path without duplicating api-base /ws", () => {
    const original = globalThis.WebSocket;
    class SocketStub {
      static CONNECTING = 0;
      static OPEN = 1;
      static CLOSING = 2;
      static CLOSED = 3;
      static url = "";
      readyState = SocketStub.CONNECTING;
      bufferedAmount = 0;
      binaryType: BinaryType = "blob";
      onopen: ((event: Event) => void) | null = null;
      onmessage: ((event: MessageEvent) => void) | null = null;
      onerror: ((event: Event) => void) | null = null;
      onclose: ((event: CloseEvent) => void) | null = null;
      constructor(url: string) { SocketStub.url = url; }
      send(): void {}
      close(): void {}
    }
    vi.stubGlobal("WebSocket", SocketStub);
    try {
      const api = createVoiceApi(makeFakeClient({}));
      api.openVoiceStream({
        language: "en", sessionId: "session", resumeToken: "token", protocolVersion: 2,
        generation: 0n, inputFormat: "pcm_s16le", sampleRateHz: 16000,
      }, { onReady: vi.fn(), onEvent: vi.fn(), onClose: vi.fn() });
      expect(SocketStub.url).toBe("ws://test/ws/stt/transcribe-stream-v2");
    } finally {
      vi.stubGlobal("WebSocket", original);
    }
  });

  it("surfaces failed StreamDone terminal reasons as errors, not final transcripts", () => {
    const original = globalThis.WebSocket;
    class SocketStub {
      static CONNECTING = 0;
      static OPEN = 1;
      static CLOSING = 2;
      static CLOSED = 3;
      static instance: SocketStub | undefined;
      readyState = SocketStub.CONNECTING;
      bufferedAmount = 0;
      binaryType: BinaryType = "blob";
      onopen: ((event: Event) => void) | null = null;
      onmessage: ((event: MessageEvent) => void) | null = null;
      onerror: ((event: Event) => void) | null = null;
      onclose: ((event: CloseEvent) => void) | null = null;
      constructor() { SocketStub.instance = this; }
      send(): void {}
      close(): void {}
    }
    vi.stubGlobal("WebSocket", SocketStub);
    try {
      const onEvent = vi.fn();
      const api = createVoiceApi(makeFakeClient({}));
      api.openVoiceStream({
        language: "en", sessionId: "failed-session", resumeToken: "token", protocolVersion: 2,
        generation: 0n, inputFormat: "pcm_s16le", sampleRateHz: 16000,
      }, { onReady: vi.fn(), onEvent, onClose: vi.fn() });
      const socket = SocketStub.instance!;
      socket.readyState = SocketStub.OPEN;
      socket.onopen?.({} as Event);
      const accepted = create(TranscribeStreamEventSchema, { event: { case: "sessionStatus", value: { state: "accepted" } } });
      socket.onmessage?.({ data: toBinary(TranscribeStreamEventSchema, accepted).buffer } as MessageEvent);
      const failed = create(TranscribeStreamEventSchema, { event: { case: "done", value: { terminalReason: "input_failed", finalText: "partial" } } });
      socket.onmessage?.({ data: toBinary(TranscribeStreamEventSchema, failed).buffer } as MessageEvent);
      expect(onEvent).toHaveBeenCalledWith({
        type: "error", code: "input_failed", text: "The speech stream did not complete all captured audio.",
      });
      expect(onEvent).not.toHaveBeenCalledWith(expect.objectContaining({ type: "final" }));
    } finally {
      vi.stubGlobal("WebSocket", original);
    }
  });
});

describe("strategyPreferenceLabel", () => {
  it("maps each enum value to its CLI label", () => {
    expect(strategyPreferenceLabel(StrategyPreference.AUTO)).toBe("auto");
    expect(strategyPreferenceLabel(StrategyPreference.VAD)).toBe("vad");
    expect(strategyPreferenceLabel(StrategyPreference.OVERLAP)).toBe("overlap");
    expect(strategyPreferenceLabel(StrategyPreference.PASSTHROUGH)).toBe("passthrough");
    expect(strategyPreferenceLabel(StrategyPreference.UNSPECIFIED)).toBe("unspecified");
    expect(strategyPreferenceLabel(undefined)).toBe("unspecified");
  });
});

describe("getVoiceStreamConfig", () => {
  it("decodes the five advanced fields", async () => {
    const getStreamConfig = vi.fn().mockResolvedValue({
      config: {
        flushIntervalMs: 250,
        minDeltaBytes: 16384,
        overlapBytes: 2048,
        persistentMode: false,
        wakeWordEnabled: false,
        wakeWordThreshold: 0,
        segmentSilenceMs: 800,
        streamingMode: StreamingMode.AUTO,
        strategyPreference: StrategyPreference.OVERLAP,
        vadSilenceMs: 1200,
        overlapWindowMs: 3000,
        overlapCommitRuns: 3,
      },
    });
    const api = createVoiceApi(makeFakeClient({ getStreamConfig }));
    const cfg = await api.getVoiceStreamConfig();
    expect(cfg.streamingMode).toBe("auto");
    expect(cfg.strategyPreference).toBe("overlap");
    expect(cfg.vadSilenceMs).toBe(1200);
    expect(cfg.overlapWindowMs).toBe(3000);
    expect(cfg.overlapCommitRuns).toBe(3);
  });

  it("decodes the overlap stall-reject guard", async () => {
    const getStreamConfig = vi.fn().mockResolvedValue({
      config: { overlapMaxStallRejects: 5 },
    });
    const api = createVoiceApi(makeFakeClient({ getStreamConfig }));
    const cfg = await api.getVoiceStreamConfig();
    expect(cfg.overlapMaxStallRejects).toBe(5);
  });

  it("defaults advanced fields when the server omits them", async () => {
    const getStreamConfig = vi.fn().mockResolvedValue({ config: undefined });
    const api = createVoiceApi(makeFakeClient({ getStreamConfig }));
    const cfg = await api.getVoiceStreamConfig();
    expect(cfg.streamingMode).toBe("unspecified");
    expect(cfg.strategyPreference).toBe("unspecified");
    expect(cfg.vadSilenceMs).toBe(0);
    expect(cfg.overlapWindowMs).toBe(0);
    expect(cfg.overlapCommitRuns).toBe(0);
  });
});

describe("updateVoiceStreamConfig", () => {
  it("builds a FieldMask that includes patched advanced paths", async () => {
    const updateStreamConfig = vi.fn().mockResolvedValue({
      config: {
        flushIntervalMs: 0, minDeltaBytes: 0, overlapBytes: 0,
        persistentMode: false, wakeWordEnabled: false, wakeWordThreshold: 0, segmentSilenceMs: 0,
        streamingMode: StreamingMode.AUTO,
        strategyPreference: StrategyPreference.VAD,
        vadSilenceMs: 1500,
        overlapWindowMs: 0,
        overlapCommitRuns: 0,
      },
    });
    const api = createVoiceApi(makeFakeClient({ updateStreamConfig }));
    await api.updateVoiceStreamConfig({
      vadSilenceMs: 1500,
      strategyPreference: "vad",
      streamingMode: "auto",
      overlapWindowMs: 2500,
      overlapCommitRuns: 2,
    });
    expect(updateStreamConfig).toHaveBeenCalledTimes(1);
    const callArg = updateStreamConfig.mock.calls[0]![0];
    const paths: string[] = callArg.updateMask.paths;
    expect(paths).toContain("vad_silence_ms");
    expect(paths).toContain("strategy_preference");
    expect(paths).toContain("streaming_mode");
    expect(paths).toContain("overlap_window_ms");
    expect(paths).toContain("overlap_commit_runs");
    expect(callArg.config.vadSilenceMs).toBe(1500);
    expect(callArg.config.strategyPreference).toBe(StrategyPreference.VAD);
    expect(callArg.config.streamingMode).toBe(StreamingMode.AUTO);
  });

  it("includes overlap_max_stall_rejects when the guard is patched", async () => {
    const updateStreamConfig = vi.fn().mockResolvedValue({ config: {} });
    const api = createVoiceApi(makeFakeClient({ updateStreamConfig }));
    await api.updateVoiceStreamConfig({ overlapMaxStallRejects: 0 });
    const callArg = updateStreamConfig.mock.calls[0]![0];
    const paths: string[] = callArg.updateMask.paths;
    expect(paths).toEqual(["overlap_max_stall_rejects"]);
    expect(callArg.config.overlapMaxStallRejects).toBe(0);
  });

  it("omits paths for fields that were not patched", async () => {
    const updateStreamConfig = vi.fn().mockResolvedValue({ config: {} });
    const api = createVoiceApi(makeFakeClient({ updateStreamConfig }));
    await api.updateVoiceStreamConfig({ vadSilenceMs: 900 });
    const callArg = updateStreamConfig.mock.calls[0]![0];
    const paths: string[] = callArg.updateMask.paths;
    expect(paths).toEqual(["vad_silence_ms"]);
  });
});
