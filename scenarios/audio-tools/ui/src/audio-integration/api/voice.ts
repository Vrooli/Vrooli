// Voice (STT) API client for audio-integration.
//
// Binds to audio-tools' STTService Connect handler and exposes a hook-
// shaped voice operation surface for consumer scenarios. WebSocket URL
// construction reads baseUrl from the active AudioToolsClient registered
// via <AudioToolsProvider>.

import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";

import { getActiveAudioToolsClient, useAudioToolsClient, type AudioToolsClient } from "../client";
import type { WakeWordTemplate } from "../hooks/voice/wakeword/types";
import {
  audioFormatFromString,
  rejectBehaviorFromString,
  rejectBehaviorLabel,
  speakerModeFromString,
  speakerModeLabel,
  streamingModeLabel,
  strategyPreferenceLabel,
  streamingModeFromString,
  strategyPreferenceFromString,
  timestampToISO,
  type StreamingModeLabel,
  type StrategyPreferenceLabel,
} from "@vrooli/audio-capture-browser";

import type { SpeakerConfig, SpeakerProfile, StreamConfig as StreamConfigMsg, WakeWordConfig as WakeWordConfigMsg, WakeWordTemplate as WakeWordTemplateMsg } from "@vrooli/proto-types/audio-tools/v1/stt/stt_pb";

export interface VoiceTranscriptionResult {
  text: string;
  providerIdentity?: {
    providerId?: string;
    modelId?: string;
  };
}

// Mirrors audio-tools' StreamConfig (proto) end-to-end. The five advanced
// fields (streamingMode, strategyPreference, vadSilenceMs, overlapWindowMs,
// overlapCommitRuns) are the authoritative source for client-side VAD
// timing — the mic-button ring and segment-boundary emission must derive
// from vadSilenceMs, not from any local default.
export interface VoiceStreamConfig {
  flushIntervalMs: number;
  minDeltaBytes: number;
  overlapBytes: number;
  persistentMode: boolean;
  wakeWordEnabled: boolean;
  wakeWordThreshold: number;
  segmentSilenceMs: number;
  streamingMode: StreamingModeLabel;
  strategyPreference: StrategyPreferenceLabel;
  vadSilenceMs: number;
  overlapWindowMs: number;
  overlapCommitRuns: number;
  /**
   * overlapMaxStallRejects bounds how many consecutive divergence-rejected
   * commit attempts the Overlap-Agree strategy tolerates before it
   * force-commits the longest stable prefix (the stall fallback). 0 disables
   * the guard; the persisted default is 3.
   */
  overlapMaxStallRejects: number;
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
  clip_count: number;
  total_voiced_seconds: number;
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
  clip_id: string;
  label: string;
  voiced_seconds: number;
  clip_count: number;
  total_voiced_seconds: number;
  embedding_dim: number;
  sample_rate: number;
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
    streamingMode: streamingModeLabel(c?.streamingMode),
    strategyPreference: strategyPreferenceLabel(c?.strategyPreference),
    vadSilenceMs: c?.vadSilenceMs ?? 0,
    overlapWindowMs: c?.overlapWindowMs ?? 0,
    overlapCommitRuns: c?.overlapCommitRuns ?? 0,
    overlapMaxStallRejects: c?.overlapMaxStallRejects ?? 0,
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
    clip_count: p.clipCount,
    total_voiced_seconds: p.totalVoicedSeconds,
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
    buildVoiceStreamWsUrl(language?: string, sessionId?: string, resumeToken?: string): string {
      const wsBase = apiBaseToWsBase(client.baseUrl.replace(/\/$/, ""));
      // The embed streams raw 16 kHz mono signed-16-bit PCM, so it declares
      // `format=pcm_s16le` to take the server's ffmpeg-free fast-path (the
      // audioformat substrate's identity decoder). See PcmVoiceStreamProvider
      // + handlers/stt/stream_ws.go::buildStreamStart.
      const params = new URLSearchParams({ format: "pcm_s16le" });
      // Browser automation cannot set arbitrary WebSocket handshake headers.
      // The boot-only server qualification gate accepts this explicit URL pair
      // solely for deterministic browser fault cases; ordinary pages never set
      // it, and a normal server ignores it even if they do.
		if (typeof window !== "undefined") {
			const pageParams = new URLSearchParams(window.location.search);
        const fault = pageParams.get("stt_test_fault");
        if (pageParams.get("stt_test_mode") === "1" && fault) {
          params.set("test_mode", "1");
          params.set("test_fault", fault);
        }
        const engine = pageParams.get("stt_engine_id");
			if (engine) params.set("engine_id", engine);
			if (pageParams.get("stt_test_mode") === "1" && pageParams.get("stt_capture_source") === "virtual") {
				params.set("test_mode", "1");
				params.set("capture_source", "virtual");
			}
		}
      if (language) params.set("language", language);
      if (sessionId) params.set("protocol_version", "2");
      if (sessionId) params.set("session_id", sessionId);
      if (resumeToken) params.set("resume_token", resumeToken);
      return `${wsBase}/api/v1/voice/stream?${params.toString()}`;
    },

