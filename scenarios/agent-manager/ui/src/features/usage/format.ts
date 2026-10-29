// Usage presentation rules in one place. Numbers, durations and plan names come
// from lib/format.ts; this file holds only what is specific to quota windows,
// recommendations and coverage. Every timestamp from the Measures procedures is
// RFC 3339 where empty means unknown.

import type { StatusTone } from "@vrooli/react-component-library/StatusBadge/1";
import { formatRelativeTimeShort, formatStatsDateTime, toValidDate } from "../../lib/dateTime";
import { formatCode, formatPercent, formatPoints, MISSING, subscriptionLabel } from "../../lib/format";

/** A reading older than this cannot support a projection (SPEC T-4). */
const STALE_READING_MS = 6 * 3600_000;

const NAMED_WINDOWS = ["five_hour", "seven_day"];

/** Pools that only restate the window or the provider's slot order carry no name of their own. */
const UNNAMED_POOLS = new Set(["primary", "secondary", ...NAMED_WINDOWS]);

function poolLabel(pool: string): string {
  if (!pool || UNNAMED_POOLS.has(pool)) return "";
  const prefix = NAMED_WINDOWS.find((name) => pool.startsWith(`${name}_`));
  return formatCode(prefix ? pool.slice(prefix.length + 1) : pool);
}

type WindowScope = { provider: string; pool: string; window: string };

function baseWindowTitle(scope: WindowScope): string {
  return [subscriptionLabel(scope.provider, scope.window), poolLabel(scope.pool)].filter(Boolean).join(" · ");
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

/** The identity used in `?window=` and as React keys: provider/pool/window. */
export function windowKey(scope: WindowScope): string {
  return `${scope.provider}/${scope.pool}/${scope.window}`;
}

export interface UsageStatusView {
  label: string;
  /** The chip on one-line surfaces (dashboard strip, window cards). */
  short: string;
  tone: StatusTone;
}

const UNKNOWN_STATUS: UsageStatusView = { label: "Forecast unknown", short: "Unknown", tone: "neutral" };

const STATUS: Record<string, UsageStatusView> = {
  will_exhaust: { label: "Will run out before reset", short: "Will run out", tone: "danger" },
  at_risk: { label: "May run out before reset", short: "May run out", tone: "warning" },
  on_track: { label: "Lasts until reset", short: "On track", tone: "success" },
  idle: { label: "Idle, lasts at this pace", short: "Idle", tone: "neutral" },
};

export function usageStatus(status: string): UsageStatusView {
  return STATUS[status] ?? UNKNOWN_STATUS;
}

/** A 0–1 share: one decimal under 10%, whole percents above ("7.5%", "53%"). */
export function formatShare(share: number): string {
  return formatPercent(share, share < 0.1 ? 1 : 0);
}

export function formatRate(percentPerHour: number | undefined): string {
  return percentPerHour === undefined || !Number.isFinite(percentPerHour) ? MISSING : `${percentPerHour.toFixed(percentPerHour < 1 ? 2 : 1)}%/h`;
}

export function formatAt(value: string): string {
  return value ? formatStatsDateTime(value).replace("N/A", MISSING) : MISSING;
}

const DAY = new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric" });

/** "Oct 30", "Oct 30–31", "Oct 30 – Nov 2": a day-level range for one-line surfaces. */
export function formatDayRange(start: string, end = ""): string {
  const from = toValidDate(start || undefined);
  if (!from) return MISSING;
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
  return value ? formatRelativeTimeShort(value, { now, fallback: MISSING }) : MISSING;
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

/** −48 pts: the saving a recommendation estimates in its window. */
export function formatSaving(points: number): string {
  return `−${formatPoints(points)}`;
}

const UUID = /\b([0-9a-f]{4})[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b/gi;

/** A raw UUID is never a label: "Workflow 5f518188-6df0-… node draft" reads "Workflow 5f51… node draft". */
export function friendlyLabel(label: string): string {
  return label.replace(UUID, "$1…");
}

const TITLE_LIMIT = 64;

/** A target label as a short id and a trimmed title: "76d970de Agent manager token usage" → { shortId, title }. */
export function targetTitle(label: string): { shortId: string; title: string } {
  const match = /^([0-9a-f]{8})\s+(.*)$/s.exec(label);
  const title = friendlyLabel(match?.[2] ?? label).replace(/\s+/g, " ").trim();
  return { shortId: match?.[1] ?? "", title: title.length > TITLE_LIMIT ? `${title.slice(0, title.lastIndexOf(" ", TITLE_LIMIT) > 24 ? title.lastIndexOf(" ", TITLE_LIMIT) : TITLE_LIMIT)}…` : title };
}

/** Evidence text that tells siblings apart: "sub-agent · 00b65371", or the label when it already names the id. */
export function evidenceText(evidence: { id: string; label: string }): string {
  const shortId = evidence.id.length > 12 ? evidence.id.slice(0, 8) : evidence.id;
  if (!evidence.label || evidence.label === evidence.id) return shortId;
  return evidence.label.includes(shortId) ? evidence.label : `${evidence.label} · ${shortId}`;
}

const LEVER_LABELS: Record<string, string> = { model: "Model", context: "Context", fanout: "Fan-out", friction: "Friction", idle: "Idle", image: "Images" };

export function leverLabel(lever: string): string {
  return LEVER_LABELS[lever] ?? formatCode(lever);
}

export function groupByLabel(groupBy: string): string {
  return formatCode(groupBy);
}

/** "scenario:image_tools" → "Image tools (Vrooli scenario)"; "web_search" → "Web search". */
export function meteredLabel(kind: string): string {
  const scenario = /^scenario:(.+)$/.exec(kind)?.[1];
  return scenario ? `${formatCode(scenario)} (Vrooli scenario)` : formatCode(kind);
}

const ROOT_LABELS: Record<string, string> = {
  "web-console-codex-home": "Web console sessions",
  "codex-home": "Default Codex home",
  "claude-home": "Claude Code projects",
  "agent-manager-run-home": "Agent Manager runs",
  configured: "Configured folders",
};

export function coverageRootLabel(kind: string): string {
  return ROOT_LABELS[kind] ?? formatCode(kind);
}

const SOURCE_LABELS: Record<string, string> = { indexed: "Operator session", agent_manager: "Agent Manager run", mixed: "Mixed" };

export function burnerSourceLabel(source: string): string {
  return SOURCE_LABELS[source] ?? formatCode(source || "unknown");
}
