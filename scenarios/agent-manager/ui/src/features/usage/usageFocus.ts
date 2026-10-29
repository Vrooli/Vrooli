// The numbers behind the focus band: what the selected window burns, what it can
// afford, and what the listed actions change. Pure functions of the forecast and
// the group savings so the consequence is checkable without rendering.

import type { UsageWindowForecast } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { isDormantWindow, usageStatus, windowKey } from "./format";

const HOUR = 3600_000;
const AT_RISK = new Set(["will_exhaust", "at_risk"]);

export function isAtRisk(status: string): boolean {
  return AT_RISK.has(status);
}

/** Urgency order: will run out (soonest first), then may run out, then the rest by use. */
function urgency(forecast: UsageWindowForecast): number {
  const rank = forecast.status === "will_exhaust" ? 0 : forecast.status === "at_risk" ? 1 : 2;
  const exhaustion = Date.parse(forecast.projectedExhaustionAt) || Number.MAX_SAFE_INTEGER / 4;
  return rank * 1e16 + (rank < 2 ? exhaustion : 1e15 - forecast.usedPercent * 1e12);
}

/** The requested window when it exists, otherwise the most urgent live one. */
export function pickWindow(windows: readonly UsageWindowForecast[], requested: string, now = Date.now()): UsageWindowForecast | undefined {
  const match = windows.find((window) => windowKey(window) === requested);
  if (match) return match;
  const live = windows.filter((window) => !isDormantWindow(window, now) && window.observedAt);
  return [...live].sort((a, b) => urgency(a) - urgency(b))[0];
}

export interface FocusFigures {
  used: number;
  /** Percent of the window burned per hour: the last-24 h rate when it exists, otherwise the projection's rate. */
  burn: number;
  /** True when `burn` is the measured last-24 h rate, false when it is the projection's rate. */
  burnMeasured: boolean;
  /** The most the window can burn per hour and still last until its reset. */
  affordable: number;
  /** Share of the current burn that has to go, 0–1; undefined when the window already lasts. */
  cutNeeded?: number;
  /** Points the listed actions recover over the rest of the window. */
  saving: number;
  /** One honest sentence: where the run-out moves and whether it reaches the reset. */
  consequence: string;
  reachesReset: boolean;
}

const DAY_AT = new Intl.DateTimeFormat("en-US", { weekday: "short", month: "short", day: "numeric", hour: "numeric" });

function shortfall(ms: number): string {
  const hours = Math.round(ms / HOUR);
  return hours >= 36 ? `${Math.round(hours / 24)} days` : `${Math.max(1, hours)} hours`;
}

/** The focus figures for one window and the combined saving of the actions shown for it. */
export function focusFigures(forecast: UsageWindowForecast, saving: number, now = Date.now()): FocusFigures {
  const rate24 = forecast.burn.find((rate) => rate.horizon === "24h" && rate.available)?.percentPerHour;
  const burn = rate24 ?? forecast.centralPercentPerHour;
  const affordable = forecast.lastingPercentPerHour;
  const atRisk = isAtRisk(forecast.status);
  const resets = Date.parse(forecast.resetsAt);
  const hoursLeft = Number.isFinite(resets) ? Math.max(0, (resets - now) / HOUR) : 0;
  const base = { used: forecast.usedPercent, burn, burnMeasured: rate24 !== undefined, affordable, saving };
  const cutNeeded = atRisk && burn > 0 && affordable > 0 ? Math.max(0, 1 - affordable / burn) : undefined;
  if (!atRisk || hoursLeft <= 0 || forecast.centralPercentPerHour <= 0) {
    return { ...base, cutNeeded, consequence: `${usageStatus(forecast.status).label} at the current burn.`, reachesReset: !atRisk };
  }
  const atReset = forecast.usedPercent + forecast.centralPercentPerHour * hoursLeft;
  const after = atReset - saving;
  const resetText = DAY_AT.format(resets);
  if (after <= 100) {
    return { ...base, cutNeeded, consequence: `With these actions the window is projected to end at ${Math.round(Math.max(after, forecast.usedPercent))}% and last until it resets (${resetText}).`, reachesReset: true };
  }
  if (saving <= 0) {
    return { ...base, cutNeeded, consequence: `None of the actions below recovers enough to change the run-out. To last until ${resetText}, the burn has to drop to ${affordable.toFixed(2)}%/h.`, reachesReset: false };
  }
  const newBurn = forecast.centralPercentPerHour - saving / hoursLeft;
  const runOut = now + ((100 - forecast.usedPercent) / newBurn) * HOUR;
  return {
    ...base,
    cutNeeded,
    consequence: `These actions move the run-out to around ${DAY_AT.format(runOut)}, still ${shortfall(resets - runOut)} short of the reset (${resetText}). To last until then, also pause non-urgent work on this plan or move it to another one.`,
    reachesReset: false,
  };
}
