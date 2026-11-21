// Unit tests for the StreamConfig codec: decodeStreamConfig must surface
// the five advanced fields and updateVoiceStreamConfig must build a
// FieldMask that includes every patched advanced path.

import { describe, expect, it, vi } from "vitest";

import {
  StreamingMode,
  StrategyPreference,
} from "@vrooli/proto-types/audio-tools/v1/stt/stt_pb";

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
