// Claude-hook / Codex-tailer TTS routing state — a web-console-internal
// concern that does NOT belong in audio-integration (audio-tools knows
// nothing about Claude project settings or the Codex rollout tailer).
//
// All audio synthesis, voice listing, and summarize knobs go through
// audio-integration; this module exposes only the web-console-specific
// glue: routing status, ack ingestion, playback-event ingestion, and the
// auto/backend/startMuted preference triple.
import { API_BASE } from "./client";
function url(path) {
    return `${API_BASE}${path}`;
}
async function jsonOrThrow(resp) {
    if (!resp.ok) {
        const text = await resp.text().catch(() => "");
        throw new Error(`tts-hook ${resp.status}: ${text || resp.statusText}`);
    }
    return (await resp.json());
}
export async function getTTSHookStatus() {
    const resp = await fetch(url("/api/v1/tts-hook/status"), {
        method: "GET",
        headers: { Accept: "application/json" },
    });
    return jsonOrThrow(resp);
}
export async function updateTTSHookConfig(patch) {
    const resp = await fetch(url("/api/v1/tts-hook/config"), {
        method: "PUT",
        headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify(patch),
    });
    return jsonOrThrow(resp);
}
export async function recordTTSHookAck(ack) {
    const resp = await fetch(url("/api/v1/tts-hook/ack"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(ack),
    });
    if (!resp.ok) {
        const text = await resp.text().catch(() => "");
        throw new Error(`tts-hook/ack ${resp.status}: ${text || resp.statusText}`);
    }
}
export async function recordTTSPlaybackEvent(event) {
    const resp = await fetch(url("/api/v1/tts-hook/playback"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(event),
    });
    if (!resp.ok) {
        const text = await resp.text().catch(() => "");
        throw new Error(`tts-hook/playback ${resp.status}: ${text || resp.statusText}`);
    }
}
