// Unit tests for the StreamConfig codec: decodeStreamConfig must surface
// the five advanced fields (streamingMode, strategyPreference, vadSilenceMs,
// overlapWindowMs, overlapCommitRuns), and updateVoiceStreamConfig must
// build a FieldMask that includes every patched advanced path.
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { toBinary } from "@bufbuild/protobuf";
import { streamingModeLabel, strategyPreferenceLabel, } from "@vrooli/audio-capture-browser";
import { AudioFormat, StreamingMode, StrategyPreference, } from "@vrooli/proto-types/web-console/v1/audio_common/audio_common_pb";
import { WakeWordTemplateSchema, } from "@vrooli/proto-types/web-console/v1/audio_admin/audio_admin_pb";
const initialLocation = globalThis.location.href;
vi.mock("../../api/client", () => ({
    transport: {},
    API_BASE: "http://test",
}));
function requireDefined(value, message) {
    if (value === undefined)
        throw new Error(message);
    return value;
}
// jsdom's Blob doesn't implement arrayBuffer(); blobToBytes/blobFormat only
// need .arrayBuffer() and .type, so a minimal stand-in is enough here.
function fakeBlob(bytes, type) {
    return {
        type,
        size: bytes.byteLength,
        arrayBuffer: async () => bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength),
    };
}
const updateMock = vi.fn();
const getMock = vi.fn();
const wwUpdateMock = vi.fn();
const wwGetMock = vi.fn();
vi.mock("@connectrpc/connect", async (importOriginal) => {
    const actual = await importOriginal();
    return {
        ...actual,
        createClient: () => ({
            getStreamConfig: getMock,
            updateStreamConfig: updateMock,
            getWakeWordConfig: wwGetMock,
            updateWakeWordTemplate: wwUpdateMock,
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
describe("buildVoiceStreamWsUrl", () => {
    afterEach(() => {
        globalThis.history.replaceState({}, "", initialLocation);
        vi.resetModules();
    });
    it("forwards an explicitly armed page fault through the same-origin proxy", async () => {
        globalThis.history.replaceState({}, "", "/?stt_test_mode=1&stt_test_fault=suppress_processed_ack");
        const { buildVoiceStreamWsUrl } = await import("./voice");
        const url = new URL(buildVoiceStreamWsUrl("en", "session-1", "resume-1"));
        expect(url.searchParams.get("test_mode")).toBe("1");
        expect(url.searchParams.get("test_fault")).toBe("suppress_processed_ack");
        expect(url.searchParams.get("protocol_version")).toBe("2");
        expect(url.searchParams.get("session_id")).toBe("session-1");
    });
    it("does not forward a fault unless page test mode is explicitly armed", async () => {
        globalThis.history.replaceState({}, "", "/?stt_test_fault=suppress_processed_ack");
        const { buildVoiceStreamWsUrl } = await import("./voice");
        const url = new URL(buildVoiceStreamWsUrl());
        expect(url.searchParams.has("test_mode")).toBe(false);
        expect(url.searchParams.has("test_fault")).toBe(false);
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
        const paths = callArg.updateMask.paths;
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
        const paths = callArg.updateMask.paths;
        expect(paths).toEqual(["vad_silence_ms"]);
    });
});
describe("updateWakeWordConfig", () => {
    beforeEach(() => {
        wwUpdateMock.mockReset();
        wwGetMock.mockReset();
    });
    afterEach(() => {
        vi.resetModules();
    });
    it("builds a proto request from raw blobs that encodes without throwing", async () => {
        // Regression for `[internal] Invalid time value`: the old code force-cast
        // an ISO-string `updatedAt` onto the Timestamp field, and protobuf-es threw
        // while encoding. Here we build the message the production way and prove the
        // wire-encoding step (where it used to crash) succeeds, with updatedAt unset.
        wwUpdateMock.mockImplementationOnce(async (req) => ({ config: { configured: true, template: req.template } }));
        const { updateWakeWordConfig } = await import("./voice");
        const blobs = [0, 1, 2].map(() => fakeBlob(new Uint8Array([1, 2, 3, 4, 5]), "audio/webm"));
        const result = await updateWakeWordConfig({
            label: "Hey Vrooli",
            threshold: 0.65,
            samples: blobs.map((audio) => ({ audio, sampleRateHz: 16000 })),
        });
        expect(wwUpdateMock).toHaveBeenCalledTimes(1);
        const tmpl = requireDefined(requireDefined(wwUpdateMock.mock.calls[0], "updateWakeWordTemplate was not called")[0].template, "request template was undefined");
        expect(tmpl.label).toBe("Hey Vrooli");
        expect(tmpl.threshold).toBeCloseTo(0.65);
        expect(tmpl.samples).toHaveLength(3);
        expect(tmpl.samples[0]?.audio.length).toBe(5);
        expect(tmpl.samples[0]?.format).toBe(AudioFormat.WEBM);
        expect(tmpl.samples[0]?.sampleRateHz).toBe(16000);
        // The crash site: encoding the message. updatedAt must be unset.
        expect(tmpl.updatedAt).toBeUndefined();
        expect(() => toBinary(WakeWordTemplateSchema, tmpl)).not.toThrow();
        // And the decoded round-trip exposes raw samples, not feature objects.
        expect(result.configured).toBe(true);
        expect(result.template?.samples[0]?.mime).toBe("audio/webm");
    });
});
describe("getWakeWordConfig", () => {
    beforeEach(() => {
        wwUpdateMock.mockReset();
        wwGetMock.mockReset();
    });
    afterEach(() => {
        vi.resetModules();
    });
    it("decodes persisted samples as RAW audio (bytes + mime), not feature objects", async () => {
        // Regression for the broken load path: it used to cast raw-audio samples to
        // feature-less AudioFeatures (no `data`/`kind`), so detection could never
        // match. Decoded samples must carry the audio bytes + a playable mime.
        wwGetMock.mockResolvedValueOnce({
            config: {
                configured: true,
                template: {
                    label: "Hey Vrooli",
                    threshold: 0.7,
                    samples: [
                        { audio: new Uint8Array([9, 8, 7]), format: AudioFormat.WEBM, sampleRateHz: 16000 },
                        { audio: new Uint8Array([1, 2]), format: AudioFormat.WAV, sampleRateHz: 16000 },
                    ],
                    updatedAt: undefined,
                },
            },
        });
        const { getWakeWordConfig } = await import("./voice");
        const cfg = await getWakeWordConfig();
        expect(cfg.configured).toBe(true);
        expect(cfg.template?.label).toBe("Hey Vrooli");
        expect(cfg.template?.samples).toHaveLength(2);
        const first = requireDefined(cfg.template?.samples[0], "missing first sample");
        expect(Array.from(first.audio)).toEqual([9, 8, 7]);
        expect(first.mime).toBe("audio/webm");
        expect(first.sampleRateHz).toBe(16000);
        expect(cfg.template?.samples[1]?.mime).toBe("audio/wav");
        // Not cast to engine features.
        expect(first).not.toHaveProperty("data");
        expect(first).not.toHaveProperty("kind");
    });
});
