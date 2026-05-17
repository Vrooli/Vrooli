// TTS API client for @audio-tools/embed.
//
// Exposes the audio operations audio-tools' TTSService + SummarizeService
// support. Consumer-scenario-specific concerns (conversation routing,
// hook ack, status snapshots tied to consumer internals) stay in the
// consumer; this module deliberately ships only the audio surface.

import { useAudioToolsClient, type AudioToolsClient } from "../client";

export interface TTSVoiceInfo {
  id: string;
  name: string;
}

// TTSConfig matches the audio-tools TTS Config message: a single canonical
// voice id (resolved by the active adapter) and the auto-enable flag.
// Consumer scenarios that historically tracked a "backend" ("auto"|"kokoro"|
// "browser") keep that selection client-side — audio-tools' provider chain
// owns backend routing now (BYOK -> Vrooli/LPBS -> Local).
export interface TTSConfig {
  autoEnabled: boolean;
  defaultVoice: string;
  defaultSpeed: number;
  defaultResponseFormat: string;
}

export interface TTSPlaybackEvent {
  source: string;
  stage: string;
  backend?: string;
  sessionId?: string;
  message?: string;
}

export interface TTSSummarizeConfig {
  enabled: boolean;
  charThreshold: number;
  level: "light" | "moderate" | "heavy";
  model: string;
  timeoutSeconds: number;
}

const TTS_SYNTHESIS_TIMEOUT_MS = 30_000;

function decodeTTSConfig(c:
  | {
      autoEnabled: boolean;
      defaultVoice: string;
      defaultSpeed: number;
      defaultResponseFormat: string;
    }
  | undefined,
): TTSConfig {
  return {
    autoEnabled: c?.autoEnabled ?? false,
    defaultVoice: c?.defaultVoice ?? "",
    defaultSpeed: c?.defaultSpeed ?? 1,
    defaultResponseFormat: c?.defaultResponseFormat ?? "mp3",
  };
}

function decodeSummarizeConfig(c:
  | { enabled: boolean; charThreshold: number; level: string; model: string; timeoutSeconds: number }
  | undefined,
): TTSSummarizeConfig {
  return {
    enabled: c?.enabled ?? false,
    charThreshold: c?.charThreshold ?? 0,
    level: ((c?.level || "moderate") as TTSSummarizeConfig["level"]),
    model: c?.model ?? "",
    timeoutSeconds: c?.timeoutSeconds ?? 0,
  };
}

/**
 * createTtsApi returns audio-tools TTS operations bound to a specific client.
 * Consumers can call these from inside React or pass the returned bag to
 * non-React utilities — the functions are stable for the client's lifetime.
 */
