/**
 * Pure presentation helpers for the scoring feature. Kept out of the
 * component files so each of those exports only components
 * (react-refresh constraint) and so the verdict/priority maps stay
 * literal key references the unused-i18n-key audit can see.
 */
import { strings } from "../../consts/strings";
import { formatNumber } from "../../i18n/format";

/** Render points the way the CLI does: one decimal, trailing .0 trimmed. */
export const formatPoints = (value: number): string =>
  formatNumber(value, { maximumFractionDigits: 1 });

const VERDICT_KEYS = {
  fresh: strings.scoring.freshness.verdict.fresh,
  stale: strings.scoring.freshness.verdict.stale,
  unknown: strings.scoring.freshness.verdict.unknown,
} as const;

type VerdictKey = (typeof VERDICT_KEYS)[keyof typeof VERDICT_KEYS];

/**
 * Translation key for a freshness verdict. The wire contract only emits
 * "fresh" | "stale" | "unknown"; anything else degrades to "unknown".
 */
export const verdictKey = (verdict: string): VerdictKey =>
  verdict in VERDICT_KEYS ? VERDICT_KEYS[verdict as keyof typeof VERDICT_KEYS] : VERDICT_KEYS.unknown;

const PRIORITY_KEYS = {
  high: strings.scoring.recommendations.priority.high,
  medium: strings.scoring.recommendations.priority.medium,
  low: strings.scoring.recommendations.priority.low,
} as const;

type PriorityKey = (typeof PRIORITY_KEYS)[keyof typeof PRIORITY_KEYS];

/**
 * Translation key for a recommendation priority, or null for values
 * outside the wire contract (callers fall back to the raw string).
 */
export const priorityKey = (priority: string): PriorityKey | null =>
  priority in PRIORITY_KEYS ? PRIORITY_KEYS[priority as keyof typeof PRIORITY_KEYS] : null;

/** Tailwind classes for the verdict badge, keyed by wire verdict. */
export const verdictBadgeClass = (verdict: string): string => {
  switch (verdict) {
    case "fresh":
      return "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400";
    case "stale":
      return "bg-amber-500/15 text-amber-600 dark:text-amber-400";
    default:
      return "bg-slate-500/15 text-app-muted-foreground";
  }
};
