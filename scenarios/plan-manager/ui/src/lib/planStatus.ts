/**
 * Status-semantics SSOT for the operator console.
 *
 * The plan-manager domain has four small enums the console renders everywhere:
 * plan status, phase status, staleness tier, and validation verdict. Each gets a
 * consistent, accessible treatment — a Tailwind token *and* a typed label key,
 * never color alone (WCAG: meaning must survive desaturation). UI surfaces import
 * the `*Tone` descriptor and pair the tone classes with an icon + the translated
 * label so the same enum reads identically across pages.
 *
 * Tones map to the design-token palette (`app-success` / `app-warning` /
 * `app-danger` / `app-info` / neutral border) so light/dark parity is automatic.
 */
import {
  PhaseStatus,
  PlanStatus,
  StalenessTier,
  ValidationVerdict,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

import { strings } from "../consts/strings";
import { type StringKey } from "../consts/stringKey";

/** Semantic tone families; each maps to a token-driven class set below. */
export type Tone = "neutral" | "info" | "active" | "success" | "warning" | "danger";

export interface ToneClasses {
  /** Badge/pill background + foreground. */
  badge: string;
  /** Dot/indicator background. */
  dot: string;
  /** Foreground-only (for inline text emphasis). */
  text: string;
}

/**
 * Token-driven class sets per tone. We deliberately use `/15` and `/10` alpha
 * surfaces over the semantic tokens so a single class set works in both light
 * and dark themes (the underlying CSS var flips; the alpha overlay stays
 * legible). Foreground uses the solid token for AA contrast on the tinted chip.
 */
export const TONE_CLASSES: Record<Tone, ToneClasses> = {
  neutral: {
    badge: "bg-app-surface-muted text-app-muted-foreground border border-app-border",
    dot: "bg-app-muted-foreground",
    text: "text-app-muted-foreground",
  },
  info: {
    badge: "bg-app-info/15 text-app-info border border-app-info/30",
    dot: "bg-app-info",
    text: "text-app-info",
  },
  active: {
    badge: "bg-app-primary/15 text-app-primary border border-app-primary/30",
    dot: "bg-app-primary",
    text: "text-app-primary",
  },
  success: {
    badge: "bg-app-success/15 text-app-success border border-app-success/30",
    dot: "bg-app-success",
    text: "text-app-success",
  },
  warning: {
    badge: "bg-app-warning/15 text-app-warning border border-app-warning/30",
    dot: "bg-app-warning",
    text: "text-app-warning",
  },
  danger: {
    badge: "bg-app-danger/15 text-app-danger border border-app-danger/30",
    dot: "bg-app-danger",
    text: "text-app-danger",
  },
};

export interface StatusDescriptor {
  tone: Tone;
  /** Typed i18n key path for the human label. */
  labelKey: StringKey;
}

const PLAN_STATUS: Record<PlanStatus, StatusDescriptor> = {
  [PlanStatus.UNSPECIFIED]: { tone: "neutral", labelKey: strings.planStatus.unspecified },
  [PlanStatus.DRAFT]: { tone: "info", labelKey: strings.planStatus.draft },
  [PlanStatus.ACTIVE]: { tone: "active", labelKey: strings.planStatus.active },
  [PlanStatus.COMPLETE]: { tone: "success", labelKey: strings.planStatus.complete },
  [PlanStatus.ARCHIVED]: { tone: "neutral", labelKey: strings.planStatus.archived },
};

const PHASE_STATUS: Record<PhaseStatus, StatusDescriptor> = {
  [PhaseStatus.UNSPECIFIED]: { tone: "neutral", labelKey: strings.phaseStatus.unspecified },
  [PhaseStatus.TODO]: { tone: "neutral", labelKey: strings.phaseStatus.todo },
  [PhaseStatus.ACTIVE]: { tone: "active", labelKey: strings.phaseStatus.active },
  [PhaseStatus.DONE]: { tone: "success", labelKey: strings.phaseStatus.done },
  [PhaseStatus.BLOCKED]: { tone: "danger", labelKey: strings.phaseStatus.blocked },
};

const STALENESS: Record<StalenessTier, StatusDescriptor> = {
  [StalenessTier.UNSPECIFIED]: { tone: "neutral", labelKey: strings.staleness.unspecified },
  [StalenessTier.FRESH]: { tone: "success", labelKey: strings.staleness.fresh },
  [StalenessTier.LIGHTLY_STALE]: { tone: "warning", labelKey: strings.staleness.lightlyStale },
  [StalenessTier.DEFINITELY_STALE]: { tone: "danger", labelKey: strings.staleness.definitelyStale },
};

const VERDICT: Record<ValidationVerdict, StatusDescriptor> = {
  [ValidationVerdict.UNSPECIFIED]: { tone: "neutral", labelKey: strings.verdict.unspecified },
  [ValidationVerdict.PASS]: { tone: "success", labelKey: strings.verdict.pass },
  [ValidationVerdict.FAIL]: { tone: "danger", labelKey: strings.verdict.fail },
  [ValidationVerdict.UNKNOWN]: { tone: "warning", labelKey: strings.verdict.unknown },
};

export const planStatusDescriptor = (status: PlanStatus): StatusDescriptor => PLAN_STATUS[status];

export const phaseStatusDescriptor = (status: PhaseStatus): StatusDescriptor => PHASE_STATUS[status];

export const stalenessDescriptor = (tier: StalenessTier): StatusDescriptor => STALENESS[tier];

export const verdictDescriptor = (verdict: ValidationVerdict): StatusDescriptor => VERDICT[verdict];

/**
 * Count phases in each status for a plan summary. Returns a stable shape so the
 * Plans list can render per-status counts without re-deriving them per row.
 */
export interface PhaseCounts {
  total: number;
  todo: number;
  active: number;
  done: number;
  blocked: number;
}

export const countPhases = (statuses: readonly PhaseStatus[]): PhaseCounts => {
  const counts: PhaseCounts = { total: statuses.length, todo: 0, active: 0, done: 0, blocked: 0 };
  for (const s of statuses) {
    if (s === PhaseStatus.TODO) counts.todo += 1;
    else if (s === PhaseStatus.ACTIVE) counts.active += 1;
    else if (s === PhaseStatus.DONE) counts.done += 1;
    else if (s === PhaseStatus.BLOCKED) counts.blocked += 1;
  }
  return counts;
};

/**
 * Safe conversion of an int64/bigint wire value to a JS number for display.
 * plan-manager velocity tokens/wall-time arrive as `bigint`; rendering a raw
 * bigint into JSX throws, and values are well within `Number.MAX_SAFE_INTEGER`
 * for any realistic run, so we convert explicitly here (one place) rather than
 * sprinkling `Number(...)` across components.
 */
export const bigintToNumber = (value: bigint): number => {
  if (value > BigInt(Number.MAX_SAFE_INTEGER)) return Number.MAX_SAFE_INTEGER;
  if (value < BigInt(Number.MIN_SAFE_INTEGER)) return Number.MIN_SAFE_INTEGER;
  return Number(value);
};
