// TTS API client for audio-integration.
//
// Exposes the audio operations audio-tools' TTSService + SummarizeService
// support. Consumer-scenario-specific concerns (conversation routing,
// hook ack, status snapshots tied to consumer internals) stay in the
// consumer; this module deliberately ships only the audio surface.

import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";

import { getActiveAudioToolsClient, useAudioToolsClient, type AudioToolsClient } from "../client";
import {
  responseFormatFromString,
  responseFormatLabel,
  summarizeLevelFromString,
} from "@vrooli/audio-capture-browser";
import type { Config as TTSConfigMsg } from "@vrooli/proto-types/audio-tools/v1/tts/tts_pb";
import type { SummarizeConfig as SummarizeConfigMsg } from "@vrooli/proto-types/audio-tools/v1/summarize/summarize_pb";
import { SummarizeLevel } from "@vrooli/proto-types/audio-tools/v1/summarize/summarize_pb";

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

function summarizeLevelLabel(l: SummarizeLevel | undefined): TTSSummarizeConfig["level"] {
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

function decodeTTSConfig(c: TTSConfigMsg | undefined): TTSConfig {
  return {
    autoEnabled: c?.autoEnabled ?? false,
    defaultVoice: c?.defaultVoice ?? "",
    defaultSpeed: c?.defaultSpeed ?? 1,
    defaultResponseFormat: responseFormatLabel(c?.defaultResponseFormat) || "mp3",
  };
}

function decodeSummarizeConfig(c: SummarizeConfigMsg | undefined): TTSSummarizeConfig {
  return {
    enabled: c?.enabled ?? false,
    charThreshold: c?.charThreshold ?? 0,
    level: summarizeLevelLabel(c?.level),
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
        { text: input, voice: voice ?? "", responseFormat: responseFormatFromString("mp3"), speed: speed ?? 0, eventId: "", voiceOverrides: [] },
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
      const paths: string[] = [];
      const cfg: Record<string, unknown> = {};
      if (patch.autoEnabled !== undefined) { cfg.autoEnabled = patch.autoEnabled; paths.push("auto_enabled"); }
      if (patch.defaultVoice !== undefined) { cfg.defaultVoice = patch.defaultVoice; paths.push("default_voice"); }
      if (patch.defaultSpeed !== undefined) { cfg.defaultSpeed = patch.defaultSpeed; paths.push("default_speed"); }
      if (patch.defaultResponseFormat !== undefined) { cfg.defaultResponseFormat = responseFormatFromString(patch.defaultResponseFormat); paths.push("default_response_format"); }
      const resp = await client.tts.updateConfig({
        updateMask: create(FieldMaskSchema, { paths }),
        config: cfg,
      });
      return decodeTTSConfig(resp.config);
    },

    async getTTSSummarizeConfig(): Promise<TTSSummarizeConfig> {
      const resp = await client.summarize.getSummarizeConfig({});
      return decodeSummarizeConfig(resp.config);
    },

    async updateTTSSummarizeConfig(patch: Partial<TTSSummarizeConfig>): Promise<TTSSummarizeConfig> {
      const paths: string[] = [];
      const cfg: Record<string, unknown> = {};
      if (patch.enabled !== undefined) { cfg.enabled = patch.enabled; paths.push("enabled"); }
      if (patch.charThreshold !== undefined) { cfg.charThreshold = patch.charThreshold; paths.push("char_threshold"); }
      if (patch.level !== undefined) { cfg.level = summarizeLevelFromString(patch.level); paths.push("level"); }
      if (patch.model !== undefined) { cfg.model = patch.model; paths.push("model"); }
      if (patch.timeoutSeconds !== undefined) { cfg.timeoutSeconds = patch.timeoutSeconds; paths.push("timeout_seconds"); }
      const resp = await client.summarize.updateSummarizeConfig({
        updateMask: create(FieldMaskSchema, { paths }),
        config: cfg,
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

// Module-level singleton bound to the active AudioToolsClient registered
// by <AudioToolsProvider>. Free-function exports (`synthesizeTTS`, etc.)
// resolve through this so consumer call sites do not need to thread a
// client argument.
let _lazyClient: AudioToolsClient | null = null;
let _lazyApi: ReturnType<typeof createTtsApi> | null = null;
function lazy() {
  const client = getActiveAudioToolsClient();
  if (_lazyApi === null || _lazyClient !== client) {
    _lazyClient = client;
    _lazyApi = createTtsApi(client);
  }
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
