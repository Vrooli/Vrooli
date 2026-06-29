// Additional coverage for voice.ts — covering everything NOT touched by voice.test.ts:
//   buildVoiceStreamWsUrl, transcribeAudio/BypassFilter/WithRetry,
//   wake-word API, speaker-verification API, blobFormat branches,
//   and the module-level lazy() singleton wrappers.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { SpeakerMode, RejectBehavior } from "@vrooli/proto-types/audio-tools/v1/stt/stt_pb";

import { createVoiceApi } from "./voice";
import { setActiveAudioToolsClientForTesting } from "../client";
import type { AudioToolsClient } from "../client";
import {
  buildVoiceStreamWsUrl,
  transcribeAudio,
  transcribeAudioBypassFilter,
  transcribeAudioWithRetry,
  getWakeWordConfig,
  updateWakeWordConfig,
  deleteWakeWordConfig,
  getSpeakerVerificationConfig,
  updateSpeakerVerificationConfig,
  getSpeakerVerificationStatus,
  listSpeakerVerificationProfiles,
  enrollSpeakerVerificationProfile,
  clearSpeakerVerificationProfile,
  removeSpeakerVerificationProfile,
  deleteSpeakerVerificationProfile,
} from "./voice";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type FnMap = Partial<Record<string, ReturnType<typeof vi.fn>>>;

function makeFakeClient(fns: FnMap = {}, baseUrl = "http://test"): AudioToolsClient {
  const f = (name: string) => fns[name] ?? vi.fn();
  return {
    baseUrl,
    stt: { transcribe: f("transcribe") } as never,
    sttAdmin: {
      getStreamConfig: f("getStreamConfig"),
      updateStreamConfig: f("updateStreamConfig"),
      getWakeWordConfig: f("getWakeWordConfig"),
      updateWakeWordTemplate: f("updateWakeWordTemplate"),
      deleteWakeWordTemplate: f("deleteWakeWordTemplate"),
      getSpeakerConfig: f("getSpeakerConfig"),
      updateSpeakerConfig: f("updateSpeakerConfig"),
      getSpeakerStatus: f("getSpeakerStatus"),
      listSpeakerProfiles: f("listSpeakerProfiles"),
      enrollSpeakerProfile: f("enrollSpeakerProfile"),
      clearSpeakerProfileBinding: f("clearSpeakerProfileBinding"),
      unbindSpeakerProfile: f("unbindSpeakerProfile"),
      deleteSpeakerProfile: f("deleteSpeakerProfile"),
    } as never,
    tts: {} as never,
    summarize: {} as never,
  };
}

// jsdom's Blob lacks arrayBuffer() — return a plain object that satisfies the
// blobToBytes() call (b.arrayBuffer()) and blobFormat() call (b.type).
function makeBlob(type: string, bytes: number[] = [1, 2, 3]): Blob {
  const buf = new Uint8Array(bytes);
  return {
    type,
    size: buf.length,
    arrayBuffer: () => Promise.resolve(buf.buffer),
    text: () => Promise.resolve(""),
    stream: () => new ReadableStream(),
    slice: (_s?: number, _e?: number, _t?: string) => makeBlob(type, []),
  } as unknown as Blob;
}

// Minimal speaker profile the proto side might return
const protoProfile = {
  id: "p1",
  displayName: "Alice",
  createdAt: { seconds: 1_700_000_000n, nanos: 0 },
  updatedAt: { seconds: 1_700_000_001n, nanos: 0 },
  modelName: "ecapa",
  embeddingDim: 192,
  sampleRate: 16_000,
  clipCount: 3,
  totalVoicedSeconds: 4.5,
  notes: "clear voice",
};

const protoSpeakerConfig = {
  enabled: true,
  profileIds: ["p1"],
  threshold: 0.7,
  mode: SpeakerMode.FILTER,
  rejectBehavior: RejectBehavior.DROP,
  fallbackWithoutVerification: false,
};

// ---------------------------------------------------------------------------
// buildVoiceStreamWsUrl
// ---------------------------------------------------------------------------

