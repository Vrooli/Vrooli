// Voice (STT) API client for audio-integration.
//
// Calls web-console's own AudioAdminService + AudioRuntimeService via
// the same-origin Connect transport. The UI never talks to audio-tools
// directly — web-console's API owns the inter-scenario hop.

import { create } from "@bufbuild/protobuf";
import { createClient as createConnectClient } from "@connectrpc/connect";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";

import { transport, API_BASE } from "../../api/client";
import type { WakeWordTemplate } from "../hooks/voice/wakeword/types";
import {
  audioFormatFromString,
  rejectBehaviorFromString,
  rejectBehaviorLabel,
  speakerCapabilityLabel,
  speakerModeFromString,
  speakerModeLabel,
  timestampToISO,
} from "./protomap";

import { AudioAdminService } from "@vrooli/proto-types/web-console/v1/audio_admin/audio_admin_pb";
import { AudioRuntimeService } from "@vrooli/proto-types/web-console/v1/audio_runtime/audio_runtime_pb";
import type {
  SpeakerConfig,
  SpeakerProfile,
  StreamConfig as StreamConfigMsg,
  WakeWordConfig as WakeWordConfigMsg,
  WakeWordTemplate as WakeWordTemplateMsg,
} from "@vrooli/proto-types/web-console/v1/audio_admin/audio_admin_pb";

// Stable shape consumers already use; unchanged.
export interface VoiceStreamConfig {
  flushIntervalMs: number;
  minDeltaBytes: number;
  overlapBytes: number;
  persistentMode: boolean;
  wakeWordEnabled: boolean;
  wakeWordThreshold: number;
  segmentSilenceMs: number;
}

export interface WakeWordConfig {
  configured: boolean;
  template: WakeWordTemplate | null;
}

export interface SpeakerVerificationConfig {
  enabled: boolean;
  profileIds: string[];
  threshold: number;
  mode: "off" | "filter" | "advisory";
  rejectBehavior: "drop" | "show-muted";
  fallbackWithoutVerification: boolean;
}

export interface SpeakerVerificationProfile {
  id: string;
  display_name: string;
  created_at: string;
  updated_at: string;
  model_name: string;
  embedding_dim: number;
  sample_rate: number;
  enrollment_audio_seconds: number;
  notes: string;
}

export interface SpeakerVerificationInfo {
  backend: string;
  model: string;
  device: string;
  sample_rate: number;
  version: string;
  embedding_dim: number;
}

export interface SpeakerVerificationStatusResponse {
  config: SpeakerVerificationConfig;
  capability: string;
  capabilityLabel?: string;
  resourceReady: boolean;
  profileConfigured: boolean;
  profileExists: boolean;
  profileCount: number;
  profiles?: SpeakerVerificationProfile[];
  info?: SpeakerVerificationInfo;
  checkedAt: string;
}

export interface SpeakerVerificationEnrollmentResponse {
  profile_id: string;
  display_name: string;
  embedding_dim: number;
  sample_rate: number;
  enrollment_audio_seconds: number;
  model_name: string;
  created_at: string;
}

export interface SpeakerVerificationEnrollResult {
  enrollment: SpeakerVerificationEnrollmentResponse;
  config: SpeakerVerificationConfig;
}

// Web-console Connect clients, mounted same-origin via the shared transport.
export const audioAdminClient = createConnectClient(AudioAdminService, transport);
export const audioRuntimeClient = createConnectClient(AudioRuntimeService, transport);

async function blobToBytes(b: Blob): Promise<Uint8Array> {
  return new Uint8Array(await b.arrayBuffer());
}

function blobFormat(b: Blob): string {
  const mime = (b.type || "").toLowerCase();
  if (mime.includes("webm")) return "webm";
  if (mime.includes("wav")) return "wav";
  if (mime.includes("mp3") || mime.includes("mpeg")) return "mp3";
  if (mime.includes("ogg")) return "ogg";
  if (mime.includes("flac")) return "flac";
  return mime.split(";")[0]?.split("/")[1] ?? "webm";
}

function decodeStreamConfig(c: StreamConfigMsg | undefined): VoiceStreamConfig {
  return {
    flushIntervalMs: c?.flushIntervalMs ?? 0,
    minDeltaBytes: c?.minDeltaBytes ?? 0,
    overlapBytes: c?.overlapBytes ?? 0,
    persistentMode: c?.persistentMode ?? false,
    wakeWordEnabled: c?.wakeWordEnabled ?? false,
    wakeWordThreshold: c?.wakeWordThreshold ?? 0,
    segmentSilenceMs: c?.segmentSilenceMs ?? 0,
  };
}

