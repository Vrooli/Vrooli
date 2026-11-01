/**
 * Turning a provider reading into rows. The rows come from whatever windows the
 * API returned: no duration is special here except for how "a week" is named.
 * A harness's declared windows only reserve card height and loading placeholders.
 */
import { formatLongDuration, formatShortDuration } from "../../lib/localeFormat";
import type { AgentUsage, AgentUsageWindow } from "./useAgentUsage";

/** A reading older than this no longer describes the account. Matches Agent Manager's quota freshness. */
const MAX_READING_AGE_MS = 6 * 60 * 60_000;
const FUTURE_TOLERANCE_MS = 60_000;
const WEEK_MINUTES = 10_080;

export type UsageTone = "normal" | "hot" | "full";

export const toneFor = (usedPercent: number): UsageTone => (usedPercent >= 95 ? "full" : usedPercent >= 80 ? "hot" : "normal");

export interface WindowNames { short: string; long: string }

/** "5h" and "5 hours"; a week reads "Week" and "Weekly". Words go through the caller's i18n. */
export function windowNames(minutes: number, words: { week: string; weekly: string }, locale: string | undefined): WindowNames {
  if (minutes === WEEK_MINUTES) return { short: words.week, long: words.weekly };
  const ms = minutes * 60_000;
  return { short: formatShortDuration(ms, locale), long: formatLongDuration(ms, locale) };
}

const readable = (window: AgentUsageWindow): boolean =>
  window.windowMinutes > 0 && Number.isFinite(window.usedPercent) && window.usedPercent >= 0 && window.usedPercent <= 100;

/** Too old, from the future, or past its own reset: the number is no longer true. */
export function windowIsStale(window: AgentUsageWindow, usage: AgentUsage, now = Date.now()): boolean {
  const observed = Date.parse(window.observedAt || usage.observedAt);
  if (!Number.isFinite(observed) || observed > now + FUTURE_TOLERANCE_MS || now - observed > MAX_READING_AGE_MS) return true;
  const resets = window.resetsAt ? Date.parse(window.resetsAt) : NaN;
  return Number.isFinite(resets) && resets <= now;
}

export interface UsageRow {
  window: AgentUsageWindow;
  stale: boolean;
  /** Other quota pools report this duration too; Manage lists them. */
  otherPools: number;
}

/**
 * One row per duration, shortest first. Pools are never added together: when
 * several report a duration the card shows the tightest, because that one stops
 * the session first, and says that others exist.
 */
export function usageRows(usage: AgentUsage | null | undefined, now = Date.now()): UsageRow[] {
  if (!usage || usage.match === "unavailable") return [];
  const byDuration = new Map<number, AgentUsageWindow[]>();
  for (const window of usage.windows) {
    if (!readable(window)) continue;
    byDuration.set(window.windowMinutes, [...(byDuration.get(window.windowMinutes) ?? []), window]);
  }
  return [...byDuration.entries()].sort(([a], [b]) => a - b).map(([, windows]) => {
    const fresh = windows.filter((window) => !windowIsStale(window, usage, now));
    const tightest = [...(fresh.length > 0 ? fresh : windows)].sort((a, b) => b.usedPercent - a.usedPercent)[0] as AgentUsageWindow;
    return { window: tightest, stale: fresh.length === 0, otherPools: windows.length - 1 };
  });
}

/** Each pool's readable windows for Manage, shortest first. */
export function usagePools(usage: AgentUsage): { pool: string; windows: AgentUsageWindow[] }[] {
  const pools = [...new Set(usage.windows.filter(readable).map((window) => window.pool))].sort();
  return pools.map((pool) => ({
    pool,
    windows: usage.windows.filter((window) => readable(window) && window.pool === pool).sort((a, b) => a.windowMinutes - b.windowMinutes),
  }));
}

/**
 * How many usage rows every card reserves: the largest declared count among
 * visible harnesses, or a larger reading when a provider reports more than it
 * declared. Every card uses the same number, so cards are one height whether
 * loading, loaded or without usage.
 */
export function reservedUsageRows(declaredCounts: readonly number[], usages: readonly (AgentUsage | null | undefined)[]): number {
  return Math.max(1, ...declaredCounts, ...usages.map((usage) => usageRows(usage).length));
}
