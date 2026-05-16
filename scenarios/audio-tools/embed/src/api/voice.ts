// Voice (STT) API client for @audio-tools/embed.
//
// Mirrors the operation surface of web-console's api/voice.ts but binds to
// audio-tools' STTService Connect handler. WebSocket URL construction uses
// audio-tools' base URL injected via window.__AUDIO_TOOLS_URL__.

import { useAudioToolsClient, type AudioToolsClient } from "../client";
import type { WakeWordTemplate } from "../hooks/voice/wakeword/types";

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
  return mime.split(";")[0]?.split("/")?.[1] ?? "webm";
}

function decodeStreamConfig(c:
  | {
      flushIntervalMs: number;
      minDeltaBytes: number;
      overlapBytes: number;
      persistentMode: boolean;
      wakeWordEnabled: boolean;
      wakeWordThreshold: number;
      segmentSilenceMs: number;
    }
  | undefined,
): VoiceStreamConfig {
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

function decodeSpeakerConfig(c:
  | {
      enabled: boolean;
      profileIds: string[];
      threshold: number;
      mode: string;
      rejectBehavior: string;
      fallbackWithoutVerification: boolean;
    }
  | undefined,
): SpeakerVerificationConfig {
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

function apiBaseToWsBase(apiBase: string): string {
  if (apiBase.startsWith("https://")) return `wss://${apiBase.slice("https://".length)}`;
  if (apiBase.startsWith("http://")) return `ws://${apiBase.slice("http://".length)}`;
  return apiBase;
}

export function createVoiceApi(client: AudioToolsClient) {
  const api = {
    buildVoiceStreamWsUrl(language?: string): string {
      const wsBase = apiBaseToWsBase(client.baseUrl.replace(/\/$/, ""));
      const url = `${wsBase}/api/v1/voice/stream`;
      if (language) return `${url}?language=${encodeURIComponent(language)}`;
      return url;
    },

    async transcribeAudio(audioBlob: Blob, language?: string): Promise<string> {
      const resp = await client.stt.transcribe({
        audio: await blobToBytes(audioBlob),
        format: blobFormat(audioBlob),
        language: language ?? "",
        skipSpeakerVerification: false,
        initialPrompt: "",
      });
      return resp.text;
    },

    async transcribeAudioBypassFilter(audioBlob: Blob, language?: string): Promise<string> {
      const resp = await client.stt.transcribe({
        audio: await blobToBytes(audioBlob),
        format: blobFormat(audioBlob),
        language: language ?? "",
        skipSpeakerVerification: true,
        initialPrompt: "",
      });
      return resp.text;
    },

    async transcribeAudioWithRetry(audioBlob: Blob, maxAttempts = 2, language?: string): Promise<string> {
      let lastError: unknown;
      for (let attempt = 0; attempt < maxAttempts; attempt++) {
        try {
          return await api.transcribeAudio(audioBlob, language);
        } catch (err) {
          lastError = err;
          if (attempt < maxAttempts - 1) {
            await new Promise((r) => setTimeout(r, 500 * (attempt + 1)));
          }
        }
      }
      throw lastError;
    },

    async getVoiceStreamConfig(): Promise<VoiceStreamConfig> {
      const resp = await client.stt.getStreamConfig({});
      return decodeStreamConfig(resp.config);
    },

    async updateVoiceStreamConfig(patch: Partial<VoiceStreamConfig>): Promise<VoiceStreamConfig> {
      const req: Record<string, unknown> = {};
      if (patch.flushIntervalMs !== undefined) { req.flushIntervalMs = patch.flushIntervalMs; req.hasFlushIntervalMs = true; }
      if (patch.minDeltaBytes !== undefined) { req.minDeltaBytes = patch.minDeltaBytes; req.hasMinDeltaBytes = true; }
      if (patch.overlapBytes !== undefined) { req.overlapBytes = patch.overlapBytes; req.hasOverlapBytes = true; }
      if (patch.persistentMode !== undefined) { req.persistentMode = patch.persistentMode; req.hasPersistentMode = true; }
      if (patch.wakeWordEnabled !== undefined) { req.wakeWordEnabled = patch.wakeWordEnabled; req.hasWakeWordEnabled = true; }
      if (patch.wakeWordThreshold !== undefined) { req.wakeWordThreshold = patch.wakeWordThreshold; req.hasWakeWordThreshold = true; }
      if (patch.segmentSilenceMs !== undefined) { req.segmentSilenceMs = patch.segmentSilenceMs; req.hasSegmentSilenceMs = true; }
      const resp = await client.stt.updateStreamConfig(req as Parameters<typeof client.stt.updateStreamConfig>[0]);
      return decodeStreamConfig(resp.config);
    },

    async getWakeWordConfig(): Promise<WakeWordConfig> {
      const resp = await client.stt.getWakeWordConfig({});
      return decodeWakeWord(resp.config);
    },

    async updateWakeWordConfig(template: WakeWordTemplate): Promise<WakeWordConfig> {
      const resp = await client.stt.updateWakeWordTemplate({ templateJson: JSON.stringify(template) });
      return decodeWakeWord(resp.config);
    },

    async deleteWakeWordConfig(): Promise<WakeWordConfig> {
      const resp = await client.stt.deleteWakeWordTemplate({});
      return decodeWakeWord(resp.config);
    },

    async getSpeakerVerificationConfig(): Promise<SpeakerVerificationConfig> {
      const resp = await client.stt.getSpeakerConfig({});
      return decodeSpeakerConfig(resp.config);
    },

    async updateSpeakerVerificationConfig(
      patch: Partial<SpeakerVerificationConfig>,
    ): Promise<SpeakerVerificationConfig> {
      const req: Record<string, unknown> = {};
      if (patch.enabled !== undefined) { req.enabled = patch.enabled; req.hasEnabled = true; }
      if (patch.profileIds !== undefined) { req.profileIds = patch.profileIds; req.hasProfileIds = true; }
      if (patch.threshold !== undefined) { req.threshold = patch.threshold; req.hasThreshold = true; }
      if (patch.mode !== undefined) { req.mode = patch.mode; req.hasMode = true; }
      if (patch.rejectBehavior !== undefined) { req.rejectBehavior = patch.rejectBehavior; req.hasRejectBehavior = true; }
      if (patch.fallbackWithoutVerification !== undefined) { req.fallbackWithoutVerification = patch.fallbackWithoutVerification; req.hasFallbackWithoutVerification = true; }
      const resp = await client.stt.updateSpeakerConfig(req as Parameters<typeof client.stt.updateSpeakerConfig>[0]);
      return decodeSpeakerConfig(resp.config);
    },

    async getSpeakerVerificationStatus(): Promise<SpeakerVerificationStatusResponse> {
      const resp = await client.stt.getSpeakerStatus({});
      const st = resp.status;
      if (!st) throw new Error("speaker status response missing status field");
      return {
        config: decodeSpeakerConfig(st.config),
        capability: st.capability,
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
    },

    async listSpeakerVerificationProfiles(): Promise<SpeakerVerificationProfile[]> {
      const resp = await client.stt.listSpeakerProfiles({});
      return resp.profiles.map(decodeSpeakerProfile);
    },

    async enrollSpeakerVerificationProfile(args: {
      audioBlob: Blob;
      profileId?: string;
      displayName?: string;
      notes?: string;
      addToActive?: boolean;
      enable?: boolean;
    }): Promise<SpeakerVerificationEnrollResult> {
      const req: Record<string, unknown> = {
        audio: await blobToBytes(args.audioBlob),
        format: blobFormat(args.audioBlob),
        profileId: args.profileId ?? "",
        displayName: args.displayName ?? "",
        notes: args.notes ?? "",
      };
      if (args.addToActive !== undefined) { req.addToActive = args.addToActive; req.hasAddToActive = true; }
      if (args.enable !== undefined) { req.enable = args.enable; req.hasEnable = true; }
      const resp = await client.stt.enrollSpeakerProfile(
        req as Parameters<typeof client.stt.enrollSpeakerProfile>[0],
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
    },

    async clearSpeakerVerificationProfile(): Promise<SpeakerVerificationConfig> {
      const resp = await client.stt.clearSpeakerProfileBinding({});
      return decodeSpeakerConfig(resp.config);
    },

    async removeSpeakerVerificationProfile(profileId: string): Promise<SpeakerVerificationConfig> {
      const resp = await client.stt.removeSpeakerProfile({ profileId });
      return decodeSpeakerConfig(resp.config);
    },

    async deleteSpeakerVerificationProfile(profileId: string): Promise<SpeakerVerificationConfig> {
      const resp = await client.stt.deleteSpeakerProfile({ profileId });
      return decodeSpeakerConfig(resp.config);
    },
  };
  return api;
}

/** Convenience hook: pulls the active client from context and binds Voice ops. */
export function useVoiceApi() {
  return createVoiceApi(useAudioToolsClient());
}

// Lazy module-level singleton bound to the default client (window.__AUDIO_TOOLS_URL__).
import { createAudioToolsClient } from "../client";
let _lazyApi: ReturnType<typeof createVoiceApi> | null = null;
function lazy() {
  if (_lazyApi === null) _lazyApi = createVoiceApi(createAudioToolsClient());
  return _lazyApi;
}

export function buildVoiceStreamWsUrl(language?: string): string { return lazy().buildVoiceStreamWsUrl(language); }
export function transcribeAudio(audioBlob: Blob, language?: string): Promise<string> { return lazy().transcribeAudio(audioBlob, language); }
export function transcribeAudioBypassFilter(audioBlob: Blob, language?: string): Promise<string> { return lazy().transcribeAudioBypassFilter(audioBlob, language); }
export function transcribeAudioWithRetry(audioBlob: Blob, maxAttempts = 2, language?: string): Promise<string> { return lazy().transcribeAudioWithRetry(audioBlob, maxAttempts, language); }
export function getVoiceStreamConfig(): Promise<VoiceStreamConfig> { return lazy().getVoiceStreamConfig(); }
export function updateVoiceStreamConfig(patch: Partial<VoiceStreamConfig>): Promise<VoiceStreamConfig> { return lazy().updateVoiceStreamConfig(patch); }
export function getWakeWordConfig(): Promise<WakeWordConfig> { return lazy().getWakeWordConfig(); }
export function updateWakeWordConfig(template: WakeWordTemplate): Promise<WakeWordConfig> { return lazy().updateWakeWordConfig(template); }
export function deleteWakeWordConfig(): Promise<WakeWordConfig> { return lazy().deleteWakeWordConfig(); }
export function getSpeakerVerificationConfig(): Promise<SpeakerVerificationConfig> { return lazy().getSpeakerVerificationConfig(); }
export function updateSpeakerVerificationConfig(patch: Partial<SpeakerVerificationConfig>): Promise<SpeakerVerificationConfig> { return lazy().updateSpeakerVerificationConfig(patch); }
export function getSpeakerVerificationStatus(): Promise<SpeakerVerificationStatusResponse> { return lazy().getSpeakerVerificationStatus(); }
export function listSpeakerVerificationProfiles(): Promise<SpeakerVerificationProfile[]> { return lazy().listSpeakerVerificationProfiles(); }
export function enrollSpeakerVerificationProfile(args: {
  audioBlob: Blob;
  profileId?: string;
  displayName?: string;
  notes?: string;
  addToActive?: boolean;
  enable?: boolean;
}): Promise<SpeakerVerificationEnrollResult> { return lazy().enrollSpeakerVerificationProfile(args); }
export function clearSpeakerVerificationProfile(): Promise<SpeakerVerificationConfig> { return lazy().clearSpeakerVerificationProfile(); }
export function removeSpeakerVerificationProfile(profileId: string): Promise<SpeakerVerificationConfig> { return lazy().removeSpeakerVerificationProfile(profileId); }
export function deleteSpeakerVerificationProfile(profileId: string): Promise<SpeakerVerificationConfig> { return lazy().deleteSpeakerVerificationProfile(profileId); }