function decodeWakeWord(cfg: WakeWordConfigMsg | undefined): WakeWordConfig {
  const configured = cfg?.configured ?? false;
  const tmpl = cfg?.template;
  let template: WakeWordTemplate | null = null;
  if (configured && tmpl) {
    template = {
      label: tmpl.label,
      threshold: tmpl.threshold,
      samples: tmpl.samples.map((s) => ({
        audio: s.audio,
        format: s.format,
        sampleRateHz: s.sampleRateHz,
      })) as unknown as WakeWordTemplate["samples"],
      updatedAt: timestampToISO(tmpl.updatedAt),
    };
  }
  return { configured, template };
}

function decodeSpeakerConfig(c: SpeakerConfig | undefined): SpeakerVerificationConfig {
  return {
    enabled: c?.enabled ?? false,
    profileIds: c?.profileIds ?? [],
    threshold: c?.threshold ?? 0,
    mode: speakerModeLabel(c?.mode),
    rejectBehavior: rejectBehaviorLabel(c?.rejectBehavior),
    fallbackWithoutVerification: c?.fallbackWithoutVerification ?? false,
  };
}

function decodeSpeakerProfile(p: SpeakerProfile): SpeakerVerificationProfile {
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

function apiBaseToWsBase(apiBase: string): string {
  if (apiBase.startsWith("https://")) return `wss://${apiBase.slice("https://".length)}`;
  if (apiBase.startsWith("http://")) return `ws://${apiBase.slice("http://".length)}`;
  return apiBase;
}

/**
 * Build the WebSocket URL for voice streaming. Same-origin — points at
 * web-console's API, which proxies the upstream WebSocket to audio-tools
 * server-side (Phase E of the UI↔own-API migration).
 */
export function buildVoiceStreamWsUrl(language?: string): string {
  const wsBase = apiBaseToWsBase(API_BASE.replace(/\/$/, ""));
  const url = `${wsBase}/api/v1/voice/stream`;
  if (language) return `${url}?language=${encodeURIComponent(language)}`;
  return url;
}

export async function transcribeAudio(audioBlob: Blob, language?: string): Promise<string> {
  const resp = await audioRuntimeClient.transcribe({
    audio: await blobToBytes(audioBlob),
    format: audioFormatFromString(blobFormat(audioBlob)),
    language: language ?? "",
    skipSpeakerVerification: false,
    initialPrompt: "",
  });
  return resp.text;
}

export async function transcribeAudioBypassFilter(audioBlob: Blob, language?: string): Promise<string> {
  const resp = await audioRuntimeClient.transcribe({
    audio: await blobToBytes(audioBlob),
    format: audioFormatFromString(blobFormat(audioBlob)),
    language: language ?? "",
    skipSpeakerVerification: true,
    initialPrompt: "",
  });
  return resp.text;
}

export async function transcribeAudioWithRetry(audioBlob: Blob, maxAttempts = 2, language?: string): Promise<string> {
  let lastError: unknown;
  for (let attempt = 0; attempt < maxAttempts; attempt++) {
    try {
      return await transcribeAudio(audioBlob, language);
    } catch (err) {
      lastError = err;
      if (attempt < maxAttempts - 1) {
        await new Promise((r) => setTimeout(r, 500 * (attempt + 1)));
      }
    }
  }
  throw lastError;
}

export async function getVoiceStreamConfig(): Promise<VoiceStreamConfig> {
  const resp = await audioAdminClient.getStreamConfig({});
  return decodeStreamConfig(resp.config);
}

export async function updateVoiceStreamConfig(patch: Partial<VoiceStreamConfig>): Promise<VoiceStreamConfig> {
  const paths: string[] = [];
  const cfg: Record<string, unknown> = {};
  if (patch.flushIntervalMs !== undefined) { cfg.flushIntervalMs = patch.flushIntervalMs; paths.push("flush_interval_ms"); }
  if (patch.minDeltaBytes !== undefined) { cfg.minDeltaBytes = patch.minDeltaBytes; paths.push("min_delta_bytes"); }
  if (patch.overlapBytes !== undefined) { cfg.overlapBytes = patch.overlapBytes; paths.push("overlap_bytes"); }
  if (patch.persistentMode !== undefined) { cfg.persistentMode = patch.persistentMode; paths.push("persistent_mode"); }
  if (patch.wakeWordEnabled !== undefined) { cfg.wakeWordEnabled = patch.wakeWordEnabled; paths.push("wake_word_enabled"); }
  if (patch.wakeWordThreshold !== undefined) { cfg.wakeWordThreshold = patch.wakeWordThreshold; paths.push("wake_word_threshold"); }
  if (patch.segmentSilenceMs !== undefined) { cfg.segmentSilenceMs = patch.segmentSilenceMs; paths.push("segment_silence_ms"); }
  const resp = await audioAdminClient.updateStreamConfig({
    updateMask: create(FieldMaskSchema, { paths }),
    config: cfg,
  });
  return decodeStreamConfig(resp.config);
}

export async function getWakeWordConfig(): Promise<WakeWordConfig> {
  const resp = await audioAdminClient.getWakeWordConfig({});
  return decodeWakeWord(resp.config);
}

export async function updateWakeWordConfig(template: WakeWordTemplate): Promise<WakeWordConfig> {
  const resp = await audioAdminClient.updateWakeWordTemplate({
    template: template as unknown as WakeWordTemplateMsg,
  });
  return decodeWakeWord(resp.config);
}

export async function deleteWakeWordConfig(): Promise<WakeWordConfig> {
  const resp = await audioAdminClient.deleteWakeWordTemplate({});
  return decodeWakeWord(resp.config);
}

export async function getSpeakerVerificationConfig(): Promise<SpeakerVerificationConfig> {
  const resp = await audioAdminClient.getSpeakerConfig({});
  return decodeSpeakerConfig(resp.config);
}

export async function updateSpeakerVerificationConfig(
  patch: Partial<SpeakerVerificationConfig>,
): Promise<SpeakerVerificationConfig> {
  const paths: string[] = [];
  const cfg: Record<string, unknown> = {};
  if (patch.enabled !== undefined) { cfg.enabled = patch.enabled; paths.push("enabled"); }
  if (patch.profileIds !== undefined) { cfg.profileIds = patch.profileIds; paths.push("profile_ids"); }
  if (patch.threshold !== undefined) { cfg.threshold = patch.threshold; paths.push("threshold"); }
  if (patch.mode !== undefined) { cfg.mode = speakerModeFromString(patch.mode); paths.push("mode"); }
  if (patch.rejectBehavior !== undefined) { cfg.rejectBehavior = rejectBehaviorFromString(patch.rejectBehavior); paths.push("reject_behavior"); }
  if (patch.fallbackWithoutVerification !== undefined) { cfg.fallbackWithoutVerification = patch.fallbackWithoutVerification; paths.push("fallback_without_verification"); }
  const resp = await audioAdminClient.updateSpeakerConfig({
    updateMask: create(FieldMaskSchema, { paths }),
    config: cfg,
  });
  return decodeSpeakerConfig(resp.config);
}

export async function getSpeakerVerificationStatus(): Promise<SpeakerVerificationStatusResponse> {
  const resp = await audioAdminClient.getSpeakerStatus({});
  const st = resp.status;
  if (!st) throw new Error("speaker status response missing status field");
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

export async function listSpeakerVerificationProfiles(): Promise<SpeakerVerificationProfile[]> {
  const resp = await audioAdminClient.listSpeakerProfiles({});
  return resp.profiles.map(decodeSpeakerProfile);
}

export async function enrollSpeakerVerificationProfile(args: {
  audioBlob: Blob;
  profileId?: string;
  displayName?: string;
  notes?: string;
  addToActive?: boolean;
  enable?: boolean;
}): Promise<SpeakerVerificationEnrollResult> {
  const req: Record<string, unknown> = {
    audio: await blobToBytes(args.audioBlob),
    format: audioFormatFromString(blobFormat(args.audioBlob)),
    profileId: args.profileId ?? "",
    displayName: args.displayName ?? "",
    notes: args.notes ?? "",
  };
  if (args.addToActive !== undefined) req.addToActive = args.addToActive;
  if (args.enable !== undefined) req.enable = args.enable;
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

export async function clearSpeakerVerificationProfile(): Promise<SpeakerVerificationConfig> {
  const resp = await audioAdminClient.clearSpeakerProfileBinding({});
  return decodeSpeakerConfig(resp.config);
}

export async function removeSpeakerVerificationProfile(profileId: string): Promise<SpeakerVerificationConfig> {
  const resp = await audioAdminClient.unbindSpeakerProfile({ profileId });
  return decodeSpeakerConfig(resp.config);
}

export async function deleteSpeakerVerificationProfile(profileId: string): Promise<SpeakerVerificationConfig> {
  const resp = await audioAdminClient.deleteSpeakerProfile({ profileId });
  return decodeSpeakerConfig(resp.config);
}

