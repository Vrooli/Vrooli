// Voice (STT) API client for audio-integration.
//
// Calls web-console's own AudioAdminService + AudioRuntimeService via
// the same-origin Connect transport. The UI never talks to audio-tools
// directly — web-console's API owns the inter-scenario hop.
// HOST DIFFERENCE: this adapter uses web-console's Connect transport and
// generated scenario proto package; the browser substrate is shared below it.
import { create } from "@bufbuild/protobuf";
import { createClient as createConnectClient } from "@connectrpc/connect";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { transport, API_BASE } from "../../api/client";
import { audioFormatFromString, audioFormatToMime, rejectBehaviorFromString, rejectBehaviorLabel, speakerCapabilityLabel, speakerModeFromString, speakerModeLabel, streamingModeLabel, strategyPreferenceLabel, streamingModeFromString, strategyPreferenceFromString, timestampToISO, } from "@vrooli/audio-capture-browser";
import { AudioAdminService, WakeWordSampleSchema, WakeWordTemplateSchema, } from "@vrooli/proto-types/web-console/v1/audio_admin/audio_admin_pb";
import { AudioRuntimeService } from "@vrooli/proto-types/web-console/v1/audio_runtime/audio_runtime_pb";
// Web-console Connect clients, mounted same-origin via the shared transport.
export const audioAdminClient = createConnectClient(AudioAdminService, transport);
export const audioRuntimeClient = createConnectClient(AudioRuntimeService, transport);
async function blobToBytes(b) {
    return new Uint8Array(await b.arrayBuffer());
}
function blobFormat(b) {
    const mime = (b.type || "").toLowerCase();
    if (mime.includes("webm"))
        return "webm";
    if (mime.includes("wav"))
        return "wav";
    if (mime.includes("mp3") || mime.includes("mpeg"))
        return "mp3";
    if (mime.includes("ogg"))
        return "ogg";
    if (mime.includes("flac"))
        return "flac";
    return mime.split(";")[0]?.split("/")[1] ?? "webm";
}
function decodeStreamConfig(c) {
    return {
        flushIntervalMs: c?.flushIntervalMs ?? 0,
        minDeltaBytes: c?.minDeltaBytes ?? 0,
        overlapBytes: c?.overlapBytes ?? 0,
        persistentMode: c?.persistentMode ?? false,
        wakeWordEnabled: c?.wakeWordEnabled ?? false,
        wakeWordThreshold: c?.wakeWordThreshold ?? 0,
        segmentSilenceMs: c?.segmentSilenceMs ?? 0,
        streamingMode: streamingModeLabel(c?.streamingMode),
        strategyPreference: strategyPreferenceLabel(c?.strategyPreference),
        vadSilenceMs: c?.vadSilenceMs ?? 0,
        overlapWindowMs: c?.overlapWindowMs ?? 0,
        overlapCommitRuns: c?.overlapCommitRuns ?? 0,
    };
}
function decodeWakeWord(cfg) {
    const configured = cfg?.configured ?? false;
    const tmpl = cfg?.template;
    let template = null;
    if (configured && tmpl) {
        template = {
            label: tmpl.label,
            threshold: tmpl.threshold,
            samples: tmpl.samples.map((s) => ({
                audio: s.audio,
                mime: audioFormatToMime(s.format),
                sampleRateHz: s.sampleRateHz,
            })),
            updatedAt: timestampToISO(tmpl.updatedAt),
        };
    }
    return { configured, template };
}
function decodeSpeakerConfig(c) {
    return {
        enabled: c?.enabled ?? false,
        profileIds: c?.profileIds ?? [],
        threshold: c?.threshold ?? 0,
        mode: speakerModeLabel(c?.mode),
        rejectBehavior: rejectBehaviorLabel(c?.rejectBehavior),
        fallbackWithoutVerification: c?.fallbackWithoutVerification ?? false,
    };
}
function decodeSpeakerProfile(p) {
    return {
        id: p.id,
        display_name: p.displayName,
        created_at: timestampToISO(p.createdAt),
        updated_at: timestampToISO(p.updatedAt),
        model_name: p.modelName,
        embedding_dim: p.embeddingDim,
        sample_rate: p.sampleRate,
        enrollment_audio_seconds: p.enrollmentAudioSeconds,
        notes: p.notes,
    };
}
function apiBaseToWsBase(apiBase) {
    if (apiBase.startsWith("https://"))
        return `wss://${apiBase.slice("https://".length)}`;
    if (apiBase.startsWith("http://"))
        return `ws://${apiBase.slice("http://".length)}`;
    return apiBase;
}
/**
 * Build the WebSocket URL for voice streaming. Same-origin — points at
 * web-console's API, which proxies the upstream WebSocket to audio-tools
 * server-side (Phase E of the UI↔own-API migration).
 */
