// Usage presentation rules in one place: quota points are estimates ("≈N pts"),
// a missing value is "—" (unknown, never zero), and every timestamp from the
// Measures procedures is RFC 3339 where empty means unknown.

import type { StatusTone } from "@vrooli/react-component-library/StatusBadge/1";
import { formatRelativeTimeShort, formatStatsDateTime, toValidDate } from "../../lib/dateTime";
import { formatHyphenatedLabel } from "../../lib/display";

export const UNKNOWN = "—";

/** A reading older than this cannot support a projection (SPEC T-4). */
const STALE_READING_MS = 6 * 3600_000;

const PROVIDER_LABELS: Record<string, string> = { codex: "Codex", claude: "Claude Code", "claude-code": "Claude Code", opencode: "OpenCode" };

export function providerLabel(provider: string): string {
  return PROVIDER_LABELS[provider] ?? formatHyphenatedLabel(provider);
}

/** Quota scopes are named by the subscription they meter, as the CLI names them. */
const QUOTA_PROVIDER_LABELS: Record<string, string> = { anthropic: "Claude", claude: "Claude", "claude-code": "Claude", openai: "Codex", codex: "Codex" };

const NAMED_WINDOW_MINUTES: Record<string, number> = { five_hour: 300, seven_day: 10_080 };

/** Pools that only restate the window or the provider's slot order carry no name of their own. */
const UNNAMED_POOLS = new Set(["primary", "secondary", ...Object.keys(NAMED_WINDOW_MINUTES)]);

function humanize(value: string): string {
  return formatHyphenatedLabel(value.replace(/_/g, "-"));
}

/** "weekly", "5-hour", "daily"; an unrecognised window is humanised as-is. */
export function windowLabel(window: string): string {
  const digits = /^(\d+)m$/.exec(window)?.[1];
  const minutes = digits ? Number(digits) : NAMED_WINDOW_MINUTES[window];
  if (minutes === 10_080) return "weekly";
  if (minutes === 1_440) return "daily";
  if (minutes && minutes % 60 === 0) return `${minutes / 60}-hour`;
  return humanize(window);
}

function poolLabel(pool: string): string {
  if (!pool || UNNAMED_POOLS.has(pool)) return "";
  const prefix = Object.keys(NAMED_WINDOW_MINUTES).find((name) => pool.startsWith(`${name}_`));
  return humanize(prefix ? pool.slice(prefix.length + 1) : pool);
}

type WindowScope = { provider: string; pool: string; window: string };

function baseWindowTitle(scope: WindowScope): string {
  return [QUOTA_PROVIDER_LABELS[scope.provider] ?? humanize(scope.provider), windowLabel(scope.window), poolLabel(scope.pool)].filter(Boolean).join(" · ");
}

/**
 * "Claude · weekly", "Codex · 5-hour", "Claude · weekly · Opus" when the pool has a name of its own.
 * When another window on the same surface would get the same title, both append their raw pool id
 * ("Codex · weekly · primary"), the same rule as the CLI.
 */
export function windowTitle(scope: WindowScope, siblings: readonly WindowScope[] = []): string {
  const title = baseWindowTitle(scope);
  const collides = siblings.some((other) => windowKey(other) !== windowKey(scope) && baseWindowTitle(other) === title);
  return collides && scope.pool ? `${title} · ${scope.pool}` : title;
}

export function windowKey(scope: { provider: string; pool: string; window: string }): string {
  return `${scope.provider}/${scope.pool}/${scope.window}`;
}

export interface UsageStatusView {
  label: string;
  /** The chip on one-line surfaces (dashboard strip). */
  short: string;
  tone: StatusTone;
}

const UNKNOWN_STATUS: UsageStatusView = { label: "Forecast unknown", short: "Unknown", tone: "neutral" };

const STATUS: Record<string, UsageStatusView> = {
  will_exhaust: { label: "Will run out before reset", short: "Will run out", tone: "danger" },
  at_risk: { label: "May run out before reset", short: "May run out", tone: "warning" },
  on_track: { label: "Lasts until reset", short: "On track", tone: "success" },
};

export function usageStatus(status: string): UsageStatusView {
  return STATUS[status] ?? UNKNOWN_STATUS;
}

/** "≈ 2.2 pts": one decimal under 10, whole points above. */
export function formatPoints(points: number): string {
  if (!Number.isFinite(points)) return UNKNOWN;
  const rounded = Math.abs(points) < 10 ? points.toFixed(1) : Math.round(points).toString();
  return `≈ ${rounded} ${rounded === "1.0" ? "pt" : "pts"}`;
}

export function formatShare(share: number): string {
  if (!Number.isFinite(share)) return UNKNOWN;
  const percent = share * 100;
  return `${percent < 10 ? percent.toFixed(1).replace(/\.0$/, "") : Math.round(percent)}%`;
}

export function formatRate(percentPerHour: number | undefined): string {
  return percentPerHour === undefined || !Number.isFinite(percentPerHour) ? UNKNOWN : `${percentPerHour.toFixed(percentPerHour < 1 ? 2 : 1)}%/h`;
}

export function formatAt(value: string): string {
  return formatStatsDateTime(value || undefined).replace("N/A", UNKNOWN);
}

