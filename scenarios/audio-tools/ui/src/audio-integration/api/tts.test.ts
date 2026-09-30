// Unit tests for tts.ts — createTtsApi and the module-level lazy() wrappers.
//
// Build a fake AudioToolsClient (same pattern as voice.test.ts / voice.more.test.ts)
// and drive each exported function through success and error/edge paths.

import { beforeEach, afterEach, describe, expect, it, vi } from "vitest";

// jsdom does not implement AbortSignal.any (added in WHATWG Streams 1.0+)
// Polyfill it so synthesizeTTS's AbortSignal.any([signal, timeout]) branch runs.
if (typeof AbortSignal.any === "undefined") {
  (AbortSignal as unknown as Record<string, unknown>).any = (signals: AbortSignal[]): AbortSignal => {
    const controller = new AbortController();
    for (const signal of signals) {
      if (signal.aborted) {
        controller.abort(signal.reason);
        break;
      }
      signal.addEventListener("abort", () => controller.abort(signal.reason), { once: true });
    }
    return controller.signal;
  };
}
import { SummarizeLevel } from "@vrooli/proto-types/audio-tools/v1/summarize/summarize_pb";
import { ResponseFormat } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";

import type { AudioToolsClient } from "../client";
import { setActiveAudioToolsClientForTesting } from "../client";


import {
  createTtsApi,
  reportTTSEvent,
  synthesizeTTS,
  fetchCachedTTS,
  getTTSVoices,
  getTTSConfig,
  updateTTSConfig,
  getTTSSummarizeConfig,
  updateTTSSummarizeConfig,
} from "./tts";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type FnMap = Partial<Record<string, ReturnType<typeof vi.fn>>>;

function makeFakeClient(fns: FnMap = {}, baseUrl = "http://test"): AudioToolsClient {
  const f = (name: string) => fns[name] ?? vi.fn();
  return {
    baseUrl,
    stt: {} as never,
    sttAdmin: {} as never,
    tts: {
      synthesize: f("synthesize"),
      getCache: f("getCache"),
      listVoices: f("listVoices"),
      getConfig: f("getConfig"),
      updateConfig: f("updateConfig"),
      recordPlaybackEvent: f("recordPlaybackEvent"),
    } as never,
    summarize: {
      getSummarizeConfig: f("getSummarizeConfig"),
      updateSummarizeConfig: f("updateSummarizeConfig"),
    } as never,
  };
}

// ---------------------------------------------------------------------------
// synthesizeTTS
// ---------------------------------------------------------------------------

describe("synthesizeTTS", () => {
  it("returns a Blob with the response audio", async () => {
    const audio = new Uint8Array([1, 2, 3]);
    const synthesize = vi.fn().mockResolvedValue({
      audio,
      contentType: "audio/mpeg",
    });
    const api = createTtsApi(makeFakeClient({ synthesize }));
    const blob = await api.synthesizeTTS("hello");
    expect(blob).toBeInstanceOf(Blob);
    expect(blob.type).toBe("audio/mpeg");
    expect(blob.size).toBe(3);
  });

  it("defaults to audio/mpeg when contentType is absent", async () => {
    const synthesize = vi.fn().mockResolvedValue({
      audio: new Uint8Array([9]),
      contentType: "",
    });
    const api = createTtsApi(makeFakeClient({ synthesize }));
    const blob = await api.synthesizeTTS("test");
    expect(blob.type).toBe("audio/mpeg");
  });

  it("passes voice and speed to the request", async () => {
    const synthesize = vi.fn().mockResolvedValue({ audio: new Uint8Array(), contentType: "audio/mpeg" });
    const api = createTtsApi(makeFakeClient({ synthesize }));
    await api.synthesizeTTS("hi", "en-us", 1.2);
    const req = synthesize.mock.calls[0]![0];
    expect(req.voice).toBe("en-us");
    expect(req.speed).toBe(1.2);
  });

  it("defaults voice to empty string when not provided", async () => {
    const synthesize = vi.fn().mockResolvedValue({ audio: new Uint8Array(), contentType: "audio/mpeg" });
    const api = createTtsApi(makeFakeClient({ synthesize }));
    await api.synthesizeTTS("hello");
    expect(synthesize.mock.calls[0]![0].voice).toBe("");
  });

  it("defaults speed to 0 when not provided", async () => {
    const synthesize = vi.fn().mockResolvedValue({ audio: new Uint8Array(), contentType: "audio/mpeg" });
    const api = createTtsApi(makeFakeClient({ synthesize }));
    await api.synthesizeTTS("hello");
    expect(synthesize.mock.calls[0]![0].speed).toBe(0);
  });

  it("forwards an AbortSignal to the underlying call", async () => {
    const synthesize = vi.fn().mockResolvedValue({ audio: new Uint8Array(), contentType: "audio/mpeg" });
    const api = createTtsApi(makeFakeClient({ synthesize }));
    const controller = new AbortController();
    await api.synthesizeTTS("hi", undefined, undefined, controller.signal);
    // The call options object is the second arg to synthesize
    const callOptions = synthesize.mock.calls[0]![1];
    expect(callOptions).toBeDefined();
    expect(callOptions.signal).toBeDefined();
  });
});

