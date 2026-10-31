// The focus band's window choice and its consequence sentence. The window is
// the API's focus mark unless one is requested, and every figure (the combined
// saving, the projected end, run-out, shortfall and cut) is the API's
// UsageWindowOutcome; nothing here recomputes the forecast.

import type { UsageWindowForecast, UsageWindowOutcome } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { usageStatus, windowKey } from "./format";

const AT_RISK = new Set(["will_exhaust", "at_risk"]);

export function isAtRisk(status: string): boolean {
  return AT_RISK.has(status);
}

/** The requested window when it exists, otherwise the one the API marks as the focus (the most urgent live window). */
export function pickWindow(windows: readonly UsageWindowForecast[], requested: string): UsageWindowForecast | undefined {
  return windows.find((window) => windowKey(window) === requested) ?? windows.find((window) => window.focus);
}

const DAY_AT = new Intl.DateTimeFormat("en-US", { weekday: "short", month: "short", day: "numeric", hour: "numeric" });

function shortfall(seconds: number): string {
  const hours = Math.round(seconds / 3600);
  return hours >= 36 ? `${Math.round(hours / 24)} days` : `${Math.max(1, hours)} hours`;
}

/** One honest sentence from the API's projection of the window with its levers applied: where the run-out moves and whether it reaches the reset. */
export function consequence(forecast: UsageWindowForecast, outcome: UsageWindowOutcome | undefined): string {
  if (!outcome?.projected) return `${usageStatus(forecast.status).label} at the current burn.`;
  const resetText = DAY_AT.format(Date.parse(forecast.resetsAt));
  if (outcome.reachesReset) return `With these actions the window is projected to end at ${Math.round(outcome.endPercent)}% and last until it resets (${resetText}).`;
  if (outcome.recoverablePoints <= 0 || !outcome.runOutAt) {
    return `None of the actions below recovers enough to change the run-out. To last until ${resetText}, the burn has to drop to ${forecast.lastingPercentPerHour.toFixed(2)}%/h.`;
  }
  return `These actions move the run-out to around ${DAY_AT.format(Date.parse(outcome.runOutAt))}, still ${shortfall(Number(outcome.shortfallSeconds))} short of the reset (${resetText}). To last until then, also pause non-urgent work on this plan or move it to another one.`;
}