    async transcribeAudioDetailed(audioBlob: Blob, language?: string): Promise<VoiceTranscriptionResult> {
      const resp = await client.stt.transcribe({
        audio: await blobToBytes(audioBlob),
        format: audioFormatFromString(blobFormat(audioBlob)),
        language: language ?? "",
        skipSpeakerVerification: false,
        initialPrompt: "",
      });
      const providerIdentity = resp.providerId || resp.modelId
        ? { providerId: resp.providerId || undefined, modelId: resp.modelId || undefined }
        : undefined;
      return { text: resp.text, providerIdentity };
    },

    async transcribeAudio(audioBlob: Blob, language?: string): Promise<string> {
      return (await api.transcribeAudioDetailed(audioBlob, language)).text;
    },

    async transcribeAudioBypassFilter(audioBlob: Blob, language?: string): Promise<string> {
      const resp = await client.stt.transcribe({
        audio: await blobToBytes(audioBlob),
        format: audioFormatFromString(blobFormat(audioBlob)),
        language: language ?? "",
        skipSpeakerVerification: true,
        initialPrompt: "",
      });
      return resp.text;
    },

    async transcribeAudioWithRetry(audioBlob: Blob, maxAttempts = 2, language?: string): Promise<string> {
      return (await api.transcribeAudioWithRetryDetailed(audioBlob, maxAttempts, language)).text;
    },