// ---------------------------------------------------------------------------
// fetchCachedTTS
// ---------------------------------------------------------------------------

describe("fetchCachedTTS", () => {
  it("returns null when audio is empty", async () => {
    const getCache = vi.fn().mockResolvedValue({
      audio: new Uint8Array(0),
      contentType: "audio/mpeg",
    });
    const api = createTtsApi(makeFakeClient({ getCache }));
    const result = await api.fetchCachedTTS("ev1", "en-us", 1.0);
    expect(result).toBeNull();
  });

  it("returns a Blob when audio is present", async () => {
    const getCache = vi.fn().mockResolvedValue({
      audio: new Uint8Array([5, 6, 7]),
      contentType: "audio/mpeg",
    });
    const api = createTtsApi(makeFakeClient({ getCache }));
    const blob = await api.fetchCachedTTS("ev1", "en-us", 1.0);
    expect(blob).toBeInstanceOf(Blob);
    expect(blob!.type).toBe("audio/mpeg");
  });

  it("returns null on network error (swallows exception)", async () => {
    const getCache = vi.fn().mockRejectedValue(new Error("network error"));
    const api = createTtsApi(makeFakeClient({ getCache }));
    const result = await api.fetchCachedTTS("ev1", "en-us", 1.0);
    expect(result).toBeNull();
  });

  it("passes eventId, voice, speed, version to the cache RPC", async () => {
    const getCache = vi.fn().mockResolvedValue({ audio: new Uint8Array([1]), contentType: "" });
    const api = createTtsApi(makeFakeClient({ getCache }));
    await api.fetchCachedTTS("ev42", "uk-ua", 0.9, "original");
    const req = getCache.mock.calls[0]![0];
    expect(req.eventId).toBe("ev42");
    expect(req.voice).toBe("uk-ua");
    expect(req.speed).toBe(0.9);
    expect(req.version).toBe("original");
  });

  it("defaults version to 'active' when not provided", async () => {
    const getCache = vi.fn().mockResolvedValue({ audio: new Uint8Array([1]), contentType: "" });
    const api = createTtsApi(makeFakeClient({ getCache }));
    await api.fetchCachedTTS("ev1", "v1", 1.0);
    expect(getCache.mock.calls[0]![0].version).toBe("active");
  });

  it("defaults contentType to audio/mpeg when absent", async () => {
    const getCache = vi.fn().mockResolvedValue({ audio: new Uint8Array([1]), contentType: "" });
    const api = createTtsApi(makeFakeClient({ getCache }));
    const blob = await api.fetchCachedTTS("ev1", "v1", 1.0);
    expect(blob!.type).toBe("audio/mpeg");
  });
});

// ---------------------------------------------------------------------------
// getTTSVoices
// ---------------------------------------------------------------------------

