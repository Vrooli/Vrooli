// Speaker-admin client: calls audio-tools' own STTAdminService via the
// same-origin Connect transport. No cross-scenario calls — audio-tools
// owns this surface and serves it to its own UI.

import { create } from "@bufbuild/protobuf";
import { createClient as createConnectClient } from "@connectrpc/connect";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";

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
}

export interface SpeakerProfile {
  id: string;
  displayName: string;
  createdAt: string;
  modelName: string;
  sampleRate: number;
  enrollmentAudioSeconds: number;
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
} | undefined): SpeakerConfig {
  return {
    enabled: c?.enabled ?? false,
    profileIds: c?.profileIds ?? [],
    threshold: c?.threshold ?? 0,
    mode: speakerModeToLabel(c?.mode),
    rejectBehavior: rejectBehaviorToLabel(c?.rejectBehavior),
    fallbackWithoutVerification: c?.fallbackWithoutVerification ?? false,
    extractionEnabled: c?.extractionEnabled ?? false,
  };
}

function decodeProfile(p: {
  id: string;
  displayName: string;
  modelName: string;
  sampleRate: number;
  enrollmentAudioSeconds: number;
  createdAt?: { seconds?: bigint | number; nanos?: number };
}): SpeakerProfile {
  const ts = p.createdAt;
  let createdAt = "";
  if (ts) {
    const seconds = typeof ts.seconds === "bigint" ? Number(ts.seconds) : (ts.seconds ?? 0);
    createdAt = new Date(seconds * 1000 + Math.floor((ts.nanos ?? 0) / 1_000_000)).toISOString();
  }
  return {
    id: p.id,
    displayName: p.displayName,
    createdAt,
    modelName: p.modelName,
    sampleRate: p.sampleRate,
    enrollmentAudioSeconds: p.enrollmentAudioSeconds,
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
