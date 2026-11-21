import type { TFunction } from "i18next";
import { strings } from "../../consts/strings";
import type {
  CommercialEligibility,
  ReadinessLevel,
} from "../../services/models";

export type Tone = "neutral" | "success" | "warning" | "danger" | "info";

/**
 * Union of every leaf value in the string registry (each leaf is its dotted
 * key path). Label helpers return this instead of `string` so `t()` keeps its
 * compile-time key checking.
 */
export type StringKey = DeepString<typeof strings>;
type DeepString<T> = T extends string
  ? T
  : { [K in keyof T]: DeepString<T[K]> }[keyof T];

const READINESS_TONES: Record<ReadinessLevel, Tone> = {
  available: "success",
  candidate: "info",
  experimental: "warning",
  watch: "neutral",
  unspecified: "neutral",
};

const READINESS_LABEL_KEYS: Record<ReadinessLevel, StringKey> = {
  available: strings.models.readiness.available,
  candidate: strings.models.readiness.candidate,
  experimental: strings.models.readiness.experimental,
  watch: strings.models.readiness.watch,
  unspecified: strings.models.readiness.unspecified,
};

const COMMERCIAL_TONES: Record<CommercialEligibility, Tone> = {
  allowed: "success",
  disallowed: "danger",
  unknown: "warning",
  unspecified: "neutral",
};

const COMMERCIAL_LABEL_KEYS: Record<CommercialEligibility, StringKey> = {
  allowed: strings.models.commercial.allowed,
  disallowed: strings.models.commercial.disallowed,
  unknown: strings.models.commercial.unknown,
  unspecified: strings.models.commercial.unspecified,
};

export const CAPABILITIES = ["stt", "tts", "summarize", "speaker"] as const;

export type Capability = (typeof CAPABILITIES)[number];

export function readinessTone(level: ReadinessLevel): Tone {
  return READINESS_TONES[level];
}

export function readinessLabelKey(level: ReadinessLevel): StringKey {
  return READINESS_LABEL_KEYS[level];
}

export function commercialTone(level: CommercialEligibility): Tone {
  return COMMERCIAL_TONES[level];
}

export function commercialLabelKey(level: CommercialEligibility): StringKey {
  return COMMERCIAL_LABEL_KEYS[level];
}

/**
 * Unknown capabilities fall back to the raw registry code. The registry is
 * intentionally extensible, so a future capability must stay visible instead
 * of rendering an empty cell. Known values never touch the fallback.
 */
export function capabilityLabel(capability: string, t: TFunction): string {
  switch (capability) {
    case "stt":
      return t(strings.models.capabilities.stt);
    case "tts":
      return t(strings.models.capabilities.tts);
    case "summarize":
      return t(strings.models.capabilities.summarize);
    case "speaker":
      return t(strings.models.capabilities.speaker);
    default:
      return capability;
  }
}

export interface LanguageSummary {
  shown: string[];
  overflow: number;
}

export function languageSummary(
  languages: string[],
  limit = 3,
): LanguageSummary {
  const shown = languages.slice(0, limit);
  return { shown, overflow: Math.max(0, languages.length - shown.length) };
}