describe("getTTSVoices", () => {
  it("maps voices to id/name pairs", async () => {
    const listVoices = vi.fn().mockResolvedValue({
      voices: [
        { id: "v1", name: "Alice" },
        { id: "v2", name: "Bob" },
      ],
    });
    const api = createTtsApi(makeFakeClient({ listVoices }));
    const voices = await api.getTTSVoices();
    expect(voices).toEqual([
      { id: "v1", name: "Alice" },
      { id: "v2", name: "Bob" },
    ]);
  });

  it("returns empty array when no voices", async () => {
    const listVoices = vi.fn().mockResolvedValue({ voices: [] });
    const api = createTtsApi(makeFakeClient({ listVoices }));
    expect(await api.getTTSVoices()).toEqual([]);
  });
});

// ---------------------------------------------------------------------------
// getTTSConfig / updateTTSConfig
// ---------------------------------------------------------------------------

describe("getTTSConfig", () => {
  it("decodes the config with all fields", async () => {
    const getConfig = vi.fn().mockResolvedValue({
      config: {
        autoEnabled: true,
        defaultVoice: "en-us",
        defaultSpeed: 1.25,
        defaultResponseFormat: ResponseFormat.MP3,
      },
    });
    const api = createTtsApi(makeFakeClient({ getConfig }));
    const cfg = await api.getTTSConfig();
    expect(cfg.autoEnabled).toBe(true);
    expect(cfg.defaultVoice).toBe("en-us");
    expect(cfg.defaultSpeed).toBe(1.25);
    expect(cfg.defaultResponseFormat).toBe("mp3");
  });

  it("falls back to defaults when config is undefined", async () => {
    const getConfig = vi.fn().mockResolvedValue({ config: undefined });
    const api = createTtsApi(makeFakeClient({ getConfig }));
    const cfg = await api.getTTSConfig();
    expect(cfg.autoEnabled).toBe(false);
    expect(cfg.defaultVoice).toBe("");
    expect(cfg.defaultSpeed).toBe(1);
    expect(cfg.defaultResponseFormat).toBe("mp3"); // responseFormatLabel(undefined) → "" → fallback "mp3"
  });

  it("decodes wav response format", async () => {
    const getConfig = vi.fn().mockResolvedValue({
      config: { autoEnabled: false, defaultVoice: "", defaultSpeed: 1, defaultResponseFormat: ResponseFormat.WAV },
    });
    const api = createTtsApi(makeFakeClient({ getConfig }));
    const cfg = await api.getTTSConfig();
    expect(cfg.defaultResponseFormat).toBe("wav");
  });

  it("decodes opus response format", async () => {
    const getConfig = vi.fn().mockResolvedValue({
      config: { autoEnabled: false, defaultVoice: "", defaultSpeed: 1, defaultResponseFormat: ResponseFormat.OPUS },
    });
    const api = createTtsApi(makeFakeClient({ getConfig }));
    const cfg = await api.getTTSConfig();
    expect(cfg.defaultResponseFormat).toBe("opus");
  });

  it("decodes flac response format", async () => {
    const getConfig = vi.fn().mockResolvedValue({
      config: { autoEnabled: false, defaultVoice: "", defaultSpeed: 1, defaultResponseFormat: ResponseFormat.FLAC },
    });
    const api = createTtsApi(makeFakeClient({ getConfig }));
    const cfg = await api.getTTSConfig();
    expect(cfg.defaultResponseFormat).toBe("flac");
  });
});

