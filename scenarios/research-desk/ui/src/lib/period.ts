/** The periods "What changed" offers: since the last look, or a fixed span. */
export const PERIODS = ["look", "day", "week"] as const;
export type Period = (typeof PERIODS)[number];

const SPAN_MS = { day: 24 * 3_600_000, week: 7 * 24 * 3_600_000 } as const;

/**
 * The start of a period, rounded down to the minute so a page re-rendering
 * keeps asking the same question; undefined for the last look, which the API
 * keeps per commission and topic.
 */
export function periodStart(period: Period, now = Date.now()): Date | undefined {
  if (period === "look") return undefined;
  return new Date(Math.floor((now - SPAN_MS[period]) / 60_000) * 60_000);
}
