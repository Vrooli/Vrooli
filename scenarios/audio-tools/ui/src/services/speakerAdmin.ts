// Speaker-admin client: calls audio-tools' own STTAdminService via the
// same-origin Connect transport. No cross-scenario calls — audio-tools
// owns this surface and serves it to its own UI.

import { create } from "@bufbuild/protobuf";
import { createClient as createConnectClient } from "@connectrpc/connect";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";

import { AudioFormat } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";
import { STTAdminService } from "@vrooli/proto-types/audio-tools/v1/stt/stt_admin_pb";
import {
  RejectBehavior,
  SpeakerMode,
} from "@vrooli/proto-types/audio-tools/v1/stt/stt_pb";

import { transport } from "../api/client";

const client = createConnectClient(STTAdminService, transport);

export type SpeakerModeLabel = "off" | "filter" | "advisory";
export type RejectBehaviorLabel = "drop" | "show-muted";

function speakerModeFromLabel(s: SpeakerModeLabel): SpeakerMode {
  switch (s) {
    case "off":
      return SpeakerMode.OFF;
    case "filter":
      return SpeakerMode.FILTER;
    case "advisory":
      return SpeakerMode.ADVISORY;
    default:
      return SpeakerMode.UNSPECIFIED;
  }
}

function speakerModeToLabel(m: SpeakerMode | undefined): SpeakerModeLabel {
  switch (m) {
    case SpeakerMode.OFF:
      return "off";
    case SpeakerMode.FILTER:
      return "filter";
    case SpeakerMode.ADVISORY:
      return "advisory";
    default:
      return "filter";
  }
}

function rejectBehaviorFromLabel(s: RejectBehaviorLabel): RejectBehavior {
  switch (s) {
    case "drop":
      return RejectBehavior.DROP;
    case "show-muted":
      return RejectBehavior.SHOW_MUTED;
    default:
      return RejectBehavior.UNSPECIFIED;
  }
}

function rejectBehaviorToLabel(r: RejectBehavior | undefined): RejectBehaviorLabel {
  switch (r) {
    case RejectBehavior.DROP:
      return "drop";
    case RejectBehavior.SHOW_MUTED:
      return "show-muted";
    default:
      return "drop";
  }
}

export interface SpeakerConfig {
  enabled: boolean;
  profileIds: string[];
  threshold: number;
  mode: SpeakerModeLabel;
  rejectBehavior: RejectBehaviorLabel;
  fallbackWithoutVerification: boolean;
  extractionEnabled: boolean;
  minDecisionSeconds: number;
  scoreSmoothing: number;
}

export interface SpeakerProfile {
  id: string;
  displayName: string;
  createdAt: string;
  modelName: string;
  sampleRate: number;
  clipCount: number;
  totalVoicedSeconds: number;
}

export interface SpeakerProfileClip {
  clipId: string;
  label: string;
  voicedSeconds: number;
  createdAt: string;
}

export interface SpeakerEnrollmentResult {
  profileId: string;
  clipId: string;
  label: string;
  voicedSeconds: number;
  clipCount: number;
  totalVoicedSeconds: number;
}

export interface SpeakerStatus {
  config: SpeakerConfig;
  capability: string;
  capabilityLabel: string;
  resourceReady: boolean;
  profileConfigured: boolean;
  profileExists: boolean;
  profileCount: number;
  profiles: SpeakerProfile[];
}

function decodeConfig(c: {
  enabled?: boolean;
  profileIds?: string[];
  threshold?: number;
  mode?: SpeakerMode;
  rejectBehavior?: RejectBehavior;
  fallbackWithoutVerification?: boolean;
  extractionEnabled?: boolean;
  minDecisionSeconds?: number;
  scoreSmoothing?: number;
} | undefined): SpeakerConfig {
  return {
    enabled: c?.enabled ?? false,
    profileIds: c?.profileIds ?? [],
    threshold: c?.threshold ?? 0,
    mode: speakerModeToLabel(c?.mode),
    rejectBehavior: rejectBehaviorToLabel(c?.rejectBehavior),
    fallbackWithoutVerification: c?.fallbackWithoutVerification ?? false,
    extractionEnabled: c?.extractionEnabled ?? false,
    minDecisionSeconds: c?.minDecisionSeconds ?? 0,
    scoreSmoothing: c?.scoreSmoothing ?? 0,
  };
}

function tsToISO(ts?: { seconds?: bigint | number; nanos?: number }): string {
  if (!ts) return "";
  const seconds = typeof ts.seconds === "bigint" ? Number(ts.seconds) : (ts.seconds ?? 0);
  return new Date(seconds * 1000 + Math.floor((ts.nanos ?? 0) / 1_000_000)).toISOString();
}

function decodeProfile(p: {
  id: string;
  displayName: string;
  modelName: string;
  sampleRate: number;
  clipCount: number;
  totalVoicedSeconds: number;
  createdAt?: { seconds?: bigint | number; nanos?: number };
}): SpeakerProfile {
  return {
    id: p.id,
    displayName: p.displayName,
    createdAt: tsToISO(p.createdAt),
    modelName: p.modelName,
    sampleRate: p.sampleRate,
    clipCount: p.clipCount,
    totalVoicedSeconds: p.totalVoicedSeconds,
  };
}

