import { createClient } from "@connectrpc/connect";
import { TTSService } from "@vrooli/proto-types/web-console/v1/tts/tts_pb";

import { transport } from "./client";

export const ttsClient = createClient(TTSService, transport);

// ── Types ────────────────────────────────────────────────────────────────

export interface TTSVoiceInfo {
  id: string;
  name: string;
}

export interface TTSPlaybackEvent {
  source: string;
  stage: string;
  backend?: string;
  sessionId?: string;
  message?: string;
}

export interface TTSConfig {
  autoEnabled: boolean;
  backend: "auto" | "kokoro" | "browser";
  kokoroVoice: string;
  kokoroSpeed: number;
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

export interface TTSSummarizeConfig {
  enabled: boolean;
  charThreshold: number;
  level: "light" | "moderate" | "heavy";
  model: string;
  timeoutSeconds: number;
}

export interface TTSStatus {
  config: TTSConfig;
  hookRegistered: boolean;
  hookCode?: string;
  hookReason: string;
  hookSettingsPath?: string;
  lastRouting?: TTSRoutingResult;
  lastRoutingAt?: string;
  lastHookRouting?: TTSRoutingResult;
  lastHookRoutingAt?: string;
  lastTailerRouting?: TTSRoutingResult;
  lastTailerRoutingAt?: string;
  lastAck?: TTSClientAck;
  lastAckAt?: string;
  lastHookAck?: TTSClientAck;
  lastHookAckAt?: string;
  lastTailerAck?: TTSClientAck;
  lastTailerAckAt?: string;
  lastPlaybackEvent?: TTSPlaybackEvent;
  lastPlaybackAt?: string;
  kokoroCapability?: string;
  kokoroCapabilityLabel?: string;
}

// ── Decoders ─────────────────────────────────────────────────────────────

function decodeTTSConfig(c: {
  autoEnabled: boolean;
  backend: string;
  kokoroVoice: string;
  kokoroSpeed: number;
} | undefined): TTSConfig {
  return {
    autoEnabled: c?.autoEnabled ?? false,
    backend: ((c?.backend || "auto") as TTSConfig["backend"]),
    kokoroVoice: c?.kokoroVoice ?? "",
    kokoroSpeed: c?.kokoroSpeed ?? 1,
  };
}

function decodeSummarizeConfig(c: {
  enabled: boolean;
  charThreshold: number;
  level: string;
  model: string;
  timeoutSeconds: number;
} | undefined): TTSSummarizeConfig {
  return {
    enabled: c?.enabled ?? false,
    charThreshold: c?.charThreshold ?? 0,
    level: ((c?.level || "moderate") as TTSSummarizeConfig["level"]),
    model: c?.model ?? "",
    timeoutSeconds: c?.timeoutSeconds ?? 0,
  };
}

function decodeRouting(r: {
  appended: boolean;
  code: string;
  reason: string;
  source: string;
  sessionId: string;
  eventId: string;
  sequence: bigint;
  duplicate: boolean;
} | undefined): TTSRoutingResult | undefined {
  if (!r || (!r.appended && !r.code && !r.source)) return undefined;
  return {
    appended: r.appended,
    code: r.code,
    reason: r.reason,
    source: r.source,
    sessionId: r.sessionId || undefined,
    eventId: r.eventId || undefined,
    sequence: r.sequence !== 0n ? Number(r.sequence) : undefined,
    duplicate: r.duplicate || undefined,
  };
}

function decodeAck(a: {
  eventId: string;
  source: string;
  sessionId: string;
  stage: string;
  backend: string;
  message: string;
} | undefined): TTSClientAck | undefined {
  if (!a || (!a.eventId && !a.source && !a.stage)) return undefined;
  return {
    eventId: a.eventId,
    source: a.source,
    sessionId: a.sessionId,
    stage: a.stage,
    backend: a.backend || undefined,
    message: a.message || undefined,
  };
}

function decodePlayback(p: {
  source: string;
  stage: string;
  backend: string;
  sessionId: string;
  message: string;
} | undefined): TTSPlaybackEvent | undefined {
  if (!p || (!p.source && !p.stage)) return undefined;
  return {
    source: p.source,
    stage: p.stage,
    backend: p.backend || undefined,
    sessionId: p.sessionId || undefined,
    message: p.message || undefined,
  };
}

// ── Wrappers ─────────────────────────────────────────────────────────────

export async function reportTTSEvent(event: TTSPlaybackEvent): Promise<void> {
  await ttsClient.recordPlaybackEvent({
    event: {
      source: event.source,
      stage: event.stage,
      backend: event.backend ?? "",
      sessionId: event.sessionId ?? "",
      message: event.message ?? "",
    },
  });
}

/** Maximum time (ms) to wait for Kokoro to return synthesized audio. */
const TTS_SYNTHESIS_TIMEOUT_MS = 30_000;

/** Synthesize text to audio via the Kokoro TTS backend. */
export async function synthesizeTTS(
  input: string,
  voice?: string,
  speed?: number,
  signal?: AbortSignal,
): Promise<Blob> {
  const timeout = AbortSignal.timeout(TTS_SYNTHESIS_TIMEOUT_MS);
  const combined = signal ? AbortSignal.any([signal, timeout]) : timeout;
  const resp = await ttsClient.synthesize(
    {
      input,
      voice: voice ?? "",
      responseFormat: "mp3",
      speed: speed ?? 0,
      eventId: "",
      version: "",
    },
    { signal: combined },
  );
  return new Blob([resp.audio as Uint8Array<ArrayBuffer>], {
    type: resp.contentType || "audio/mpeg",
  });
}

/**
 * Fetch pre-cached TTS audio for an event. Returns the audio Blob on cache
 * hit, or null on cache miss. Used by the cache-first playback path to
 * eliminate synthesis latency on tab switch.
 */
export async function fetchCachedTTS(
  eventId: string,
  voice: string,
  speed: number,
  version: "active" | "original" = "active",
  signal?: AbortSignal,
): Promise<Blob | null> {
  try {
    const resp = await ttsClient.getCache(
      { eventId, voice, speed, version },
      { signal },
    );
    if (resp.audio.byteLength === 0) return null;
    return new Blob([resp.audio as Uint8Array<ArrayBuffer>], {
      type: resp.contentType || "audio/mpeg",
    });
  } catch {
    return null;
  }
}

export async function getTTSVoices(): Promise<TTSVoiceInfo[]> {
  const resp = await ttsClient.listVoices({});
  return resp.voices.map((v) => ({ id: v.id, name: v.name }));
}

export async function getTTSConfig(): Promise<TTSConfig> {
  const resp = await ttsClient.getConfig({});
  return decodeTTSConfig(resp.config);
}

export async function updateTTSConfig(
  patch: Partial<TTSConfig>,
): Promise<TTSConfig> {
  const req = {
    autoEnabled: patch.autoEnabled ?? false,
    hasAutoEnabled: patch.autoEnabled !== undefined,
    backend: patch.backend ?? "",
    hasBackend: patch.backend !== undefined,
    kokoroVoice: patch.kokoroVoice ?? "",
    hasKokoroVoice: patch.kokoroVoice !== undefined,
    kokoroSpeed: patch.kokoroSpeed ?? 0,
    hasKokoroSpeed: patch.kokoroSpeed !== undefined,
  };
  const resp = await ttsClient.updateConfig(req);
  return decodeTTSConfig(resp.config);
}

export async function getTTSSummarizeConfig(): Promise<TTSSummarizeConfig> {
  const resp = await ttsClient.getSummarizeConfig({});
  return decodeSummarizeConfig(resp.config);
}

export async function updateTTSSummarizeConfig(
  patch: Partial<TTSSummarizeConfig>,
): Promise<TTSSummarizeConfig> {
  const req = {
    enabled: patch.enabled ?? false,
    hasEnabled: patch.enabled !== undefined,
    charThreshold: patch.charThreshold ?? 0,
    hasCharThreshold: patch.charThreshold !== undefined,
    level: patch.level ?? "",
    hasLevel: patch.level !== undefined,
    model: patch.model ?? "",
    hasModel: patch.model !== undefined,
    timeoutSeconds: patch.timeoutSeconds ?? 0,
    hasTimeoutSeconds: patch.timeoutSeconds !== undefined,
  };
  const resp = await ttsClient.updateSummarizeConfig(req);
  return decodeSummarizeConfig(resp.config);
}

export async function getTTSStatus(): Promise<TTSStatus> {
  const resp = await ttsClient.getStatus({});
  const st = resp.status;
  if (!st) {
    throw new Error("TTS status response missing payload");
  }
  return {
    config: decodeTTSConfig(st.config),
    hookRegistered: st.hookRegistered,
    hookCode: st.hookCode || undefined,
    hookReason: st.hookReason,
    hookSettingsPath: st.hookSettingsPath || undefined,
    lastRouting: decodeRouting(st.lastRouting),
    lastRoutingAt: st.lastRoutingAt || undefined,
    lastHookRouting: decodeRouting(st.lastHookRouting),
    lastHookRoutingAt: st.lastHookRoutingAt || undefined,
    lastTailerRouting: decodeRouting(st.lastTailerRouting),
    lastTailerRoutingAt: st.lastTailerRoutingAt || undefined,
    lastAck: decodeAck(st.lastAck),
    lastAckAt: st.lastAckAt || undefined,
    lastHookAck: decodeAck(st.lastHookAck),
    lastHookAckAt: st.lastHookAckAt || undefined,
    lastTailerAck: decodeAck(st.lastTailerAck),
    lastTailerAckAt: st.lastTailerAckAt || undefined,
    lastPlaybackEvent: decodePlayback(st.lastPlaybackEvent),
    lastPlaybackAt: st.lastPlaybackAt || undefined,
    kokoroCapability: st.kokoroCapability || undefined,
    kokoroCapabilityLabel: st.kokoroCapabilityLabel || undefined,
  };
}