describe("updateTTSConfig", () => {
  it("builds correct field mask and config for all fields", async () => {
    const updateConfig = vi.fn().mockResolvedValue({
      config: {
        autoEnabled: true,
        defaultVoice: "uk",
        defaultSpeed: 0.8,
        defaultResponseFormat: ResponseFormat.WAV,
      },
    });
    const api = createTtsApi(makeFakeClient({ updateConfig }));
    const result = await api.updateTTSConfig({
      autoEnabled: true,
      defaultVoice: "uk",
      defaultSpeed: 0.8,
      defaultResponseFormat: "wav",
    });
    const { updateMask, config } = updateConfig.mock.calls[0]![0];
    expect(updateMask.paths).toContain("auto_enabled");
    expect(updateMask.paths).toContain("default_voice");
    expect(updateMask.paths).toContain("default_speed");
    expect(updateMask.paths).toContain("default_response_format");
    expect(config.autoEnabled).toBe(true);
    expect(config.defaultVoice).toBe("uk");
    expect(result.defaultResponseFormat).toBe("wav");
  });

  it("omits paths for unpatched fields", async () => {
    const updateConfig = vi.fn().mockResolvedValue({
      config: { autoEnabled: false, defaultVoice: "", defaultSpeed: 1, defaultResponseFormat: ResponseFormat.UNSPECIFIED },
    });
    const api = createTtsApi(makeFakeClient({ updateConfig }));
    await api.updateTTSConfig({ autoEnabled: false });
    expect(updateConfig.mock.calls[0]![0].updateMask.paths).toEqual(["auto_enabled"]);
  });
});

// ---------------------------------------------------------------------------
// getTTSSummarizeConfig / updateTTSSummarizeConfig
// ---------------------------------------------------------------------------

describe("getTTSSummarizeConfig", () => {
  it("decodes light level", async () => {
    const getSummarizeConfig = vi.fn().mockResolvedValue({
      config: { enabled: true, charThreshold: 500, level: SummarizeLevel.LIGHT, model: "gpt4", timeoutSeconds: 5 },
    });
    const api = createTtsApi(makeFakeClient({ getSummarizeConfig }));
    const cfg = await api.getTTSSummarizeConfig();
    expect(cfg.level).toBe("light");
    expect(cfg.enabled).toBe(true);
    expect(cfg.charThreshold).toBe(500);
    expect(cfg.model).toBe("gpt4");
    expect(cfg.timeoutSeconds).toBe(5);
  });

  it("decodes moderate level", async () => {
    const getSummarizeConfig = vi.fn().mockResolvedValue({
      config: { enabled: false, charThreshold: 0, level: SummarizeLevel.MODERATE, model: "", timeoutSeconds: 0 },
    });
    const api = createTtsApi(makeFakeClient({ getSummarizeConfig }));
    const cfg = await api.getTTSSummarizeConfig();
    expect(cfg.level).toBe("moderate");
  });

  it("decodes heavy level", async () => {
    const getSummarizeConfig = vi.fn().mockResolvedValue({
      config: { enabled: false, charThreshold: 0, level: SummarizeLevel.HEAVY, model: "", timeoutSeconds: 0 },
    });
    const api = createTtsApi(makeFakeClient({ getSummarizeConfig }));
    const cfg = await api.getTTSSummarizeConfig();
    expect(cfg.level).toBe("heavy");
  });

  it("defaults level to moderate when unspecified", async () => {
    const getSummarizeConfig = vi.fn().mockResolvedValue({
      config: { enabled: false, charThreshold: 0, level: SummarizeLevel.UNSPECIFIED, model: "", timeoutSeconds: 0 },
    });
    const api = createTtsApi(makeFakeClient({ getSummarizeConfig }));
    const cfg = await api.getTTSSummarizeConfig();
    expect(cfg.level).toBe("moderate");
  });

  it("defaults all fields when config is undefined", async () => {
    const getSummarizeConfig = vi.fn().mockResolvedValue({ config: undefined });
    const api = createTtsApi(makeFakeClient({ getSummarizeConfig }));
    const cfg = await api.getTTSSummarizeConfig();
    expect(cfg.enabled).toBe(false);
    expect(cfg.charThreshold).toBe(0);
    expect(cfg.level).toBe("moderate");
    expect(cfg.model).toBe("");
    expect(cfg.timeoutSeconds).toBe(0);
  });
});

