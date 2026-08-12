// TTS API client for audio-integration.
//
// Calls web-console's own AudioAdminService + AudioRuntimeService via
// the same-origin Connect transport. The UI never talks to audio-tools
// directly; web-console's API owns the inter-scenario hop.
// HOST DIFFERENCE: web-console's generated TTS proto exposes chunk_index for
// paragraph cache keys; it stays in this adapter because it is a host RPC
// field, not browser capture behavior.
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { audioAdminClient, audioRuntimeClient } from "./voice";
import { responseFormatFromString, responseFormatLabel, summarizeLevelFromString, } from "@vrooli/audio-capture-browser";
import { SummarizeLevel } from "@vrooli/proto-types/web-console/v1/audio_common/audio_common_pb";
// Keep in sync with scenarios/swarm-manager/ui/src/audio-integration/api/tts.ts.
// Unified at 150s to match the swarm-manager UI (was 30s — too short for
// long paragraphs going through summarization). Verified by
// `tts.timeoutConsistency.test.ts` in both scenarios.
export const TTS_SYNTHESIS_TIMEOUT_MS = 150000;
function summarizeLevelLabel(l) {
    switch (l) {
        case SummarizeLevel.LIGHT:
            return "light";
        case SummarizeLevel.MODERATE:
            return "moderate";
        case SummarizeLevel.HEAVY:
            return "heavy";
        default:
            return "moderate";
    }
}
function decodeTTSConfig(c) {
    return {
        autoEnabled: c?.autoEnabled ?? false,
        defaultVoice: c?.defaultVoice ?? "",
        defaultSpeed: c?.defaultSpeed ?? 1,
        defaultResponseFormat: responseFormatLabel(c?.defaultResponseFormat) || "mp3",
    };
}
function decodeSummarizeConfig(c) {
    return {
        enabled: c?.enabled ?? false,
        charThreshold: c?.charThreshold ?? 0,
        level: summarizeLevelLabel(c?.level),
        model: c?.model ?? "",
        timeoutSeconds: c?.timeoutSeconds ?? 0,
    };
}
function decodeSummarizeModel(m) {
    return {
        id: m.id,
        displayName: m.displayName || m.id,
        installed: m.installed,
        recommended: m.recommended,
        defaultEligible: m.defaultEligible,
        reasoning: m.reasoning,
        statusLabel: m.statusLabel,
        pullCommand: m.pullCommand,
        sizeBytes: m.sizeBytes,
        parameterSize: m.parameterSize,
        sourceUrl: m.sourceUrl,
        notes: m.notes,
    };
}
function newTtsRequestId() {
    if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
        return crypto.randomUUID();
    }
    return "rid-" + Math.random().toString(36).slice(2) + Date.now().toString(36);
}
function emitTtsTiming(payload) {
    // Fire-and-forget: telemetry must never throw into the synth path.
    void reportTTSEvent({
        source: "audio-integration",
        stage: "timing",
        message: JSON.stringify(payload),
    }).catch(() => undefined);
}
export async function synthesizeTTSWithMetrics(input, voice, speed, signal, cache) {
    const requestId = newTtsRequestId();
    const synthStartMs = performance.now();
    const totalChars = input.length;
    const timeout = AbortSignal.timeout(TTS_SYNTHESIS_TIMEOUT_MS);
    const combined = signal ? AbortSignal.any([signal, timeout]) : timeout;
    try {
        const resp = await audioRuntimeClient.synthesize({
            text: input,
            voice: voice ?? "",
            responseFormat: responseFormatFromString("mp3"),
            speed: speed ?? 0,
            eventId: cache?.eventId ?? "",
            version: cache?.version ?? "",
            chunkIndex: cache?.chunkIndex ?? 0,
            voiceOverrides: [],
        }, { signal: combined, headers: { "x-tts-request-id": requestId } });
        const deltaMs = performance.now() - synthStartMs;
        emitTtsTiming({
            requestId,
            totalChars,
            chunkCount: 1,
            synthStartMs: Math.round(synthStartMs),
            firstByteMs: Math.round(deltaMs),
            lastByteMs: Math.round(deltaMs),
        });
        const blob = new Blob([resp.audio], {
            type: resp.contentType || "audio/mpeg",
        });
        return { blob, metrics: { requestId, synthStartMs, totalChars } };
    }
    catch (err) {
        emitTtsTiming({
            requestId,
            totalChars,
            chunkCount: 0,
            synthStartMs: Math.round(synthStartMs),
            errorMs: Math.round(performance.now() - synthStartMs),
            error: err instanceof Error ? err.name : "unknown",
        });
        throw err;
    }
}
export function reportTTSPlayStart(metrics) {
    emitTtsTiming({
        requestId: metrics.requestId,
        playStartMs: Math.round(performance.now() - metrics.synthStartMs),
    });
}
export async function synthesizeTTS(input, voice, speed, signal, cache) {
    return (await synthesizeTTSWithMetrics(input, voice, speed, signal, cache)).blob;
}
export async function fetchCachedTTS(eventId, voice, speed, version = "active", signal, chunkIndex = 0) {
    try {
        const resp = await audioRuntimeClient.getTTSCache({ eventId, voice, speed, version, chunkIndex }, { signal });
        if (!resp.hit || resp.audio.byteLength === 0)
            return null;
        return new Blob([resp.audio], { type: resp.contentType || "audio/mpeg" });
    }
    catch {
        return null;
    }
}
export async function getTTSVoices() {
    const resp = await audioRuntimeClient.listVoices({});
    return resp.voices.map((v) => ({ id: v.id, name: v.name }));
}
export async function getTTSConfig() {
    const resp = await audioAdminClient.getTTSConfig({});
    return decodeTTSConfig(resp.config);
}
export async function updateTTSConfig(patch) {
    const paths = [];
    const cfg = {};
    if (patch.autoEnabled !== undefined) {
        cfg.autoEnabled = patch.autoEnabled;
        paths.push("auto_enabled");
    }
    if (patch.defaultVoice !== undefined) {
        cfg.defaultVoice = patch.defaultVoice;
        paths.push("default_voice");
    }
    if (patch.defaultSpeed !== undefined) {
        cfg.defaultSpeed = patch.defaultSpeed;
        paths.push("default_speed");
    }
    if (patch.defaultResponseFormat !== undefined) {
        cfg.defaultResponseFormat = responseFormatFromString(patch.defaultResponseFormat);
        paths.push("default_response_format");
    }
    const resp = await audioAdminClient.updateTTSConfig({
        updateMask: create(FieldMaskSchema, { paths }),
        config: cfg,
    });
    return decodeTTSConfig(resp.config);
}
export async function getTTSSummarizeConfig() {
    const resp = await audioAdminClient.getSummarizeConfig({});
    return decodeSummarizeConfig(resp.config);
}
export async function listTTSSummarizeModels() {
    const resp = await audioAdminClient.listSummarizeModels({});
    return resp.models.map(decodeSummarizeModel);
}
export async function updateTTSSummarizeConfig(patch) {
    const paths = [];
    const cfg = {};
    if (patch.enabled !== undefined) {
        cfg.enabled = patch.enabled;
        paths.push("enabled");
    }
    if (patch.charThreshold !== undefined) {
        cfg.charThreshold = patch.charThreshold;
        paths.push("char_threshold");
    }
    if (patch.level !== undefined) {
        cfg.level = summarizeLevelFromString(patch.level);
        paths.push("level");
    }
    if (patch.model !== undefined) {
        cfg.model = patch.model;
        paths.push("model");
    }
    if (patch.timeoutSeconds !== undefined) {
        cfg.timeoutSeconds = patch.timeoutSeconds;
        paths.push("timeout_seconds");
    }
    const resp = await audioAdminClient.updateSummarizeConfig({
        updateMask: create(FieldMaskSchema, { paths }),
        config: cfg,
    });
    return decodeSummarizeConfig(resp.config);
}
export async function reportTTSEvent(event) {
    await audioRuntimeClient.recordPlaybackEvent({
        event: {
            source: event.source,
            stage: event.stage,
            backend: event.backend ?? "",
            sessionId: event.sessionId ?? "",
            message: event.message ?? "",
            eventId: "",
        },
    });
}
