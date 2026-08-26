import { beforeEach, describe, expect, it, vi } from "vitest";
import { SummarizeLevel } from "@vrooli/proto-types/web-console/v1/audio_common/audio_common_pb";
import { audioAdminClient, audioRuntimeClient } from "./voice";
import {
  fetchCachedTTS,
  getTTSConfig,
  getTTSSummarizeConfig,
  getTTSVoices,
  listTTSSummarizeModels,
  reportTTSPlayStart,
  reportTTSEvent,
  synthesizeTTS,
  synthesizeTTSWithMetrics,
  updateTTSConfig,
  updateTTSSummarizeConfig,
} from "./tts";

describe("TTS API adapters", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("maps config, voices, models, playback events, and cache", async () => {
    vi.spyOn(audioRuntimeClient, "listVoices").mockResolvedValue({ voices: [{ id: "v", name: "Voice" }] } as never);
    vi.spyOn(audioRuntimeClient, "getTTSCache").mockResolvedValue({ hit: true, audio: new Uint8Array([1, 2]), contentType: "audio/ogg" } as never);
    vi.spyOn(audioRuntimeClient, "recordPlaybackEvent").mockResolvedValue({} as never);
    vi.spyOn(audioAdminClient, "getTTSConfig").mockResolvedValue({ config: { autoEnabled: true, defaultVoice: "v", defaultSpeed: 1.2, defaultResponseFormat: 0 } } as never);
    vi.spyOn(audioAdminClient, "updateTTSConfig").mockResolvedValue({ config: { autoEnabled: false, defaultVoice: "w", defaultSpeed: 0.8, defaultResponseFormat: 0 } } as never);
    vi.spyOn(audioAdminClient, "getSummarizeConfig").mockResolvedValue({ config: { enabled: true, charThreshold: 100, level: SummarizeLevel.LIGHT, model: "m", timeoutSeconds: 4 } } as never);
    vi.spyOn(audioAdminClient, "listSummarizeModels").mockResolvedValue({ models: [{ id: "m", displayName: "", installed: true, recommended: true, defaultEligible: true, reasoning: false, statusLabel: "ready", pullCommand: "", sizeBytes: 2n, parameterSize: "7b", sourceUrl: "", notes: "" }] } as never);
    vi.spyOn(audioAdminClient, "updateSummarizeConfig").mockResolvedValue({ config: { enabled: false, charThreshold: 200, level: SummarizeLevel.HEAVY, model: "n", timeoutSeconds: 9 } } as never);

    await expect(getTTSVoices()).resolves.toEqual([{ id: "v", name: "Voice" }]);
    await expect(getTTSConfig()).resolves.toMatchObject({ autoEnabled: true, defaultVoice: "v" });
    await expect(updateTTSConfig({ autoEnabled: false, defaultVoice: "w", defaultSpeed: 0.8, defaultResponseFormat: "ogg" })).resolves.toMatchObject({ defaultVoice: "w" });
    await expect(getTTSSummarizeConfig()).resolves.toMatchObject({ level: "light", charThreshold: 100 });
    await expect(listTTSSummarizeModels()).resolves.toMatchObject([{ displayName: "m", sizeBytes: 2n }]);
    await expect(updateTTSSummarizeConfig({ enabled: false, charThreshold: 200, level: "heavy", model: "n", timeoutSeconds: 9 })).resolves.toMatchObject({ level: "heavy", model: "n" });
    await expect(fetchCachedTTS("e", "v", 1, "original", undefined, 2)).resolves.toBeInstanceOf(Blob);
    await reportTTSEvent({ source: "ui", stage: "stop" });
    reportTTSPlayStart({ requestId: "r", synthStartMs: performance.now(), totalChars: 2 });
  });

  it("synthesizes with metrics and degrades cache failures to misses", async () => {
    const synth = vi.spyOn(audioRuntimeClient, "synthesize").mockResolvedValue({ audio: new Uint8Array([1]), contentType: "audio/wav" } as never);
    vi.spyOn(audioRuntimeClient, "recordPlaybackEvent").mockResolvedValue({} as never);
    const result = await synthesizeTTSWithMetrics("hello", "v", 1, undefined, { eventId: "e", chunkIndex: 2, version: "active" });
    expect(result.blob.type).toBe("audio/wav");
    expect(result.metrics.totalChars).toBe(5);
    await expect(synthesizeTTS("hello", "v", 1)).resolves.toBeInstanceOf(Blob);
    expect(synth).toHaveBeenCalledWith(expect.objectContaining({ text: "hello", eventId: "e", chunkIndex: 2 }), expect.objectContaining({ headers: { "x-tts-request-id": expect.any(String) } }));

    synth.mockRejectedValueOnce(new Error("offline"));
    await expect(synthesizeTTSWithMetrics("x")).rejects.toThrow("offline");
    vi.spyOn(audioRuntimeClient, "getTTSCache").mockResolvedValueOnce({ hit: false, audio: new Uint8Array(), contentType: "" } as never).mockRejectedValueOnce(new Error("offline"));
    await expect(fetchCachedTTS("e", "v", 1)).resolves.toBeNull();
    await expect(fetchCachedTTS("e", "v", 1)).resolves.toBeNull();
  });
});
