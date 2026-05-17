// Shared proto<->legacy-string translators for audio-integration.
//
// Every enum in this file is web-console-owned (proto schema:
// packages/proto/schemas/web-console/v1/audio_common). The UI never
// imports proto types from audio-tools — those crossings happen
// server-side via internal/audioports.
//
// The embed package's public surface uses plain string-typed fields for
// historical reasons; the API client layer translates at the boundary.

import {
  AudioFormat,
  ProviderTier,
  RejectBehavior,
  ResponseFormat,
  SpeakerCapability,
  SpeakerMode,
  StreamingMode,
  StrategyPreference,
  SummarizeLevel,
} from "@vrooli/proto-types/web-console/v1/audio_common/audio_common_pb";
import type { Timestamp } from "@bufbuild/protobuf/wkt";

export function providerTierLabel(t: ProviderTier | undefined): string {
  switch (t) {
    case ProviderTier.LOCAL:
      return "local";
    case ProviderTier.BYOK:
      return "byok";
    case ProviderTier.VROOLI:
      return "vrooli";
    default:
      return "";
  }
}

export function audioFormatFromString(s: string | undefined): AudioFormat {
  switch (s) {
    case "wav":
      return AudioFormat.WAV;
    case "mp3":
      return AudioFormat.MP3;
    case "flac":
      return AudioFormat.FLAC;
    case "ogg":
      return AudioFormat.OGG;
    case "webm":
      return AudioFormat.WEBM;
    case "opus":
      return AudioFormat.OPUS;
    default:
      return AudioFormat.UNSPECIFIED;
  }
}

export function responseFormatFromString(s: string | undefined): ResponseFormat {
  switch (s) {
    case "mp3":
      return ResponseFormat.MP3;
    case "wav":
      return ResponseFormat.WAV;
    case "opus":
      return ResponseFormat.OPUS;
    case "flac":
      return ResponseFormat.FLAC;
    default:
      return ResponseFormat.UNSPECIFIED;
  }
}

export function responseFormatLabel(f: ResponseFormat | undefined): string {
  switch (f) {
    case ResponseFormat.MP3:
      return "mp3";
    case ResponseFormat.WAV:
      return "wav";
    case ResponseFormat.OPUS:
      return "opus";
    case ResponseFormat.FLAC:
      return "flac";
    default:
      return "";
  }
}

export function speakerModeFromString(s: string | undefined): SpeakerMode {
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

export function speakerModeLabel(m: SpeakerMode | undefined): "off" | "filter" | "advisory" {
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

export function rejectBehaviorFromString(s: string | undefined): RejectBehavior {
  switch (s) {
    case "drop":
      return RejectBehavior.DROP;
    case "show-muted":
      return RejectBehavior.SHOW_MUTED;
    default:
      return RejectBehavior.UNSPECIFIED;
  }
}

export function rejectBehaviorLabel(r: RejectBehavior | undefined): "drop" | "show-muted" {
  switch (r) {
    case RejectBehavior.DROP:
      return "drop";
    case RejectBehavior.SHOW_MUTED:
      return "show-muted";
    default:
      return "drop";
  }
}

export function streamingModeFromString(s: string | undefined): StreamingMode {
  switch (s) {
    case "auto":
      return StreamingMode.AUTO;
    case "off":
      return StreamingMode.OFF;
    default:
      return StreamingMode.UNSPECIFIED;
  }
}

export function strategyPreferenceFromString(s: string | undefined): StrategyPreference {
  switch (s) {
    case "auto":
      return StrategyPreference.AUTO;
    case "vad":
      return StrategyPreference.VAD;
    case "overlap":
      return StrategyPreference.OVERLAP;
    case "passthrough":
      return StrategyPreference.PASSTHROUGH;
    default:
      return StrategyPreference.UNSPECIFIED;
  }
}

export function summarizeLevelFromString(s: string | undefined): SummarizeLevel {
  switch (s) {
    case "light":
      return SummarizeLevel.LIGHT;
    case "moderate":
      return SummarizeLevel.MODERATE;
    case "heavy":
      return SummarizeLevel.HEAVY;
    default:
      return SummarizeLevel.UNSPECIFIED;
  }
}

export function speakerCapabilityLabel(c: SpeakerCapability | undefined): string {
  switch (c) {
    case SpeakerCapability.AVAILABLE:
      return "available";
    case SpeakerCapability.DEGRADED:
      return "degraded";
    case SpeakerCapability.UNAVAILABLE:
      return "unavailable";
    case SpeakerCapability.UNINITIALIZED:
      return "uninitialized";
    default:
      return "unknown";
  }
}

export function timestampToISO(ts: Timestamp | string | undefined): string {
  if (!ts) return "";
  if (typeof ts === "string") return ts;
  const seconds = typeof ts.seconds === "bigint" ? Number(ts.seconds) : ts.seconds;
  const nanos = ts.nanos;
  return new Date(seconds * 1000 + Math.floor(nanos / 1_000_000)).toISOString();
}
