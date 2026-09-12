// Claude-hook / Codex-tailer TTS routing state — a web-console-internal
// concern that does NOT belong in audio-integration (audio-tools knows
// nothing about Claude project settings or the Codex rollout tailer).
//
// All audio synthesis, voice listing, and summarize knobs go through
// audio-integration; this module exposes only the web-console-specific
// glue: routing status, ack ingestion, playback-event ingestion, and the
// auto/backend/startMuted preference triple.

import { API_BASE } from "./client";

export type TTSBackendPreference = "auto" | "kokoro" | "browser";

export interface TTSHookConfig {
  autoEnabled: boolean;
  backend: TTSBackendPreference;
  startMuted: boolean;
}

export interface TTSRoutingResult {
  appended: boolean;
  code: string;
  reason: string;
  source: string;
  sessionId?: string;
  eventId?: string;
  sequence?: number;
  duplicate?: boolean;
}

export interface TTSClientAck {
  eventId: string;
  source: string;
  sessionId: string;
  stage: string;
  backend?: string;
  message?: string;
}

export interface TTSPlaybackEvent {
  source: string;
  stage: string;
  backend?: string;
  sessionId?: string;
  message?: string;
}

export interface TTSHookStatus {
  config: TTSHookConfig;
  hookRegistered: boolean;
  hookCode?: string;
  hookReason: string;
  hookSettingsPath?: string;
  lastHookRouting?: TTSRoutingResult;
  lastHookRoutingAt?: string;
  lastTailerRouting?: TTSRoutingResult;
  lastTailerRoutingAt?: string;
  lastHookAck?: TTSClientAck;
  lastHookAckAt?: string;
  lastTailerAck?: TTSClientAck;
  lastTailerAckAt?: string;
  lastPlaybackEvent?: TTSPlaybackEvent;
  lastPlaybackAt?: string;
  audioToolsCapability?: string;
  audioToolsCapabilityLabel?: string;
}

function url(path: string): string {
  return `${API_BASE}${path}`;
}

async function jsonOrThrow<T>(resp: Response): Promise<T> {
  if (!resp.ok) {
    const text = await resp.text().catch(() => "");
    throw new Error(`tts-hook ${resp.status}: ${text || resp.statusText}`);
  }
  return (await resp.json()) as T;
}

export async function getTTSHookStatus(): Promise<TTSHookStatus> {
  const resp = await fetch(url("/api/v1/tts-hook/status"), {
    method: "GET",
    headers: { Accept: "application/json" },
  });
  return jsonOrThrow<TTSHookStatus>(resp);
}

export async function updateTTSHookConfig(
  patch: Partial<TTSHookConfig>,
): Promise<TTSHookConfig> {
  const resp = await fetch(url("/api/v1/tts-hook/config"), {
    method: "PUT",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: JSON.stringify(patch),
  });
  return jsonOrThrow<TTSHookConfig>(resp);
}

export async function recordTTSHookAck(ack: TTSClientAck): Promise<void> {
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

export async function recordTTSPlaybackEvent(event: TTSPlaybackEvent): Promise<void> {
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
