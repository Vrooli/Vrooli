// Unit tests for the StreamConfig codec: decodeStreamConfig must surface
// the five advanced fields (streamingMode, strategyPreference, vadSilenceMs,
// overlapWindowMs, overlapCommitRuns), and updateVoiceStreamConfig must
// build a FieldMask that includes every patched advanced path.

import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";

import {
  streamingModeLabel,
  strategyPreferenceLabel,
} from "./protomap";
import {
  StreamingMode,
  StrategyPreference,
} from "@vrooli/proto-types/web-console/v1/audio_common/audio_common_pb";

vi.mock("../../api/client", () => ({
  transport: {},
  API_BASE: "http://test",
}));

interface UpdateStreamConfigArg {
  updateMask: { paths: string[] };
  config: {
    vadSilenceMs?: number;
    strategyPreference?: StrategyPreference;
    streamingMode?: StreamingMode;
  };
}

function requireDefined<T>(value: T | undefined, message: string): T {
  if (value === undefined) throw new Error(message);
  return value;
}

const updateMock = vi.fn<(req: UpdateStreamConfigArg) => Promise<{ config: Record<string, unknown> }>>();
const getMock = vi.fn();

vi.mock("@connectrpc/connect", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@connectrpc/connect")>();
  return {
    ...actual,
    createClient: () => ({
      getStreamConfig: getMock,
      updateStreamConfig: updateMock,
      getWakeWordConfig: vi.fn(),
      updateWakeWordTemplate: vi.fn(),
      deleteWakeWordTemplate: vi.fn(),
      getSpeakerVerificationStatus: vi.fn(),
      getSpeakerVerificationConfig: vi.fn(),
      updateSpeakerVerificationConfig: vi.fn(),
      enrollSpeakerVerification: vi.fn(),
      deleteSpeakerVerificationProfile: vi.fn(),
      transcribe: vi.fn(),
    }),
  };
});

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
  beforeEach(() => {
    getMock.mockReset();
    updateMock.mockReset();
  });
  afterEach(() => {
    vi.resetModules();
  });

  it("decodes the five advanced fields", async () => {
    getMock.mockResolvedValueOnce({
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
    const { getVoiceStreamConfig } = await import("./voice");
    const cfg = await getVoiceStreamConfig();
    expect(cfg.streamingMode).toBe("auto");
    expect(cfg.strategyPreference).toBe("overlap");
    expect(cfg.vadSilenceMs).toBe(1200);
    expect(cfg.overlapWindowMs).toBe(3000);
    expect(cfg.overlapCommitRuns).toBe(3);
  });

  it("defaults advanced fields when the server omits them", async () => {
    getMock.mockResolvedValueOnce({ config: undefined });
    const { getVoiceStreamConfig } = await import("./voice");
    const cfg = await getVoiceStreamConfig();
    expect(cfg.streamingMode).toBe("unspecified");
    expect(cfg.strategyPreference).toBe("unspecified");
    expect(cfg.vadSilenceMs).toBe(0);
    expect(cfg.overlapWindowMs).toBe(0);
    expect(cfg.overlapCommitRuns).toBe(0);
  });
});

describe("updateVoiceStreamConfig", () => {
  beforeEach(() => {
    getMock.mockReset();
    updateMock.mockReset();
  });
  afterEach(() => {
    vi.resetModules();
  });

  it("builds a FieldMask that includes patched advanced paths", async () => {
    updateMock.mockResolvedValueOnce({
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
    const { updateVoiceStreamConfig } = await import("./voice");
    await updateVoiceStreamConfig({
      vadSilenceMs: 1500,
      strategyPreference: "vad",
      streamingMode: "auto",
      overlapWindowMs: 2500,
      overlapCommitRuns: 2,
    });
    expect(updateMock).toHaveBeenCalledTimes(1);
    const callArg = requireDefined(updateMock.mock.calls[0], "updateStreamConfig was not called")[0];
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

  it("omits paths for fields that were not patched", async () => {
    updateMock.mockResolvedValueOnce({ config: {} });
    const { updateVoiceStreamConfig } = await import("./voice");
    await updateVoiceStreamConfig({ vadSilenceMs: 900 });
    const callArg = requireDefined(updateMock.mock.calls[0], "updateStreamConfig was not called")[0];
    const paths: string[] = callArg.updateMask.paths;
    expect(paths).toEqual(["vad_silence_ms"]);
  });
});
