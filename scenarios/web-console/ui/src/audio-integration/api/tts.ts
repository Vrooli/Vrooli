// TTS API client for audio-integration.
//
// Calls web-console's own AudioAdminService + AudioRuntimeService via
// the same-origin Connect transport. The UI never talks to audio-tools
// directly; web-console's API owns the inter-scenario hop.

import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";

import { audioAdminClient, audioRuntimeClient } from "./voice";
import {
  responseFormatFromString,
  responseFormatLabel,
  summarizeLevelFromString,
} from "./protomap";
import type { TTSConfig as TTSConfigMsg } from "@vrooli/proto-types/web-console/v1/audio_admin/audio_admin_pb";
import type { SummarizeConfig as SummarizeConfigMsg } from "@vrooli/proto-types/web-console/v1/audio_admin/audio_admin_pb";
import type { SummarizeModel as SummarizeModelMsg } from "@vrooli/proto-types/web-console/v1/audio_admin/audio_admin_pb";
import { SummarizeLevel } from "@vrooli/proto-types/web-console/v1/audio_common/audio_common_pb";

export interface TTSVoiceInfo {
  id: string;
  name: string;
}

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

export interface TTSSummarizeModel {
  id: string;
  displayName: string;
  installed: boolean;
  recommended: boolean;
  defaultEligible: boolean;
  reasoning: boolean;
  statusLabel: string;
  pullCommand: string;
  sizeBytes: bigint;
  parameterSize: string;
  sourceUrl: string;
  notes: string;
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

function decodeSummarizeModel(m: SummarizeModelMsg): TTSSummarizeModel {
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

export async function synthesizeTTS(
  input: string,
  voice?: string,
  speed?: number,
  signal?: AbortSignal,
): Promise<Blob> {
  const timeout = AbortSignal.timeout(TTS_SYNTHESIS_TIMEOUT_MS);
  const combined = signal ? AbortSignal.any([signal, timeout]) : timeout;
  const resp = await audioRuntimeClient.synthesize(
    {
      text: input,
      voice: voice ?? "",
      responseFormat: responseFormatFromString("mp3"),
      speed: speed ?? 0,
      eventId: "",
      voiceOverrides: [],
    },
    { signal: combined },
  );
  return new Blob([resp.audio as Uint8Array<ArrayBuffer>], { type: resp.contentType || "audio/mpeg" });
}

export async function fetchCachedTTS(
  eventId: string,
  voice: string,
  speed: number,
  version: "active" | "original" = "active",
  signal?: AbortSignal,
): Promise<Blob | null> {
  try {
    const resp = await audioRuntimeClient.getTTSCache(
      { eventId, voice, speed, version },
      { signal },
    );
    if (!resp.hit || resp.audio.byteLength === 0) return null;
    return new Blob([resp.audio as Uint8Array<ArrayBuffer>], { type: resp.contentType || "audio/mpeg" });
  } catch {
    return null;
  }
}

export async function getTTSVoices(): Promise<TTSVoiceInfo[]> {
  const resp = await audioRuntimeClient.listVoices({});
  return resp.voices.map((v) => ({ id: v.id, name: v.name }));
}

export async function getTTSConfig(): Promise<TTSConfig> {
  const resp = await audioAdminClient.getTTSConfig({});
  return decodeTTSConfig(resp.config);
}

export async function updateTTSConfig(patch: Partial<TTSConfig>): Promise<TTSConfig> {
  const paths: string[] = [];
  const cfg: Record<string, unknown> = {};
  if (patch.autoEnabled !== undefined) { cfg.autoEnabled = patch.autoEnabled; paths.push("auto_enabled"); }
  if (patch.defaultVoice !== undefined) { cfg.defaultVoice = patch.defaultVoice; paths.push("default_voice"); }
  if (patch.defaultSpeed !== undefined) { cfg.defaultSpeed = patch.defaultSpeed; paths.push("default_speed"); }
  if (patch.defaultResponseFormat !== undefined) { cfg.defaultResponseFormat = responseFormatFromString(patch.defaultResponseFormat); paths.push("default_response_format"); }
  const resp = await audioAdminClient.updateTTSConfig({
    updateMask: create(FieldMaskSchema, { paths }),
    config: cfg,
  });
  return decodeTTSConfig(resp.config);
}

export async function getTTSSummarizeConfig(): Promise<TTSSummarizeConfig> {
  const resp = await audioAdminClient.getSummarizeConfig({});
  return decodeSummarizeConfig(resp.config);
}

export async function listTTSSummarizeModels(): Promise<TTSSummarizeModel[]> {
  const resp = await audioAdminClient.listSummarizeModels({});
  return resp.models.map(decodeSummarizeModel);
}

export async function updateTTSSummarizeConfig(patch: Partial<TTSSummarizeConfig>): Promise<TTSSummarizeConfig> {
  const paths: string[] = [];
  const cfg: Record<string, unknown> = {};
  if (patch.enabled !== undefined) { cfg.enabled = patch.enabled; paths.push("enabled"); }
  if (patch.charThreshold !== undefined) { cfg.charThreshold = patch.charThreshold; paths.push("char_threshold"); }
  if (patch.level !== undefined) { cfg.level = summarizeLevelFromString(patch.level); paths.push("level"); }
  if (patch.model !== undefined) { cfg.model = patch.model; paths.push("model"); }
  if (patch.timeoutSeconds !== undefined) { cfg.timeoutSeconds = patch.timeoutSeconds; paths.push("timeout_seconds"); }
  const resp = await audioAdminClient.updateSummarizeConfig({
    updateMask: create(FieldMaskSchema, { paths }),
    config: cfg,
  });
  return decodeSummarizeConfig(resp.config);
}

export async function reportTTSEvent(event: TTSPlaybackEvent): Promise<void> {
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
