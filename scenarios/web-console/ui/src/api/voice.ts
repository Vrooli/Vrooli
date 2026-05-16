// DOC: docs/concepts/ARCHITECTURE.md#voice-streaming
//
// Voice domain home: Connect-RPC client, type definitions, decoders,
// and wrappers for transcription, voice-stream configuration, wake-word
// templates, and speaker verification.

import { createClient } from "@connectrpc/connect";
import { resolveApiBase, buildWsUrl } from "@vrooli/api-base";
import { VoiceService } from "@vrooli/proto-types/web-console/v1/voice/voice_pb";

import type { CapabilityStatus } from "./capabilities";
import { transport } from "./client";
import type { WakeWordTemplate } from "../domains/audio";

export const voiceClient = createClient(VoiceService, transport);

const API_BASE = resolveApiBase({ appendSuffix: true });

function apiBaseToWsBase(apiBase: string): string {
  if (apiBase.startsWith("https://")) return `wss://${apiBase.slice("https://".length)}`;
  if (apiBase.startsWith("http://")) return `ws://${apiBase.slice("http://".length)}`;
  return apiBase;
}

// [REQ:P0-004b] api-base WebSocket Integration
export function buildVoiceStreamWsUrl(language?: string): string {
  const wsBase = apiBaseToWsBase(API_BASE);
  const base = buildWsUrl("/voice/stream", { baseUrl: wsBase });
  if (language) return `${base}${base.includes("?") ? "&" : "?"}language=${encodeURIComponent(language)}`;
  return base;
}

async function blobToBytes(b: Blob): Promise<Uint8Array> {
  return new Uint8Array(await b.arrayBuffer());
}

export async function transcribeAudio(audioBlob: Blob, language?: string): Promise<string> {
  const resp = await voiceClient.transcribe({
    audio: await blobToBytes(audioBlob),
    contentType: audioBlob.type,
    language: language ?? "",
    skipSpeakerVerification: false,
  });
  return resp.text;
}