export function createTtsApi(client: AudioToolsClient) {
  return {
    async synthesizeTTS(
      input: string,
      voice?: string,
      speed?: number,
      signal?: AbortSignal,
    ): Promise<Blob> {
      const timeout = AbortSignal.timeout(TTS_SYNTHESIS_TIMEOUT_MS);
      const combined = signal ? AbortSignal.any([signal, timeout]) : timeout;
      const resp = await client.tts.synthesize(
        { text: input, voice: voice ?? "", responseFormat: "mp3", speed: speed ?? 0, eventId: "", voiceOverrides: {} },
        { signal: combined },
      );
      return new Blob([resp.audio as Uint8Array<ArrayBuffer>], { type: resp.contentType || "audio/mpeg" });
    },

    async fetchCachedTTS(
      eventId: string,
      voice: string,
      speed: number,
      version: "active" | "original" = "active",
      signal?: AbortSignal,
    ): Promise<Blob | null> {
      try {
        const resp = await client.tts.getCache({ eventId, voice, speed, version }, { signal });
        if (resp.audio.byteLength === 0) return null;
        return new Blob([resp.audio as Uint8Array<ArrayBuffer>], { type: resp.contentType || "audio/mpeg" });
      } catch {
        return null;
      }
    },

    async getTTSVoices(): Promise<TTSVoiceInfo[]> {
      const resp = await client.tts.listVoices({});
      return resp.voices.map((v) => ({ id: v.id, name: v.name }));
    },

    async getTTSConfig(): Promise<TTSConfig> {
      const resp = await client.tts.getConfig({});
      return decodeTTSConfig(resp.config);
    },

    async updateTTSConfig(patch: Partial<TTSConfig>): Promise<TTSConfig> {
      const req: Record<string, unknown> = {};
      if (patch.autoEnabled !== undefined) { req.autoEnabled = patch.autoEnabled; req.hasAutoEnabled = true; }
      if (patch.defaultVoice !== undefined) { req.defaultVoice = patch.defaultVoice; req.hasDefaultVoice = true; }
      if (patch.defaultSpeed !== undefined) { req.defaultSpeed = patch.defaultSpeed; req.hasDefaultSpeed = true; }
      if (patch.defaultResponseFormat !== undefined) { req.defaultResponseFormat = patch.defaultResponseFormat; req.hasDefaultResponseFormat = true; }
      const resp = await client.tts.updateConfig(req as Parameters<typeof client.tts.updateConfig>[0]);
      return decodeTTSConfig(resp.config);
    },

    async getTTSSummarizeConfig(): Promise<TTSSummarizeConfig> {
      const resp = await client.summarize.getSummarizeConfig({});
      return decodeSummarizeConfig(resp.config);
    },

    async updateTTSSummarizeConfig(patch: Partial<TTSSummarizeConfig>): Promise<TTSSummarizeConfig> {
      const resp = await client.summarize.updateSummarizeConfig({
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
      });
      return decodeSummarizeConfig(resp.config);
    },

    async reportTTSEvent(event: TTSPlaybackEvent): Promise<void> {
      await client.tts.recordPlaybackEvent({
        event: {
          source: event.source,
          stage: event.stage,
          backend: event.backend ?? "",
          sessionId: event.sessionId ?? "",
          message: event.message ?? "",
        },
      });
    },
  };
}

/** Convenience hook: pulls the active client from context and binds TTS ops. */
export function useTtsApi() {
  return createTtsApi(useAudioToolsClient());
}

// Lazy module-level singleton bound to the default client (window.__AUDIO_TOOLS_URL__).
// Hooks in this package use free-function imports like `synthesizeTTS(...)`;
// the singleton resolves on first call so consumers that wire their own
// <AudioToolsProvider> can do so before any audio call.
import { createAudioToolsClient } from "../client";
let _lazyApi: ReturnType<typeof createTtsApi> | null = null;
function lazy() {
  if (_lazyApi === null) _lazyApi = createTtsApi(createAudioToolsClient());
  return _lazyApi;
}

export function reportTTSEvent(event: TTSPlaybackEvent): Promise<void> { return lazy().reportTTSEvent(event); }
export function synthesizeTTS(input: string, voice?: string, speed?: number, signal?: AbortSignal): Promise<Blob> {
  return lazy().synthesizeTTS(input, voice, speed, signal);
}
export function fetchCachedTTS(eventId: string, voice: string, speed: number, version: "active" | "original" = "active", signal?: AbortSignal): Promise<Blob | null> {
  return lazy().fetchCachedTTS(eventId, voice, speed, version, signal);
}
export function getTTSVoices(): Promise<TTSVoiceInfo[]> { return lazy().getTTSVoices(); }
export function getTTSConfig(): Promise<TTSConfig> { return lazy().getTTSConfig(); }
export function updateTTSConfig(patch: Partial<TTSConfig>): Promise<TTSConfig> { return lazy().updateTTSConfig(patch); }
export function getTTSSummarizeConfig(): Promise<TTSSummarizeConfig> { return lazy().getTTSSummarizeConfig(); }
export function updateTTSSummarizeConfig(patch: Partial<TTSSummarizeConfig>): Promise<TTSSummarizeConfig> { return lazy().updateTTSSummarizeConfig(patch); }
