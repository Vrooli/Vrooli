import { timestampDate, type Timestamp } from "@bufbuild/protobuf/wkt";

import { formatDate, formatNumber, formatRelativeTime } from "../i18n/format";

export function toDate(ts: Timestamp | undefined): Date | null {
  if (!ts) return null;
  const date = timestampDate(ts);
  return Number.isNaN(date.getTime()) || date.getTime() <= 0 ? null : date;
}

const STEPS: [Intl.RelativeTimeFormatUnit, number][] = [
  ["second", 60],
  ["minute", 60],
  ["hour", 24],
  ["day", 30],
  ["month", 12],
  ["year", Number.POSITIVE_INFINITY],
];

/** "3 hours ago", "in 20 minutes"; `now` is injectable so tests are stable. */
export function relative(date: Date, now: Date = new Date()): string {
  let value = (date.getTime() - now.getTime()) / 1000;
  for (const [unit, size] of STEPS) {
    if (Math.abs(value) < size) return formatRelativeTime(Math.round(value), unit, { numeric: "auto" });
    value /= size;
  }
  return formatRelativeTime(Math.round(value), "year");
}

export function absolute(date: Date): string {
  return formatDate(date, { dateStyle: "medium", timeStyle: "short" });
}

/** Sizes as an agent experiences them: kilobytes and the bytes/4 token estimate. */
export function kb(bytes: bigint | number): string {
  return `${formatNumber(Number(bytes) / 1000, { maximumFractionDigits: Number(bytes) < 10_000 ? 1 : 0 })} KB`;
}

export function tokens(bytes: bigint | number): string {
  return formatNumber(Math.round(Number(bytes) / 4));
}