describe("updateTTSSummarizeConfig", () => {
  it("builds field mask for all fields", async () => {
    const updateSummarizeConfig = vi.fn().mockResolvedValue({
      config: { enabled: true, charThreshold: 200, level: SummarizeLevel.HEAVY, model: "llama", timeoutSeconds: 10 },
    });
    const api = createTtsApi(makeFakeClient({ updateSummarizeConfig }));
    const result = await api.updateTTSSummarizeConfig({
      enabled: true,
      charThreshold: 200,
      level: "heavy",
      model: "llama",
      timeoutSeconds: 10,
    });
    const { updateMask, config } = updateSummarizeConfig.mock.calls[0]![0];
    expect(updateMask.paths).toContain("enabled");
    expect(updateMask.paths).toContain("char_threshold");
    expect(updateMask.paths).toContain("level");
    expect(updateMask.paths).toContain("model");
    expect(updateMask.paths).toContain("timeout_seconds");
    expect(config.enabled).toBe(true);
    expect(result.level).toBe("heavy");
  });

  it("omits fields not in the patch", async () => {
    const updateSummarizeConfig = vi.fn().mockResolvedValue({
      config: { enabled: false, charThreshold: 0, level: SummarizeLevel.UNSPECIFIED, model: "", timeoutSeconds: 0 },
    });
    const api = createTtsApi(makeFakeClient({ updateSummarizeConfig }));
    await api.updateTTSSummarizeConfig({ model: "phi3" });
    const paths = updateSummarizeConfig.mock.calls[0]![0].updateMask.paths;
    expect(paths).toEqual(["model"]);
  });

  it("encodes 'light' level correctly", async () => {
    const updateSummarizeConfig = vi.fn().mockResolvedValue({
      config: { enabled: false, charThreshold: 0, level: SummarizeLevel.LIGHT, model: "", timeoutSeconds: 0 },
    });
    const api = createTtsApi(makeFakeClient({ updateSummarizeConfig }));
    await api.updateTTSSummarizeConfig({ level: "light" });
    expect(updateSummarizeConfig.mock.calls[0]![0].config.level).toBe(SummarizeLevel.LIGHT);
  });
});

// ---------------------------------------------------------------------------
// reportTTSEvent
// ---------------------------------------------------------------------------

describe("reportTTSEvent", () => {
  it("sends the event to the RPC", async () => {
    const recordPlaybackEvent = vi.fn().mockResolvedValue({});
    const api = createTtsApi(makeFakeClient({ recordPlaybackEvent }));
    await api.reportTTSEvent({
      source: "tts-player",
      stage: "start",
      backend: "kokoro",
      sessionId: "sess-1",
      message: "ok",
    });
    const req = recordPlaybackEvent.mock.calls[0]![0];
    expect(req.event.source).toBe("tts-player");
    expect(req.event.stage).toBe("start");
    expect(req.event.backend).toBe("kokoro");
    expect(req.event.sessionId).toBe("sess-1");
    expect(req.event.message).toBe("ok");
  });

  it("defaults optional fields to empty strings", async () => {
    const recordPlaybackEvent = vi.fn().mockResolvedValue({});
    const api = createTtsApi(makeFakeClient({ recordPlaybackEvent }));
    await api.reportTTSEvent({ source: "s", stage: "end" });
    const { event } = recordPlaybackEvent.mock.calls[0]![0];
    expect(event.backend).toBe("");
    expect(event.sessionId).toBe("");
    expect(event.message).toBe("");
  });
});

// ---------------------------------------------------------------------------
// Module-level lazy() singleton wrappers
// ---------------------------------------------------------------------------