describe("buildVoiceStreamWsUrl", () => {
  it("converts http:// base URL to ws://", () => {
    const api = createVoiceApi(makeFakeClient({}, "http://localhost:8080"));
    const url = api.buildVoiceStreamWsUrl();
    expect(url).toMatch(/^ws:\/\/localhost:8080\//);
    expect(url).toContain("format=pcm_s16le");
  });

  it("converts https:// base URL to wss://", () => {
    const api = createVoiceApi(makeFakeClient({}, "https://api.example.com"));
    const url = api.buildVoiceStreamWsUrl();
    expect(url).toMatch(/^wss:\/\/api\.example\.com\//);
  });

  it("appends language parameter when provided", () => {
    const api = createVoiceApi(makeFakeClient({}, "http://localhost:8080"));
    const url = api.buildVoiceStreamWsUrl("en");
    expect(url).toContain("language=en");
    expect(url).toContain("format=pcm_s16le");
  });

  it("omits language parameter when not provided", () => {
    const api = createVoiceApi(makeFakeClient({}, "http://localhost:8080"));
    const url = api.buildVoiceStreamWsUrl();
    expect(url).not.toContain("language=");
  });

  it("strips trailing slash from base URL", () => {
    const api = createVoiceApi(makeFakeClient({}, "http://localhost:8080/"));
    const url = api.buildVoiceStreamWsUrl();
    expect(url).not.toContain("//api");
    expect(url).toMatch(/^ws:\/\/localhost:8080\//);
  });

  it("passes through non-http/https schemes unchanged", () => {
    const api = createVoiceApi(makeFakeClient({}, "custom://endpoint"));
    const url = api.buildVoiceStreamWsUrl();
    expect(url).toMatch(/^custom:\/\//);
  });
});

// ---------------------------------------------------------------------------
// transcribeAudio — blobFormat branches
// ---------------------------------------------------------------------------

describe("transcribeAudio", () => {
  it("transcribes webm audio and returns text", async () => {
    const transcribe = vi.fn().mockResolvedValue({ text: "hello" });
    const api = createVoiceApi(makeFakeClient({ transcribe }));
    const text = await api.transcribeAudio(makeBlob("audio/webm"));
    expect(text).toBe("hello");
    const req = transcribe.mock.calls[0]![0];
    expect(req.skipSpeakerVerification).toBe(false);
  });

  it("handles wav mime type", async () => {
    const transcribe = vi.fn().mockResolvedValue({ text: "wav" });
    const api = createVoiceApi(makeFakeClient({ transcribe }));
    await api.transcribeAudio(makeBlob("audio/wav"));
    expect(transcribe).toHaveBeenCalledTimes(1);
  });

  it("handles mp3 mime type (audio/mp3)", async () => {
    const transcribe = vi.fn().mockResolvedValue({ text: "mp3" });
    const api = createVoiceApi(makeFakeClient({ transcribe }));
    await api.transcribeAudio(makeBlob("audio/mp3"));
    expect(transcribe).toHaveBeenCalledTimes(1);
  });

  it("handles mpeg mime type (audio/mpeg)", async () => {
    const transcribe = vi.fn().mockResolvedValue({ text: "mpeg" });
    const api = createVoiceApi(makeFakeClient({ transcribe }));
    await api.transcribeAudio(makeBlob("audio/mpeg"));
    expect(transcribe).toHaveBeenCalledTimes(1);
  });

  it("handles ogg mime type", async () => {
    const transcribe = vi.fn().mockResolvedValue({ text: "ogg" });
    const api = createVoiceApi(makeFakeClient({ transcribe }));
    await api.transcribeAudio(makeBlob("audio/ogg"));
    expect(transcribe).toHaveBeenCalledTimes(1);
  });

  it("handles flac mime type", async () => {
    const transcribe = vi.fn().mockResolvedValue({ text: "flac" });
    const api = createVoiceApi(makeFakeClient({ transcribe }));
    await api.transcribeAudio(makeBlob("audio/flac"));
    expect(transcribe).toHaveBeenCalledTimes(1);
  });

  it("falls back to extracting subtype from unknown mime", async () => {
    const transcribe = vi.fn().mockResolvedValue({ text: "custom" });
    const api = createVoiceApi(makeFakeClient({ transcribe }));
    await api.transcribeAudio(makeBlob("audio/x-custom"));
    expect(transcribe).toHaveBeenCalledTimes(1);
  });

  it("falls back to webm when mime is empty", async () => {
    const transcribe = vi.fn().mockResolvedValue({ text: "" });
    const api = createVoiceApi(makeFakeClient({ transcribe }));
    await api.transcribeAudio(makeBlob(""));
    expect(transcribe).toHaveBeenCalledTimes(1);
  });

  it("passes language when provided", async () => {
    const transcribe = vi.fn().mockResolvedValue({ text: "bonjour" });
    const api = createVoiceApi(makeFakeClient({ transcribe }));
    await api.transcribeAudio(makeBlob("audio/webm"), "fr");
    expect(transcribe.mock.calls[0]![0].language).toBe("fr");
  });

  it("defaults language to empty string", async () => {
    const transcribe = vi.fn().mockResolvedValue({ text: "" });
    const api = createVoiceApi(makeFakeClient({ transcribe }));
    await api.transcribeAudio(makeBlob("audio/webm"));
    expect(transcribe.mock.calls[0]![0].language).toBe("");
  });
});

// ---------------------------------------------------------------------------
// transcribeAudioBypassFilter
// ---------------------------------------------------------------------------

describe("transcribeAudioBypassFilter", () => {
  it("sets skipSpeakerVerification=true", async () => {
    const transcribe = vi.fn().mockResolvedValue({ text: "bypass" });
    const api = createVoiceApi(makeFakeClient({ transcribe }));
    const text = await api.transcribeAudioBypassFilter(makeBlob("audio/webm"), "en");
    expect(text).toBe("bypass");
    const req = transcribe.mock.calls[0]![0];
    expect(req.skipSpeakerVerification).toBe(true);
    expect(req.language).toBe("en");
  });
});

// ---------------------------------------------------------------------------
// transcribeAudioWithRetry
// ---------------------------------------------------------------------------

describe("transcribeAudioWithRetry", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("returns on first success without delay", async () => {
    const transcribe = vi.fn().mockResolvedValue({ text: "first" });
    const api = createVoiceApi(makeFakeClient({ transcribe }));
    const p = api.transcribeAudioWithRetry(makeBlob("audio/webm"));
    await vi.runAllTimersAsync();
    expect(await p).toBe("first");
    expect(transcribe).toHaveBeenCalledTimes(1);
  });

  it("retries once and succeeds on second attempt", async () => {
    const transcribe = vi.fn()
      .mockRejectedValueOnce(new Error("transient"))
      .mockResolvedValueOnce({ text: "second" });
    const api = createVoiceApi(makeFakeClient({ transcribe }));
    const p = api.transcribeAudioWithRetry(makeBlob("audio/webm"), 2);
    await vi.runAllTimersAsync();
    expect(await p).toBe("second");
    expect(transcribe).toHaveBeenCalledTimes(2);
  });

  it("throws after exhausting all attempts", async () => {
    const err = new Error("permanent failure");
    const transcribe = vi.fn().mockRejectedValue(err);
    const api = createVoiceApi(makeFakeClient({ transcribe }));
    let caught: unknown;
    const p = api.transcribeAudioWithRetry(makeBlob("audio/webm"), 3).catch((e: unknown) => { caught = e; });
    await vi.runAllTimersAsync();
    await p;
    expect((caught as Error).message).toBe("permanent failure");
    expect(transcribe).toHaveBeenCalledTimes(3);
  });

  it("passes language through to each attempt", async () => {
    const transcribe = vi.fn()
      .mockRejectedValueOnce(new Error("fail"))
      .mockResolvedValueOnce({ text: "ok" });
    const api = createVoiceApi(makeFakeClient({ transcribe }));
    const p = api.transcribeAudioWithRetry(makeBlob("audio/webm"), 2, "de");
    await vi.runAllTimersAsync();
    await p;
    expect(transcribe.mock.calls.every((c) => c[0].language === "de")).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// updateVoiceStreamConfig — basic fields (not covered by voice.test.ts)
// ---------------------------------------------------------------------------

describe("updateVoiceStreamConfig — basic fields", () => {
  it("builds field mask for all basic stream config fields", async () => {
    const updateStreamConfig = vi.fn().mockResolvedValue({ config: {} });
    const api = createVoiceApi(makeFakeClient({ updateStreamConfig }));
    await api.updateVoiceStreamConfig({
      flushIntervalMs: 250,
      minDeltaBytes: 1024,
      overlapBytes: 512,
      persistentMode: true,
      wakeWordEnabled: true,
      wakeWordThreshold: 0.8,
      segmentSilenceMs: 500,
    });
    const paths: string[] = updateStreamConfig.mock.calls[0]![0].updateMask.paths;
    expect(paths).toContain("flush_interval_ms");
    expect(paths).toContain("min_delta_bytes");
    expect(paths).toContain("overlap_bytes");
    expect(paths).toContain("persistent_mode");
    expect(paths).toContain("wake_word_enabled");
    expect(paths).toContain("wake_word_threshold");
    expect(paths).toContain("segment_silence_ms");
  });

  it("encodes field values correctly", async () => {
    const updateStreamConfig = vi.fn().mockResolvedValue({ config: {} });
    const api = createVoiceApi(makeFakeClient({ updateStreamConfig }));
    await api.updateVoiceStreamConfig({
      flushIntervalMs: 200,
      persistentMode: false,
    });
    const cfg = updateStreamConfig.mock.calls[0]![0].config;
    expect(cfg.flushIntervalMs).toBe(200);
    expect(cfg.persistentMode).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// Wake-word API
// ---------------------------------------------------------------------------

describe("getWakeWordConfig", () => {
  it("returns unconfigured when configured=false", async () => {
    const getWakeWordConfig = vi.fn().mockResolvedValue({ config: { configured: false } });
    const api = createVoiceApi(makeFakeClient({ getWakeWordConfig }));
    const result = await api.getWakeWordConfig();
    expect(result.configured).toBe(false);
    expect(result.template).toBeNull();
  });

  it("returns template when configured=true and template present", async () => {
    const getWakeWordConfig = vi.fn().mockResolvedValue({
      config: {
        configured: true,
        template: {
          label: "hey vrooli",
          threshold: 0.8,
          samples: [{ audio: new Uint8Array([1]), format: "wav", sampleRateHz: 16_000 }],
          updatedAt: { seconds: 1_700_000_000n, nanos: 0 },
        },
      },
    });
    const api = createVoiceApi(makeFakeClient({ getWakeWordConfig }));
    const result = await api.getWakeWordConfig();
    expect(result.configured).toBe(true);
    expect(result.template?.label).toBe("hey vrooli");
    expect(result.template?.threshold).toBe(0.8);
    expect(result.template?.samples).toHaveLength(1);
    expect(result.template?.updatedAt).toContain("T");
  });

  it("returns null template when configured=true but no template object", async () => {
    const getWakeWordConfig = vi.fn().mockResolvedValue({
      config: { configured: true, template: undefined },
    });
    const api = createVoiceApi(makeFakeClient({ getWakeWordConfig }));
    const result = await api.getWakeWordConfig();
    expect(result.configured).toBe(true);
    expect(result.template).toBeNull();
  });

  it("defaults configured=false when config is undefined", async () => {
    const getWakeWordConfig = vi.fn().mockResolvedValue({ config: undefined });
    const api = createVoiceApi(makeFakeClient({ getWakeWordConfig }));
    const result = await api.getWakeWordConfig();
    expect(result.configured).toBe(false);
    expect(result.template).toBeNull();
  });
});

describe("updateWakeWordConfig", () => {
  it("passes the template and returns decoded config", async () => {
    const updateWakeWordTemplate = vi.fn().mockResolvedValue({
      config: { configured: true, template: { label: "test", threshold: 0.9, samples: [], updatedAt: undefined } },
    });
    const api = createVoiceApi(makeFakeClient({ updateWakeWordTemplate }));
    const template = { label: "test", threshold: 0.9, samples: [] as never[], updatedAt: "" };
    const result = await api.updateWakeWordConfig(template);
    expect(updateWakeWordTemplate).toHaveBeenCalledTimes(1);
    expect(result.configured).toBe(true);
  });
});

describe("deleteWakeWordConfig", () => {
  it("calls deleteWakeWordTemplate and returns decoded config", async () => {
    const deleteWakeWordTemplate = vi.fn().mockResolvedValue({
      config: { configured: false },
    });
    const api = createVoiceApi(makeFakeClient({ deleteWakeWordTemplate }));
    const result = await api.deleteWakeWordConfig();
    expect(deleteWakeWordTemplate).toHaveBeenCalledTimes(1);
    expect(result.configured).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// Speaker-verification API
// ---------------------------------------------------------------------------

describe("getSpeakerVerificationConfig", () => {
  it("decodes the config", async () => {
    const getSpeakerConfig = vi.fn().mockResolvedValue({ config: protoSpeakerConfig });
    const api = createVoiceApi(makeFakeClient({ getSpeakerConfig }));
    const cfg = await api.getSpeakerVerificationConfig();
    expect(cfg.enabled).toBe(true);
    expect(cfg.profileIds).toEqual(["p1"]);
    expect(cfg.threshold).toBe(0.7);
    expect(cfg.mode).toBe("filter");
    expect(cfg.rejectBehavior).toBe("drop");
    expect(cfg.fallbackWithoutVerification).toBe(false);
  });

  it("defaults when config is undefined", async () => {
    const getSpeakerConfig = vi.fn().mockResolvedValue({ config: undefined });
    const api = createVoiceApi(makeFakeClient({ getSpeakerConfig }));
    const cfg = await api.getSpeakerVerificationConfig();
    expect(cfg.enabled).toBe(false);
    expect(cfg.profileIds).toEqual([]);
    expect(cfg.threshold).toBe(0);
    expect(cfg.mode).toBe("filter"); // default
    expect(cfg.rejectBehavior).toBe("drop"); // default
  });
});

describe("updateSpeakerVerificationConfig", () => {
  it("builds field mask for all provided fields", async () => {
    const updateSpeakerConfig = vi.fn().mockResolvedValue({ config: protoSpeakerConfig });
    const api = createVoiceApi(makeFakeClient({ updateSpeakerConfig }));
    await api.updateSpeakerVerificationConfig({
      enabled: true,
      profileIds: ["p1"],
      threshold: 0.8,
      mode: "advisory",
      rejectBehavior: "show-muted",
      fallbackWithoutVerification: true,
    });
    const callArg = updateSpeakerConfig.mock.calls[0]![0];
    const paths: string[] = callArg.updateMask.paths;
    expect(paths).toContain("enabled");
    expect(paths).toContain("profile_ids");
    expect(paths).toContain("threshold");
    expect(paths).toContain("mode");
    expect(paths).toContain("reject_behavior");
    expect(paths).toContain("fallback_without_verification");
  });

  it("omits paths for fields not in the patch", async () => {
    const updateSpeakerConfig = vi.fn().mockResolvedValue({ config: protoSpeakerConfig });
    const api = createVoiceApi(makeFakeClient({ updateSpeakerConfig }));
    await api.updateSpeakerVerificationConfig({ enabled: false });
    const paths: string[] = updateSpeakerConfig.mock.calls[0]![0].updateMask.paths;
    expect(paths).toEqual(["enabled"]);
  });
});

describe("getSpeakerVerificationStatus", () => {
  it("decodes a full status response", async () => {
    const getSpeakerStatus = vi.fn().mockResolvedValue({
      status: {
        config: protoSpeakerConfig,
        capability: "full",
        capabilityLabel: "Full",
        resourceReady: true,
        profileConfigured: true,
        profileExists: true,
        profileCount: 1,
        profiles: [protoProfile],
        info: {
          backend: "speechbrain",
          model: "ecapa",
          device: "cpu",
          sampleRate: 16_000,
          version: "1.0",
          embeddingDim: 192,
        },
        checkedAt: { seconds: 1_700_000_000n, nanos: 0 },
      },
    });
    const api = createVoiceApi(makeFakeClient({ getSpeakerStatus }));
    const st = await api.getSpeakerVerificationStatus();
    expect(st.capability).toBe("full");
    expect(st.capabilityLabel).toBe("Full");
    expect(st.resourceReady).toBe(true);
    expect(st.profileCount).toBe(1);
    expect(st.profiles).toHaveLength(1);
    expect(st.profiles![0]!.id).toBe("p1");
    expect(st.profiles![0]!.display_name).toBe("Alice");
    expect(st.info?.backend).toBe("speechbrain");
    expect(st.info?.sample_rate).toBe(16_000);
    expect(st.checkedAt).toContain("T");
  });

  it("returns undefined info when absent", async () => {
    const getSpeakerStatus = vi.fn().mockResolvedValue({
      status: {
        config: protoSpeakerConfig,
        capability: "none",
        capabilityLabel: "",
        resourceReady: false,
        profileConfigured: false,
        profileExists: false,
        profileCount: 0,
        profiles: [],
        info: undefined,
        checkedAt: undefined,
      },
    });
    const api = createVoiceApi(makeFakeClient({ getSpeakerStatus }));
    const st = await api.getSpeakerVerificationStatus();
    expect(st.info).toBeUndefined();
    expect(st.capabilityLabel).toBeUndefined();
  });

  it("throws when status field is missing", async () => {
    const getSpeakerStatus = vi.fn().mockResolvedValue({});
    const api = createVoiceApi(makeFakeClient({ getSpeakerStatus }));
    await expect(api.getSpeakerVerificationStatus()).rejects.toThrow("missing status field");
  });
});

describe("listSpeakerVerificationProfiles", () => {
  it("decodes profiles", async () => {
    const listSpeakerProfiles = vi.fn().mockResolvedValue({ profiles: [protoProfile] });
    const api = createVoiceApi(makeFakeClient({ listSpeakerProfiles }));
    const profiles = await api.listSpeakerVerificationProfiles();
    expect(profiles).toHaveLength(1);
    expect(profiles[0]!.id).toBe("p1");
    expect(profiles[0]!.clip_count).toBe(3);
    expect(profiles[0]!.total_voiced_seconds).toBe(4.5);
    expect(profiles[0]!.created_at).toContain("T");
    expect(profiles[0]!.updated_at).toContain("T");
  });

  it("returns empty array when no profiles", async () => {
    const listSpeakerProfiles = vi.fn().mockResolvedValue({ profiles: [] });
    const api = createVoiceApi(makeFakeClient({ listSpeakerProfiles }));
    expect(await api.listSpeakerVerificationProfiles()).toEqual([]);
  });
});

describe("enrollSpeakerVerificationProfile", () => {
  const protoEnrollment = {
    profileId: "p2",
    clipId: "c1",
    label: "clip-1",
    voicedSeconds: 1.5,
    clipCount: 1,
    totalVoicedSeconds: 1.5,
    embeddingDim: 192,
    sampleRate: 16_000,
    modelName: "ecapa",
    createdAt: { seconds: 1_700_000_000n, nanos: 0 },
  };

  it("encodes audio bytes and decodes enrollment result", async () => {
    const enrollSpeakerProfile = vi.fn().mockResolvedValue({
      enrollment: protoEnrollment,
      config: protoSpeakerConfig,
    });
    const api = createVoiceApi(makeFakeClient({ enrollSpeakerProfile }));
    const result = await api.enrollSpeakerVerificationProfile({
      audioBlob: makeBlob("audio/wav"),
    });
    expect(result.enrollment.profile_id).toBe("p2");
    expect(result.enrollment.clip_count).toBe(1);
    expect(result.enrollment.created_at).toContain("T");
    expect(result.config.enabled).toBe(true);
  });

  it("passes optional fields when provided", async () => {
    const enrollSpeakerProfile = vi.fn().mockResolvedValue({
      enrollment: protoEnrollment,
      config: protoSpeakerConfig,
    });
    const api = createVoiceApi(makeFakeClient({ enrollSpeakerProfile }));
    await api.enrollSpeakerVerificationProfile({
      audioBlob: makeBlob("audio/wav"),
      profileId: "existing-p",
      displayName: "Bob",
      notes: "test",
      label: "intro",
      addToActive: true,
      enable: true,
    });
    const req = enrollSpeakerProfile.mock.calls[0]![0];
    expect(req.profileId).toBe("existing-p");
    expect(req.displayName).toBe("Bob");
    expect(req.addToActive).toBe(true);
    expect(req.enable).toBe(true);
  });

  it("handles missing enrollment gracefully", async () => {
    const enrollSpeakerProfile = vi.fn().mockResolvedValue({
      enrollment: undefined,
      config: protoSpeakerConfig,
    });
    const api = createVoiceApi(makeFakeClient({ enrollSpeakerProfile }));
    const result = await api.enrollSpeakerVerificationProfile({
      audioBlob: makeBlob("audio/wav"),
    });
    expect(result.enrollment.profile_id).toBe("");
    expect(result.enrollment.voiced_seconds).toBe(0);
  });
});

describe("clearSpeakerVerificationProfile", () => {
  it("calls clearSpeakerProfileBinding and decodes result", async () => {
    const clearSpeakerProfileBinding = vi.fn().mockResolvedValue({ config: protoSpeakerConfig });
    const api = createVoiceApi(makeFakeClient({ clearSpeakerProfileBinding }));
    const result = await api.clearSpeakerVerificationProfile();
    expect(clearSpeakerProfileBinding).toHaveBeenCalledOnce();
    expect(result.enabled).toBe(true);
  });
});

describe("removeSpeakerVerificationProfile", () => {
  it("unbinds by profile ID", async () => {
    const unbindSpeakerProfile = vi.fn().mockResolvedValue({ config: protoSpeakerConfig });
    const api = createVoiceApi(makeFakeClient({ unbindSpeakerProfile }));
    await api.removeSpeakerVerificationProfile("p1");
    expect(unbindSpeakerProfile).toHaveBeenCalledWith({ profileId: "p1" });
  });
});

describe("deleteSpeakerVerificationProfile", () => {
  it("deletes by profile ID", async () => {
    const deleteSpeakerProfile = vi.fn().mockResolvedValue({ config: protoSpeakerConfig });
    const api = createVoiceApi(makeFakeClient({ deleteSpeakerProfile }));
    await api.deleteSpeakerVerificationProfile("p99");
    expect(deleteSpeakerProfile).toHaveBeenCalledWith({ profileId: "p99" });
  });
});

// ---------------------------------------------------------------------------
// Module-level lazy() singleton wrappers
// ---------------------------------------------------------------------------

describe("module-level lazy() singleton", () => {
  let client: AudioToolsClient;

  beforeEach(() => {
    client = makeFakeClient({}, "http://lazy-test");
    setActiveAudioToolsClientForTesting(client);
  });

  afterEach(() => {
    setActiveAudioToolsClientForTesting(null);
  });

  it("buildVoiceStreamWsUrl() delegates through lazy() to the active client", () => {
    const url = buildVoiceStreamWsUrl();
    expect(url).toMatch(/^ws:\/\/lazy-test\//);
  });

  it("buildVoiceStreamWsUrl() passes language", () => {
    const url = buildVoiceStreamWsUrl("ja");
    expect(url).toContain("language=ja");
  });

  it("transcribeAudio() delegates to active client", async () => {
    const transcribe = vi.fn().mockResolvedValue({ text: "lazy-transcribe" });
    client = makeFakeClient({ transcribe }, "http://lazy-test");
    setActiveAudioToolsClientForTesting(client);
    const text = await transcribeAudio(makeBlob("audio/webm"));
    expect(text).toBe("lazy-transcribe");
  });

  it("transcribeAudioBypassFilter() delegates", async () => {
    const transcribe = vi.fn().mockResolvedValue({ text: "bypass-lazy" });
    client = makeFakeClient({ transcribe }, "http://lazy-test");
    setActiveAudioToolsClientForTesting(client);
    const text = await transcribeAudioBypassFilter(makeBlob("audio/webm"));
    expect(text).toBe("bypass-lazy");
  });

  it("transcribeAudioWithRetry() delegates", async () => {
    const transcribe = vi.fn().mockResolvedValue({ text: "retry-lazy" });
    client = makeFakeClient({ transcribe }, "http://lazy-test");
    setActiveAudioToolsClientForTesting(client);
    const text = await transcribeAudioWithRetry(makeBlob("audio/webm"));
    expect(text).toBe("retry-lazy");
  });

  it("getWakeWordConfig() delegates", async () => {
    const getWakeWordConfigFn = vi.fn().mockResolvedValue({ config: { configured: false } });
    client = makeFakeClient({ getWakeWordConfig: getWakeWordConfigFn }, "http://lazy-test");
    setActiveAudioToolsClientForTesting(client);
    const cfg = await getWakeWordConfig();
    expect(cfg.configured).toBe(false);
  });

  it("updateWakeWordConfig() delegates", async () => {
    const updateWakeWordTemplate = vi.fn().mockResolvedValue({ config: { configured: true } });
    client = makeFakeClient({ updateWakeWordTemplate }, "http://lazy-test");
    setActiveAudioToolsClientForTesting(client);
    const cfg = await updateWakeWordConfig({ label: "", threshold: 0, samples: [] as never[], updatedAt: "" });
    expect(cfg.configured).toBe(true);
  });

  it("deleteWakeWordConfig() delegates", async () => {
    const deleteWakeWordTemplate = vi.fn().mockResolvedValue({ config: { configured: false } });
    client = makeFakeClient({ deleteWakeWordTemplate }, "http://lazy-test");
    setActiveAudioToolsClientForTesting(client);
    const cfg = await deleteWakeWordConfig();
    expect(cfg.configured).toBe(false);
  });

  it("getSpeakerVerificationConfig() delegates", async () => {
    const getSpeakerConfig = vi.fn().mockResolvedValue({ config: undefined });
    client = makeFakeClient({ getSpeakerConfig }, "http://lazy-test");
    setActiveAudioToolsClientForTesting(client);
    const cfg = await getSpeakerVerificationConfig();
    expect(cfg.enabled).toBe(false);
  });

  it("updateSpeakerVerificationConfig() delegates", async () => {
    const updateSpeakerConfig = vi.fn().mockResolvedValue({ config: protoSpeakerConfig });
    client = makeFakeClient({ updateSpeakerConfig }, "http://lazy-test");
    setActiveAudioToolsClientForTesting(client);
    await updateSpeakerVerificationConfig({ enabled: true });
    expect(updateSpeakerConfig).toHaveBeenCalledOnce();
  });

  it("getSpeakerVerificationStatus() delegates", async () => {
    const getSpeakerStatus = vi.fn().mockResolvedValue({
      status: {
        config: protoSpeakerConfig,
        capability: "full",
        resourceReady: true,
        profileConfigured: true,
        profileExists: true,
        profileCount: 0,
        profiles: [],
        checkedAt: undefined,
      },
    });
    client = makeFakeClient({ getSpeakerStatus }, "http://lazy-test");
    setActiveAudioToolsClientForTesting(client);
    const st = await getSpeakerVerificationStatus();
    expect(st.capability).toBe("full");
  });

  it("listSpeakerVerificationProfiles() delegates", async () => {
    const listSpeakerProfiles = vi.fn().mockResolvedValue({ profiles: [] });
    client = makeFakeClient({ listSpeakerProfiles }, "http://lazy-test");
    setActiveAudioToolsClientForTesting(client);
    const result = await listSpeakerVerificationProfiles();
    expect(result).toEqual([]);
  });

  it("enrollSpeakerVerificationProfile() delegates", async () => {
    const enrollment = { profileId: "", clipId: "", label: "", voicedSeconds: 0, clipCount: 0, totalVoicedSeconds: 0, embeddingDim: 0, sampleRate: 0, modelName: "", createdAt: undefined };
    const enrollSpeakerProfile = vi.fn().mockResolvedValue({ enrollment, config: protoSpeakerConfig });
    client = makeFakeClient({ enrollSpeakerProfile }, "http://lazy-test");
    setActiveAudioToolsClientForTesting(client);
    const result = await enrollSpeakerVerificationProfile({ audioBlob: makeBlob("audio/webm") });
    expect(result.enrollment.profile_id).toBe("");
  });

  it("clearSpeakerVerificationProfile() delegates", async () => {
    const clearSpeakerProfileBinding = vi.fn().mockResolvedValue({ config: protoSpeakerConfig });
    client = makeFakeClient({ clearSpeakerProfileBinding }, "http://lazy-test");
    setActiveAudioToolsClientForTesting(client);
    await clearSpeakerVerificationProfile();
    expect(clearSpeakerProfileBinding).toHaveBeenCalledOnce();
  });

  it("removeSpeakerVerificationProfile() delegates", async () => {
    const unbindSpeakerProfile = vi.fn().mockResolvedValue({ config: protoSpeakerConfig });
    client = makeFakeClient({ unbindSpeakerProfile }, "http://lazy-test");
    setActiveAudioToolsClientForTesting(client);
    await removeSpeakerVerificationProfile("p-id");
    expect(unbindSpeakerProfile).toHaveBeenCalledWith({ profileId: "p-id" });
  });

  it("deleteSpeakerVerificationProfile() delegates", async () => {
    const deleteSpeakerProfile = vi.fn().mockResolvedValue({ config: protoSpeakerConfig });
    client = makeFakeClient({ deleteSpeakerProfile }, "http://lazy-test");
    setActiveAudioToolsClientForTesting(client);
    await deleteSpeakerVerificationProfile("del-id");
    expect(deleteSpeakerProfile).toHaveBeenCalledWith({ profileId: "del-id" });
  });

  it("rebinds lazy api when client identity changes", () => {
    // First call with client1
    const url1 = buildVoiceStreamWsUrl();
    expect(url1).toMatch(/^ws:\/\/lazy-test\//);

    // Switch to a different client
    const client2 = makeFakeClient({}, "http://other-host");
    setActiveAudioToolsClientForTesting(client2);

    const url2 = buildVoiceStreamWsUrl();
    expect(url2).toMatch(/^ws:\/\/other-host\//);
  });
});