    async transcribeAudioWithRetryDetailed(audioBlob: Blob, maxAttempts = 2, language?: string): Promise<VoiceTranscriptionResult> {
      let lastError: unknown;
      for (let attempt = 0; attempt < maxAttempts; attempt++) {
        try {
          return await api.transcribeAudioDetailed(audioBlob, language);
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
      const resp = await client.sttAdmin.getStreamConfig({});
      return decodeStreamConfig(resp.config);
    },

    async updateVoiceStreamConfig(patch: Partial<VoiceStreamConfig>): Promise<VoiceStreamConfig> {
      const paths: string[] = [];
      const cfg: Record<string, unknown> = {};
      if (patch.flushIntervalMs !== undefined) { cfg.flushIntervalMs = patch.flushIntervalMs; paths.push("flush_interval_ms"); }
      if (patch.minDeltaBytes !== undefined) { cfg.minDeltaBytes = patch.minDeltaBytes; paths.push("min_delta_bytes"); }
      if (patch.overlapBytes !== undefined) { cfg.overlapBytes = patch.overlapBytes; paths.push("overlap_bytes"); }
      if (patch.persistentMode !== undefined) { cfg.persistentMode = patch.persistentMode; paths.push("persistent_mode"); }
      if (patch.wakeWordEnabled !== undefined) { cfg.wakeWordEnabled = patch.wakeWordEnabled; paths.push("wake_word_enabled"); }
      if (patch.wakeWordThreshold !== undefined) { cfg.wakeWordThreshold = patch.wakeWordThreshold; paths.push("wake_word_threshold"); }
      if (patch.segmentSilenceMs !== undefined) { cfg.segmentSilenceMs = patch.segmentSilenceMs; paths.push("segment_silence_ms"); }
      if (patch.streamingMode !== undefined) { cfg.streamingMode = streamingModeFromString(patch.streamingMode); paths.push("streaming_mode"); }
      if (patch.strategyPreference !== undefined) { cfg.strategyPreference = strategyPreferenceFromString(patch.strategyPreference); paths.push("strategy_preference"); }
      if (patch.vadSilenceMs !== undefined) { cfg.vadSilenceMs = patch.vadSilenceMs; paths.push("vad_silence_ms"); }
      if (patch.overlapWindowMs !== undefined) { cfg.overlapWindowMs = patch.overlapWindowMs; paths.push("overlap_window_ms"); }
      if (patch.overlapCommitRuns !== undefined) { cfg.overlapCommitRuns = patch.overlapCommitRuns; paths.push("overlap_commit_runs"); }
      if (patch.overlapMaxStallRejects !== undefined) { cfg.overlapMaxStallRejects = patch.overlapMaxStallRejects; paths.push("overlap_max_stall_rejects"); }
      const resp = await client.sttAdmin.updateStreamConfig({
        updateMask: create(FieldMaskSchema, { paths }),
        config: cfg,
      });
      return decodeStreamConfig(resp.config);
    },

    async getWakeWordConfig(): Promise<WakeWordConfig> {
      const resp = await client.sttAdmin.getWakeWordConfig({});
      return decodeWakeWord(resp.config);
    },

    async updateWakeWordConfig(template: WakeWordTemplate): Promise<WakeWordConfig> {
      // The embed-side WakeWordTemplate domain shape carries an array
      // of {audio, format, sampleRateHz} samples that are already
      // compatible with the proto message; the proto generator
      // accepts a plain object init.
      const resp = await client.sttAdmin.updateWakeWordTemplate({
        template: template as unknown as WakeWordTemplateMsg,
      });
      return decodeWakeWord(resp.config);
    },

    async deleteWakeWordConfig(): Promise<WakeWordConfig> {
      const resp = await client.sttAdmin.deleteWakeWordTemplate({});
      return decodeWakeWord(resp.config);
    },

    async getSpeakerVerificationConfig(): Promise<SpeakerVerificationConfig> {
      const resp = await client.sttAdmin.getSpeakerConfig({});
      return decodeSpeakerConfig(resp.config);
    },

    async updateSpeakerVerificationConfig(
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
      const resp = await client.sttAdmin.updateSpeakerConfig({
        updateMask: create(FieldMaskSchema, { paths }),
        config: cfg,
      });
      return decodeSpeakerConfig(resp.config);
    },

    async getSpeakerVerificationStatus(): Promise<SpeakerVerificationStatusResponse> {
      const resp = await client.sttAdmin.getSpeakerStatus({});
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
    },

    async listSpeakerVerificationProfiles(): Promise<SpeakerVerificationProfile[]> {
      const resp = await client.sttAdmin.listSpeakerProfiles({});
      return resp.profiles.map(decodeSpeakerProfile);
    },

    async enrollSpeakerVerificationProfile(args: {
      audioBlob: Blob;
      profileId?: string;
      displayName?: string;
      notes?: string;
      label?: string;
      addToActive?: boolean;
      enable?: boolean;
    }): Promise<SpeakerVerificationEnrollResult> {
      const req: Record<string, unknown> = {
        audio: await blobToBytes(args.audioBlob),
        format: audioFormatFromString(blobFormat(args.audioBlob)),
        profileId: args.profileId ?? "",
        displayName: args.displayName ?? "",
        notes: args.notes ?? "",
        label: args.label ?? "",
      };
      if (args.addToActive !== undefined) req.addToActive = args.addToActive;
      if (args.enable !== undefined) req.enable = args.enable;
      const resp = await client.sttAdmin.enrollSpeakerProfile(
        req,
      );
      const en = resp.enrollment;
      return {
        enrollment: {
          profile_id: en?.profileId ?? "",
          clip_id: en?.clipId ?? "",
          label: en?.label ?? "",
          voiced_seconds: en?.voicedSeconds ?? 0,
          clip_count: en?.clipCount ?? 0,
          total_voiced_seconds: en?.totalVoicedSeconds ?? 0,
          embedding_dim: en?.embeddingDim ?? 0,
          sample_rate: en?.sampleRate ?? 0,
          model_name: en?.modelName ?? "",
          created_at: timestampToISO(en?.createdAt),
        },
        config: decodeSpeakerConfig(resp.config),
      };
    },

    async clearSpeakerVerificationProfile(): Promise<SpeakerVerificationConfig> {
      const resp = await client.sttAdmin.clearSpeakerProfileBinding({});
      return decodeSpeakerConfig(resp.config);
    },

    async removeSpeakerVerificationProfile(profileId: string): Promise<SpeakerVerificationConfig> {
      const resp = await client.sttAdmin.unbindSpeakerProfile({ profileId });
      return decodeSpeakerConfig(resp.config);
    },

    async deleteSpeakerVerificationProfile(profileId: string): Promise<SpeakerVerificationConfig> {
      const resp = await client.sttAdmin.deleteSpeakerProfile({ profileId });
      return decodeSpeakerConfig(resp.config);
    },
  };
  return api;
}

/** Convenience hook: pulls the active client from context and binds Voice ops. */
export function useVoiceApi() {
  return createVoiceApi(useAudioToolsClient());
}

// Module-level singleton bound to whichever AudioToolsClient is currently
// registered via <AudioToolsProvider>. Cached per client identity so that
// switching clients (e.g. in tests) rebinds the api surface.
let _lazyClient: AudioToolsClient | null = null;
let _lazyApi: ReturnType<typeof createVoiceApi> | null = null;
function lazy() {
  const client = getActiveAudioToolsClient();
  if (_lazyApi === null || _lazyClient !== client) {
    _lazyClient = client;
    _lazyApi = createVoiceApi(client);
  }
  return _lazyApi;
}

export function buildVoiceStreamWsUrl(language?: string, sessionId?: string, resumeToken?: string): string { return lazy().buildVoiceStreamWsUrl(language, sessionId, resumeToken); }
export function transcribeAudio(audioBlob: Blob, language?: string): Promise<string> { return lazy().transcribeAudio(audioBlob, language); }
export function transcribeAudioDetailed(audioBlob: Blob, language?: string): Promise<VoiceTranscriptionResult> { return lazy().transcribeAudioDetailed(audioBlob, language); }
export function transcribeAudioBypassFilter(audioBlob: Blob, language?: string): Promise<string> { return lazy().transcribeAudioBypassFilter(audioBlob, language); }
export function transcribeAudioWithRetry(audioBlob: Blob, maxAttempts = 2, language?: string): Promise<string> { return lazy().transcribeAudioWithRetry(audioBlob, maxAttempts, language); }
export function transcribeAudioWithRetryDetailed(audioBlob: Blob, maxAttempts = 2, language?: string): Promise<VoiceTranscriptionResult> { return lazy().transcribeAudioWithRetryDetailed(audioBlob, maxAttempts, language); }
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
