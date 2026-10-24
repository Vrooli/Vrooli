// Shared proto<->legacy-string translators for audio-integration.
//
// The embed package's public surface (VoiceStreamConfig,
// SpeakerVerificationConfig, etc.) uses plain string-typed fields for
// historical reasons. Now that the audio-tools wire shape uses typed
// enums + google.protobuf.Timestamp, the API client layer translates
// at the boundary so the rest of the embed code (hooks, components)
// stays string-typed and consumers see no churn.

type ProtoEnum = number;
interface TimestampLike { seconds: bigint | number; nanos: number; }

const ProviderTier = { UNSPECIFIED: 0, LOCAL: 1, BYOK: 2, VROOLI: 3 } as const;
const AudioFormat = { UNSPECIFIED: 0, WAV: 1, MP3: 2, FLAC: 3, OGG: 4, WEBM: 5, OPUS: 6, AAC: 7 } as const;
const ResponseFormat = { UNSPECIFIED: 0, MP3: 1, WAV: 2, OPUS: 3, FLAC: 4 } as const;
const SpeakerMode = { UNSPECIFIED: 0, OFF: 1, FILTER: 2, ADVISORY: 3 } as const;
const RejectBehavior = { UNSPECIFIED: 0, DROP: 1, SHOW_MUTED: 2 } as const;
const StreamingMode = { UNSPECIFIED: 0, AUTO: 1, OFF: 2 } as const;
const StrategyPreference = { UNSPECIFIED: 0, AUTO: 1, VAD: 2, OVERLAP: 3, PASSTHROUGH: 4 } as const;
const SummarizeLevel = { UNSPECIFIED: 0, LIGHT: 1, MODERATE: 2, HEAVY: 3 } as const;

export function providerTierLabel(t: ProtoEnum | undefined): string {
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

export function audioFormatFromString(s: string | undefined): ProtoEnum {
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

/** MIME labels used when persisted audio bytes are restored for playback. */
export function audioFormatToMime(f: ProtoEnum | undefined): string {
  switch (f) {
    case AudioFormat.WAV:
      return "audio/wav";
    case AudioFormat.MP3:
      return "audio/mpeg";
    case AudioFormat.FLAC:
      return "audio/flac";
    case AudioFormat.OGG:
      return "audio/ogg";
    case AudioFormat.OPUS:
      return "audio/ogg; codecs=opus";
    case AudioFormat.AAC:
      return "audio/aac";
    case AudioFormat.WEBM:
    default:
      return "audio/webm";
  }
}

/** Translate scenario-local speaker capability enum values without coupling
 * the browser package to a consumer scenario's generated proto namespace. */
export function speakerCapabilityLabel(c: number | undefined): string {
  switch (c) {
    case 1:
      return "available";
    case 2:
      return "degraded";
    case 3:
      return "unavailable";
    case 4:
      return "uninitialized";
    default:
      return "unknown";
  }
}

export function responseFormatFromString(s: string | undefined): ProtoEnum {
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

export function responseFormatLabel(f: ProtoEnum | undefined): string {
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

export function speakerModeFromString(s: string | undefined): ProtoEnum {
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

export function speakerModeLabel(m: ProtoEnum | undefined): "off" | "filter" | "advisory" {
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

export function rejectBehaviorFromString(s: string | undefined): ProtoEnum {
  switch (s) {
    case "drop":
      return RejectBehavior.DROP;
    case "show-muted":
      return RejectBehavior.SHOW_MUTED;
    default:
      return RejectBehavior.UNSPECIFIED;
  }
}

export function rejectBehaviorLabel(r: ProtoEnum | undefined): "drop" | "show-muted" {
  switch (r) {
    case RejectBehavior.DROP:
      return "drop";
    case RejectBehavior.SHOW_MUTED:
      return "show-muted";
    default:
      return "drop";
  }
}

export function streamingModeFromString(s: string | undefined): ProtoEnum {
  switch (s) {
    case "auto":
      return StreamingMode.AUTO;
    case "off":
      return StreamingMode.OFF;
    default:
      return StreamingMode.UNSPECIFIED;
  }
}

export function strategyPreferenceFromString(s: string | undefined): ProtoEnum {
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

export type StreamingModeLabel = "unspecified" | "auto" | "off";

export function streamingModeLabel(m: ProtoEnum | undefined): StreamingModeLabel {
  switch (m) {
    case StreamingMode.AUTO:
      return "auto";
    case StreamingMode.OFF:
      return "off";
    default:
      return "unspecified";
  }
}

export type StrategyPreferenceLabel = "unspecified" | "auto" | "vad" | "overlap" | "passthrough";

export function strategyPreferenceLabel(p: ProtoEnum | undefined): StrategyPreferenceLabel {
  switch (p) {
    case StrategyPreference.AUTO:
      return "auto";
    case StrategyPreference.VAD:
      return "vad";
    case StrategyPreference.OVERLAP:
      return "overlap";
    case StrategyPreference.PASSTHROUGH:
      return "passthrough";
    default:
      return "unspecified";
  }
}

export function summarizeLevelFromString(s: string | undefined): ProtoEnum {
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

export function timestampToISO(ts: TimestampLike | string | undefined): string {
  if (!ts) return "";
  if (typeof ts === "string") return ts;
  const seconds = typeof ts.seconds === "bigint" ? Number(ts.seconds) : ts.seconds;
  const nanos = ts.nanos;
  return new Date(seconds * 1000 + Math.floor(nanos / 1_000_000)).toISOString();
}