function decodeClip(c: {
  clipId: string;
  label: string;
  voicedSeconds: number;
  createdAt?: { seconds?: bigint | number; nanos?: number };
}): SpeakerProfileClip {
  return {
    clipId: c.clipId,
    label: c.label,
    voicedSeconds: c.voicedSeconds,
    createdAt: tsToISO(c.createdAt),
  };
}

export async function getSpeakerStatus(): Promise<SpeakerStatus> {
  const resp = await client.getSpeakerStatus({});
  const st = resp.status;
  if (!st) throw new Error("speaker status response missing status");
  return {
    config: decodeConfig(st.config),
    capability: st.capability,
    capabilityLabel: st.capabilityLabel,
    resourceReady: st.resourceReady,
    profileConfigured: st.profileConfigured,
    profileExists: st.profileExists,
    profileCount: st.profileCount,
    profiles: st.profiles.map(decodeProfile),
  };
}

export async function updateSpeakerConfig(patch: Partial<SpeakerConfig>): Promise<SpeakerConfig> {
  const paths: string[] = [];
  const cfg: Record<string, unknown> = {};
  if (patch.enabled !== undefined) { cfg.enabled = patch.enabled; paths.push("enabled"); }
  if (patch.profileIds !== undefined) { cfg.profileIds = patch.profileIds; paths.push("profile_ids"); }
  if (patch.threshold !== undefined) { cfg.threshold = patch.threshold; paths.push("threshold"); }
  if (patch.mode !== undefined) { cfg.mode = speakerModeFromLabel(patch.mode); paths.push("mode"); }
  if (patch.rejectBehavior !== undefined) { cfg.rejectBehavior = rejectBehaviorFromLabel(patch.rejectBehavior); paths.push("reject_behavior"); }
  if (patch.fallbackWithoutVerification !== undefined) { cfg.fallbackWithoutVerification = patch.fallbackWithoutVerification; paths.push("fallback_without_verification"); }
  if (patch.extractionEnabled !== undefined) { cfg.extractionEnabled = patch.extractionEnabled; paths.push("extraction_enabled"); }
  if (patch.minDecisionSeconds !== undefined) { cfg.minDecisionSeconds = patch.minDecisionSeconds; paths.push("min_decision_seconds"); }
  if (patch.scoreSmoothing !== undefined) { cfg.scoreSmoothing = patch.scoreSmoothing; paths.push("score_smoothing"); }
  const resp = await client.updateSpeakerConfig({
    updateMask: create(FieldMaskSchema, { paths }),
    config: cfg,
  });
  return decodeConfig(resp.config);
}

export async function unbindSpeakerProfile(profileId: string): Promise<SpeakerConfig> {
  const resp = await client.unbindSpeakerProfile({ profileId });
  return decodeConfig(resp.config);
}

export async function deleteSpeakerProfile(profileId: string): Promise<SpeakerConfig> {
  const resp = await client.deleteSpeakerProfile({ profileId });
  return decodeConfig(resp.config);
}

export interface EnrollSpeakerArgs {
  audio: Uint8Array;
  format: AudioFormat;
  profileId?: string;
  displayName?: string;
  label?: string;
  addToActive?: boolean;
  enable?: boolean;
}

// enrollSpeakerProfile appends ONE labeled clip to a profile (creating it when
// new). Call it once per recorded clip to build a multi-condition identity.
export async function enrollSpeakerProfile(args: EnrollSpeakerArgs): Promise<SpeakerEnrollmentResult> {
  const req: Record<string, unknown> = {
    audio: args.audio,
    format: args.format,
    profileId: args.profileId ?? "",
    displayName: args.displayName ?? "",
    label: args.label ?? "",
  };
  if (args.addToActive !== undefined) req.addToActive = args.addToActive;
  if (args.enable !== undefined) req.enable = args.enable;
  const resp = await client.enrollSpeakerProfile(req);
  const en = resp.enrollment;
  return {
    profileId: en?.profileId ?? "",
    clipId: en?.clipId ?? "",
    label: en?.label ?? "",
    voicedSeconds: en?.voicedSeconds ?? 0,
    clipCount: en?.clipCount ?? 0,
    totalVoicedSeconds: en?.totalVoicedSeconds ?? 0,
  };
}

export async function listSpeakerProfileClips(profileId: string): Promise<SpeakerProfileClip[]> {
  const resp = await client.listSpeakerProfileClips({ profileId });
  return resp.clips.map(decodeClip);
}

export async function deleteSpeakerProfileClip(
  profileId: string,
  clipId: string,
): Promise<{ deletedProfile: boolean; clipCount: number }> {
  const resp = await client.deleteSpeakerProfileClip({ profileId, clipId });
  return { deletedProfile: resp.deletedProfile, clipCount: resp.clipCount };
}