// Used exclusively by the "Transcribe anyway" retry action after a
// false speaker-verification rejection.
export async function transcribeAudioBypassFilter(
  audioBlob: Blob,
  language?: string,
): Promise<string> {
  const resp = await voiceClient.transcribe({
    audio: await blobToBytes(audioBlob),
    contentType: audioBlob.type,
    language: language ?? "",
    skipSpeakerVerification: true,
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

// Voice streaming configuration (server-side pipeline tuning)
export interface VoiceStreamConfig {
  flushIntervalMs: number;
  minDeltaBytes: number;
  overlapBytes: number;
  persistentMode: boolean;
  wakeWordEnabled: boolean;
  wakeWordThreshold: number;
  segmentSilenceMs: number;
}

function decodeStreamConfig(c: {
  flushIntervalMs: number;
  minDeltaBytes: number;
  overlapBytes: number;
  persistentMode: boolean;
  wakeWordEnabled: boolean;
  wakeWordThreshold: number;
  segmentSilenceMs: number;
} | undefined): VoiceStreamConfig {
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

export async function getVoiceStreamConfig(): Promise<VoiceStreamConfig> {
  const resp = await voiceClient.getStreamConfig({});
  return decodeStreamConfig(resp.config);
}

export async function updateVoiceStreamConfig(
  patch: Partial<VoiceStreamConfig>,
): Promise<VoiceStreamConfig> {
  const req: Record<string, unknown> = {};
  if (patch.flushIntervalMs !== undefined) {
    req.flushIntervalMs = patch.flushIntervalMs;
    req.hasFlushIntervalMs = true;
  }
  if (patch.minDeltaBytes !== undefined) {
    req.minDeltaBytes = patch.minDeltaBytes;
    req.hasMinDeltaBytes = true;
  }
  if (patch.overlapBytes !== undefined) {
    req.overlapBytes = patch.overlapBytes;
    req.hasOverlapBytes = true;
  }
  if (patch.persistentMode !== undefined) {
    req.persistentMode = patch.persistentMode;
    req.hasPersistentMode = true;
  }
  if (patch.wakeWordEnabled !== undefined) {
    req.wakeWordEnabled = patch.wakeWordEnabled;
    req.hasWakeWordEnabled = true;
  }
  if (patch.wakeWordThreshold !== undefined) {
    req.wakeWordThreshold = patch.wakeWordThreshold;
    req.hasWakeWordThreshold = true;
  }
  if (patch.segmentSilenceMs !== undefined) {
    req.segmentSilenceMs = patch.segmentSilenceMs;
    req.hasSegmentSilenceMs = true;
  }
  const resp = await voiceClient.updateStreamConfig(req as Parameters<typeof voiceClient.updateStreamConfig>[0]);
  return decodeStreamConfig(resp.config);
}

// Wake word template configuration
export interface WakeWordConfig {
  configured: boolean;
  template: WakeWordTemplate | null;
}

function decodeWakeWord(cfg: { configured: boolean; templateJson: string } | undefined): WakeWordConfig {
  const configured = cfg?.configured ?? false;
  const tj = cfg?.templateJson ?? "";
  let template: WakeWordTemplate | null = null;
  if (configured && tj) {
    try {
      template = JSON.parse(tj) as WakeWordTemplate;
    } catch {
      template = null;
    }
  }
  return { configured, template };
}

export async function getWakeWordConfig(): Promise<WakeWordConfig> {
  const resp = await voiceClient.getWakeWordConfig({});
  return decodeWakeWord(resp.config);
}

export async function updateWakeWordConfig(template: WakeWordTemplate): Promise<WakeWordConfig> {
  const resp = await voiceClient.updateWakeWordTemplate({ templateJson: JSON.stringify(template) });
  return decodeWakeWord(resp.config);
}

export async function deleteWakeWordConfig(): Promise<WakeWordConfig> {
  const resp = await voiceClient.deleteWakeWordTemplate({});
  return decodeWakeWord(resp.config);
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
  capability: CapabilityStatus;
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

function decodeSpeakerConfig(c: {
  enabled: boolean;
  profileIds: string[];
  threshold: number;
  mode: string;
  rejectBehavior: string;
  fallbackWithoutVerification: boolean;
} | undefined): SpeakerVerificationConfig {
  const mode = (c?.mode ?? "filter") as SpeakerVerificationConfig["mode"];
  const reject = (c?.rejectBehavior ?? "drop") as SpeakerVerificationConfig["rejectBehavior"];
  return {
    enabled: c?.enabled ?? false,
    profileIds: c?.profileIds ?? [],
    threshold: c?.threshold ?? 0,
    mode,
    rejectBehavior: reject,
    fallbackWithoutVerification: c?.fallbackWithoutVerification ?? false,
  };
}

function decodeSpeakerProfile(p: {
  id: string;
  displayName: string;
  createdAt: string;
  updatedAt: string;
  modelName: string;
  embeddingDim: number;
  sampleRate: number;
  enrollmentAudioSeconds: number;
  notes: string;
}): SpeakerVerificationProfile {
  return {
    id: p.id,
    display_name: p.displayName,
    created_at: p.createdAt,
    updated_at: p.updatedAt,
    model_name: p.modelName,
    embedding_dim: p.embeddingDim,
    sample_rate: p.sampleRate,
    enrollment_audio_seconds: p.enrollmentAudioSeconds,
    notes: p.notes,
  };
}

export async function getSpeakerVerificationConfig(): Promise<SpeakerVerificationConfig> {
  const resp = await voiceClient.getSpeakerConfig({});
  return decodeSpeakerConfig(resp.config);
}

export async function updateSpeakerVerificationConfig(
  patch: Partial<SpeakerVerificationConfig>,
): Promise<SpeakerVerificationConfig> {
  const req: Record<string, unknown> = {};
  if (patch.enabled !== undefined) {
    req.enabled = patch.enabled;
    req.hasEnabled = true;
  }
  if (patch.profileIds !== undefined) {
    req.profileIds = patch.profileIds;
    req.hasProfileIds = true;
  }
  if (patch.threshold !== undefined) {
    req.threshold = patch.threshold;
    req.hasThreshold = true;
  }
  if (patch.mode !== undefined) {
    req.mode = patch.mode;
    req.hasMode = true;
  }
  if (patch.rejectBehavior !== undefined) {
    req.rejectBehavior = patch.rejectBehavior;
    req.hasRejectBehavior = true;
  }
  if (patch.fallbackWithoutVerification !== undefined) {
    req.fallbackWithoutVerification = patch.fallbackWithoutVerification;
    req.hasFallbackWithoutVerification = true;
  }
  const resp = await voiceClient.updateSpeakerConfig(req as Parameters<typeof voiceClient.updateSpeakerConfig>[0]);
  return decodeSpeakerConfig(resp.config);
}

export async function getSpeakerVerificationStatus(): Promise<SpeakerVerificationStatusResponse> {
  const resp = await voiceClient.getSpeakerStatus({});
  const st = resp.status;
  if (!st) throw new Error("speaker status response missing status field");
  return {
    config: decodeSpeakerConfig(st.config),
    capability: st.capability as CapabilityStatus,
    capabilityLabel: st.capabilityLabel || undefined,
    resourceReady: st.resourceReady,
    profileConfigured: st.profileConfigured,
    profileExists: st.profileExists,
    profileCount: st.profileCount,
    profiles: st.profiles?.map(decodeSpeakerProfile),
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
    checkedAt: st.checkedAt,
  };
}

export async function listSpeakerVerificationProfiles(): Promise<SpeakerVerificationProfile[]> {
  const resp = await voiceClient.listSpeakerProfiles({});
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
    contentType: args.audioBlob.type,
    profileId: args.profileId ?? "",
    displayName: args.displayName ?? "",
    notes: args.notes ?? "",
  };
  if (args.addToActive !== undefined) {
    req.addToActive = args.addToActive;
    req.hasAddToActive = true;
  }
  if (args.enable !== undefined) {
    req.enable = args.enable;
    req.hasEnable = true;
  }
  const resp = await voiceClient.enrollSpeakerProfile(
    req as Parameters<typeof voiceClient.enrollSpeakerProfile>[0],
  );
  const en = resp.enrollment;
  return {
    enrollment: {
      profile_id: en?.profileId ?? "",
      display_name: en?.displayName ?? "",
      embedding_dim: en?.embeddingDim ?? 0,
      sample_rate: en?.sampleRate ?? 0,
      enrollment_audio_seconds: en?.enrollmentAudioSeconds ?? 0,
      model_name: en?.modelName ?? "",
      created_at: en?.createdAt ?? "",
    },
    config: decodeSpeakerConfig(resp.config),
  };
}

export async function clearSpeakerVerificationProfile(): Promise<SpeakerVerificationConfig> {
  const resp = await voiceClient.clearSpeakerProfileBinding({});
  return decodeSpeakerConfig(resp.config);
}

export async function removeSpeakerVerificationProfile(profileId: string): Promise<SpeakerVerificationConfig> {
  const resp = await voiceClient.removeSpeakerProfile({ profileId });
  return decodeSpeakerConfig(resp.config);
}

export async function deleteSpeakerVerificationProfile(profileId: string): Promise<SpeakerVerificationConfig> {
  const resp = await voiceClient.deleteSpeakerProfile({ profileId });
  return decodeSpeakerConfig(resp.config);
}