export function buildVoiceStreamWsUrl(language, sessionId, resumeToken) {
    const wsBase = apiBaseToWsBase(API_BASE.replace(/\/$/, ""));
    // Legacy MediaRecorder callers stay on WebM. Passing a durable session
    // identity opts the canonical PCM provider into the replay-safe v2 wire
    // contract, without changing the legacy transport during migration.
    const params = new URLSearchParams({ format: sessionId ? "pcm_s16le" : "webm" });
    if (language)
        params.set("language", language);
    if (sessionId)
        params.set("protocol_version", "2");
    if (sessionId)
        params.set("session_id", sessionId);
    if (resumeToken)
        params.set("resume_token", resumeToken);
    // Browser qualification cannot attach custom headers to a WebSocket
    // handshake. Mirror Audio Tools' explicitly opt-in page parameters so the
    // Web Console's same-origin proxy can forward a bounded test fault to the
    // upstream server. Normal pages do not carry these parameters, and the
    // upstream still requires its boot-only fault gate.
    if (typeof window !== "undefined") {
        const pageParams = new URLSearchParams(window.location.search);
        const fault = pageParams.get("stt_test_fault");
        if (pageParams.get("stt_test_mode") === "1" && fault) {
            params.set("test_mode", "1");
            params.set("test_fault", fault);
        }
    }
    return `${wsBase}/api/v1/voice/stream?${params.toString()}`;
}
export async function transcribeAudio(audioBlob, language) {
    const resp = await audioRuntimeClient.transcribe({
        audio: await blobToBytes(audioBlob),
        format: audioFormatFromString(blobFormat(audioBlob)),
        language: language ?? "",
        skipSpeakerVerification: false,
        initialPrompt: "",
    });
    return resp.text;
}
export async function transcribeAudioBypassFilter(audioBlob, language) {
    const resp = await audioRuntimeClient.transcribe({
        audio: await blobToBytes(audioBlob),
        format: audioFormatFromString(blobFormat(audioBlob)),
        language: language ?? "",
        skipSpeakerVerification: true,
        initialPrompt: "",
    });
    return resp.text;
}
export async function transcribeAudioWithRetry(audioBlob, maxAttempts = 2, language) {
    let lastError;
    for (let attempt = 0; attempt < maxAttempts; attempt++) {
        try {
            return await transcribeAudio(audioBlob, language);
        }
        catch (err) {
            lastError = err;
            if (attempt < maxAttempts - 1) {
                await new Promise((r) => setTimeout(r, 500 * (attempt + 1)));
            }
        }
    }
    throw lastError;
}
export async function getVoiceStreamConfig() {
    const resp = await audioAdminClient.getStreamConfig({});
    return decodeStreamConfig(resp.config);
}
export async function updateVoiceStreamConfig(patch) {
    const paths = [];
    const cfg = {};
    if (patch.flushIntervalMs !== undefined) {
        cfg.flushIntervalMs = patch.flushIntervalMs;
        paths.push("flush_interval_ms");
    }
    if (patch.minDeltaBytes !== undefined) {
        cfg.minDeltaBytes = patch.minDeltaBytes;
        paths.push("min_delta_bytes");
    }
    if (patch.overlapBytes !== undefined) {
        cfg.overlapBytes = patch.overlapBytes;
        paths.push("overlap_bytes");
    }
    if (patch.persistentMode !== undefined) {
        cfg.persistentMode = patch.persistentMode;
        paths.push("persistent_mode");
    }
    if (patch.wakeWordEnabled !== undefined) {
        cfg.wakeWordEnabled = patch.wakeWordEnabled;
        paths.push("wake_word_enabled");
    }
    if (patch.wakeWordThreshold !== undefined) {
        cfg.wakeWordThreshold = patch.wakeWordThreshold;
        paths.push("wake_word_threshold");
    }
    if (patch.segmentSilenceMs !== undefined) {
        cfg.segmentSilenceMs = patch.segmentSilenceMs;
        paths.push("segment_silence_ms");
    }
    if (patch.streamingMode !== undefined) {
        cfg.streamingMode = streamingModeFromString(patch.streamingMode);
        paths.push("streaming_mode");
    }
    if (patch.strategyPreference !== undefined) {
        cfg.strategyPreference = strategyPreferenceFromString(patch.strategyPreference);
        paths.push("strategy_preference");
    }
    if (patch.vadSilenceMs !== undefined) {
        cfg.vadSilenceMs = patch.vadSilenceMs;
        paths.push("vad_silence_ms");
    }
    if (patch.overlapWindowMs !== undefined) {
        cfg.overlapWindowMs = patch.overlapWindowMs;
        paths.push("overlap_window_ms");
    }
    if (patch.overlapCommitRuns !== undefined) {
        cfg.overlapCommitRuns = patch.overlapCommitRuns;
        paths.push("overlap_commit_runs");
    }
    const resp = await audioAdminClient.updateStreamConfig({
        updateMask: create(FieldMaskSchema, { paths }),
        config: cfg,
    });
    return decodeStreamConfig(resp.config);
}
export async function getWakeWordConfig() {
    const resp = await audioAdminClient.getWakeWordConfig({});
    return decodeWakeWord(resp.config);
}
export async function updateWakeWordConfig(input) {
    const samples = await Promise.all(input.samples.map(async (s) => create(WakeWordSampleSchema, {
        audio: await blobToBytes(s.audio),
        format: audioFormatFromString(blobFormat(s.audio)),
        sampleRateHz: s.sampleRateHz,
    })));
    // updated_at is intentionally omitted — it is server/display metadata, never
    // client-authored. Assigning an ISO string here was the original crash:
    // protobuf-es encodes the Timestamp field via new Date(NaN).toISOString().
    const template = create(WakeWordTemplateSchema, {
        label: input.label,
        threshold: input.threshold,
        samples,
    });
    const resp = await audioAdminClient.updateWakeWordTemplate({ template });
    return decodeWakeWord(resp.config);
}
export async function deleteWakeWordConfig() {
    const resp = await audioAdminClient.deleteWakeWordTemplate({});
    return decodeWakeWord(resp.config);
}
export async function getSpeakerVerificationConfig() {
    const resp = await audioAdminClient.getSpeakerConfig({});
    return decodeSpeakerConfig(resp.config);
}
export async function updateSpeakerVerificationConfig(patch) {
    const paths = [];
    const cfg = {};
    if (patch.enabled !== undefined) {
        cfg.enabled = patch.enabled;
        paths.push("enabled");
    }
    if (patch.profileIds !== undefined) {
        cfg.profileIds = patch.profileIds;
        paths.push("profile_ids");
    }
    if (patch.threshold !== undefined) {
        cfg.threshold = patch.threshold;
        paths.push("threshold");
    }
    if (patch.mode !== undefined) {
        cfg.mode = speakerModeFromString(patch.mode);
        paths.push("mode");
    }
    if (patch.rejectBehavior !== undefined) {
        cfg.rejectBehavior = rejectBehaviorFromString(patch.rejectBehavior);
        paths.push("reject_behavior");
    }
    if (patch.fallbackWithoutVerification !== undefined) {
        cfg.fallbackWithoutVerification = patch.fallbackWithoutVerification;
        paths.push("fallback_without_verification");
    }
    const resp = await audioAdminClient.updateSpeakerConfig({
        updateMask: create(FieldMaskSchema, { paths }),
        config: cfg,
    });
    return decodeSpeakerConfig(resp.config);
}
export async function getSpeakerVerificationStatus() {
    const resp = await audioAdminClient.getSpeakerStatus({});
    const st = resp.status;
    if (!st)
        throw new Error("speaker status response missing status field");
    return {
        config: decodeSpeakerConfig(st.config),
        capability: speakerCapabilityLabel(st.capability),
        capabilityLabel: st.capabilityLabel || undefined,
        resourceReady: st.resourceReady,
        profileConfigured: st.profileConfigured,
        profileExists: st.profileExists,
        profileCount: st.profileCount,
        profiles: st.profiles.map(decodeSpeakerProfile),
        info: st.info
            ? {
                backend: st.info.backend,
                model: st.info.model,
                device: st.info.device,
                sample_rate: st.info.sampleRate,
                version: st.info.version,
                embedding_dim: st.info.embeddingDim,
            }
            : undefined,
        checkedAt: timestampToISO(st.checkedAt),
    };
}
export async function listSpeakerVerificationProfiles() {
    const resp = await audioAdminClient.listSpeakerProfiles({});
    return resp.profiles.map(decodeSpeakerProfile);
}
export async function enrollSpeakerVerificationProfile(args) {
    const req = {
        audio: await blobToBytes(args.audioBlob),
        format: audioFormatFromString(blobFormat(args.audioBlob)),
        profileId: args.profileId ?? "",
        displayName: args.displayName ?? "",
        notes: args.notes ?? "",
    };
    if (args.addToActive !== undefined)
        req.addToActive = args.addToActive;
    if (args.enable !== undefined)
        req.enable = args.enable;
    const resp = await audioAdminClient.enrollSpeakerProfile(req);
    const en = resp.enrollment;
    return {
        enrollment: {
            profile_id: en?.profileId ?? "",
            display_name: en?.displayName ?? "",
            embedding_dim: en?.embeddingDim ?? 0,
            sample_rate: en?.sampleRate ?? 0,
            enrollment_audio_seconds: en?.enrollmentAudioSeconds ?? 0,
            model_name: en?.modelName ?? "",
            created_at: timestampToISO(en?.createdAt),
        },
        config: decodeSpeakerConfig(resp.config),
    };
}
export async function clearSpeakerVerificationProfile() {
    const resp = await audioAdminClient.clearSpeakerProfileBinding({});
    return decodeSpeakerConfig(resp.config);
}
export async function removeSpeakerVerificationProfile(profileId) {
    const resp = await audioAdminClient.unbindSpeakerProfile({ profileId });
    return decodeSpeakerConfig(resp.config);
}
export async function deleteSpeakerVerificationProfile(profileId) {
    const resp = await audioAdminClient.deleteSpeakerProfile({ profileId });
    return decodeSpeakerConfig(resp.config);
}