const DAY = new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric" });

/** "Oct 30", "Oct 30–31", "Oct 30 – Nov 2": a day-level range for one-line surfaces. */
export function formatDayRange(start: string, end = ""): string {
  const from = toValidDate(start || undefined);
  if (!from) return UNKNOWN;
  const to = toValidDate(end || undefined);
  const first = DAY.format(from);
  if (!to) return first;
  const last = DAY.format(to);
  if (last === first) return first;
  return from.getMonth() === to.getMonth() ? `${first}–${to.getDate()}` : `${first} – ${last}`;
}

const ISO_TIMESTAMP = /\b\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(?::\d{2}(?:\.\d+)?)?(?:Z|[+-]\d{2}:\d{2})/g;

/** Server sentences (status reasons, rationales) with their RFC 3339 times shown like every other time. */
export function formatServerText(text: string): string {
  return text.replace(ISO_TIMESTAMP, (stamp) => formatAt(stamp));
}

export function formatAgo(value: string, now = new Date()): string {
  return value ? formatRelativeTimeShort(value, { now, fallback: UNKNOWN }) : UNKNOWN;
}

export function isStale(observedAt: string, now = Date.now()): boolean {
  const at = toValidDate(observedAt || undefined);
  return !at || now - at.getTime() > STALE_READING_MS;
}

/**
 * A window whose newest reading predates its current period: the scheduled
 * reset has passed (or the period began after the reading), so nothing is known
 * about the period in progress. Shown as one muted line, not a forecast card.
 */
export function isDormantWindow(forecast: { observedAt: string; resetsAt: string; windowStart: string }, now = Date.now()): boolean {
  const observed = toValidDate(forecast.observedAt || undefined);
  if (!observed) return false;
  const resets = toValidDate(forecast.resetsAt || undefined);
  const start = toValidDate(forecast.windowStart || undefined);
  return (!!resets && resets.getTime() <= now) || (!!start && observed.getTime() < start.getTime());
}

/** "Codex · 5-hour: no reading since Sep 9, 4:28 PM". */
export function dormantWindowLine(forecast: WindowScope & { observedAt: string }, siblings: readonly WindowScope[] = []): string {
  return `${windowTitle(forecast, siblings)}: no reading since ${formatAt(forecast.observedAt)}`;
}

/** 1.14 B, 4.6 M, 850 K, 12. Token counters are int64 on the wire. */
export function formatTokenCount(value: bigint | number): string {
  const count = Number(value);
  if (count >= 1e9) return `${(count / 1e9).toFixed(2)} B`;
  if (count >= 1e6) return `${(count / 1e6).toFixed(1)} M`;
  if (count >= 1e4) return `${Math.round(count / 1e3)} K`;
  return count.toLocaleString("en-US");
}

/** −N pts/wk, the saving a recommendation estimates in its window. */
export function formatSaving(points: number): string {
  return `−${formatPoints(points).replace("≈ ", "")}/wk`;
}

/** Evidence text that tells siblings apart: "sub-agent · 00b65371", or the label when it already names the id. */
export function evidenceText(evidence: { id: string; label: string }): string {
  const shortId = evidence.id.length > 12 ? evidence.id.slice(0, 8) : evidence.id;
  if (!evidence.label || evidence.label === evidence.id) return shortId;
  return evidence.label.includes(shortId) ? evidence.label : `${evidence.label} · ${shortId}`;
}

/** "sub_agent_points" → "Sub agent points". */
export function metricLabel(name: string): string {
  const words = name.replace(/_+/g, " ").trim();
  return words.charAt(0).toUpperCase() + words.slice(1);
}

const CONFIDENCE_TONES: Record<string, StatusTone> = { high: "success", medium: "warning", low: "neutral" };

export function confidenceTone(confidence: string): StatusTone {
  return CONFIDENCE_TONES[confidence] ?? "neutral";
}

const LEVER_LABELS: Record<string, string> = { model: "Model", context: "Context", fanout: "Fan-out", friction: "Friction", idle: "Idle", image: "Images" };

export function leverLabel(lever: string): string {
  return LEVER_LABELS[lever] ?? formatHyphenatedLabel(lever);
}

const GROUP_LABELS: Record<string, string> = { tool_kind: "Tool kind" };

export function groupByLabel(groupBy: string): string {
  return GROUP_LABELS[groupBy] ?? formatHyphenatedLabel(groupBy);
}

const ROOT_LABELS: Record<string, string> = {
  "web-console-codex-home": "Web console sessions",
  "codex-home": "Default Codex home",
  "claude-home": "Claude Code projects",
  "agent-manager-run-home": "Agent Manager runs",
  configured: "Configured folders",
};

export function coverageRootLabel(kind: string): string {
  return ROOT_LABELS[kind] ?? formatHyphenatedLabel(kind);
}

const SOURCE_LABELS: Record<string, string> = { indexed: "Operator session", agent_manager: "Agent Manager run", mixed: "Mixed" };

export function burnerSourceLabel(source: string): string {
  return SOURCE_LABELS[source] ?? formatHyphenatedLabel(source || "unknown");
}