describe("module-level lazy() wrappers", () => {
  let client: AudioToolsClient;

  beforeEach(() => {
    client = makeFakeClient({}, "http://tts-lazy");
    setActiveAudioToolsClientForTesting(client);
  });

  afterEach(() => {
    setActiveAudioToolsClientForTesting(null);
  });

  it("reportTTSEvent() delegates through active client", async () => {
    const recordPlaybackEvent = vi.fn().mockResolvedValue({});
    client = makeFakeClient({ recordPlaybackEvent }, "http://tts-lazy");
    setActiveAudioToolsClientForTesting(client);
    await reportTTSEvent({ source: "s", stage: "e" });
    expect(recordPlaybackEvent).toHaveBeenCalledOnce();
  });

  it("synthesizeTTS() delegates through active client", async () => {
    const synthesize = vi.fn().mockResolvedValue({ audio: new Uint8Array([1]), contentType: "audio/mpeg" });
    client = makeFakeClient({ synthesize }, "http://tts-lazy");
    setActiveAudioToolsClientForTesting(client);
    const blob = await synthesizeTTS("test");
    expect(blob).toBeInstanceOf(Blob);
  });

  it("fetchCachedTTS() delegates through active client", async () => {
    const getCache = vi.fn().mockResolvedValue({ audio: new Uint8Array(0), contentType: "" });
    client = makeFakeClient({ getCache }, "http://tts-lazy");
    setActiveAudioToolsClientForTesting(client);
    const result = await fetchCachedTTS("ev", "v", 1.0);
    expect(result).toBeNull();
  });

  it("getTTSVoices() delegates", async () => {
    const listVoices = vi.fn().mockResolvedValue({ voices: [] });
    client = makeFakeClient({ listVoices }, "http://tts-lazy");
    setActiveAudioToolsClientForTesting(client);
    expect(await getTTSVoices()).toEqual([]);
  });

  it("getTTSConfig() delegates", async () => {
    const getConfig = vi.fn().mockResolvedValue({ config: undefined });
    client = makeFakeClient({ getConfig }, "http://tts-lazy");
    setActiveAudioToolsClientForTesting(client);
    const cfg = await getTTSConfig();
    expect(cfg.autoEnabled).toBe(false);
  });

  it("updateTTSConfig() delegates", async () => {
    const updateConfig = vi.fn().mockResolvedValue({
      config: { autoEnabled: false, defaultVoice: "", defaultSpeed: 1, defaultResponseFormat: ResponseFormat.UNSPECIFIED },
    });
    client = makeFakeClient({ updateConfig }, "http://tts-lazy");
    setActiveAudioToolsClientForTesting(client);
    await updateTTSConfig({ autoEnabled: false });
    expect(updateConfig).toHaveBeenCalledOnce();
  });

  it("getTTSSummarizeConfig() delegates", async () => {
    const getSummarizeConfig = vi.fn().mockResolvedValue({ config: undefined });
    client = makeFakeClient({ getSummarizeConfig }, "http://tts-lazy");
    setActiveAudioToolsClientForTesting(client);
    const cfg = await getTTSSummarizeConfig();
    expect(cfg.enabled).toBe(false);
  });

  it("updateTTSSummarizeConfig() delegates", async () => {
    const updateSummarizeConfig = vi.fn().mockResolvedValue({
      config: { enabled: false, charThreshold: 0, level: SummarizeLevel.UNSPECIFIED, model: "", timeoutSeconds: 0 },
    });
    client = makeFakeClient({ updateSummarizeConfig }, "http://tts-lazy");
    setActiveAudioToolsClientForTesting(client);
    await updateTTSSummarizeConfig({ enabled: false });
    expect(updateSummarizeConfig).toHaveBeenCalledOnce();
  });

  it("rebinds lazy api when client identity changes", async () => {
    const listVoices1 = vi.fn().mockResolvedValue({ voices: [{ id: "v1", name: "First" }] });
    const client1 = makeFakeClient({ listVoices: listVoices1 }, "http://tts-lazy");
    setActiveAudioToolsClientForTesting(client1);
    await getTTSVoices();
    expect(listVoices1).toHaveBeenCalledOnce();

    // Switch to a different client
    const listVoices2 = vi.fn().mockResolvedValue({ voices: [] });
    const client2 = makeFakeClient({ listVoices: listVoices2 }, "http://tts-lazy-2");
    setActiveAudioToolsClientForTesting(client2);
    await getTTSVoices();
    expect(listVoices2).toHaveBeenCalledOnce();
  });
});
