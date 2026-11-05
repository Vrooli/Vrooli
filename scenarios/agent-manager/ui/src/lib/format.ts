// The single owner of how numbers, durations and provider names read in the UI.
// A missing value is always "—": unknown is never rendered as zero or "N/A".

import { RunnerType } from "../types";

export const MISSING = "—";

type Numeric = number | bigint | null | undefined;

function finite(value: Numeric): number | undefined {
  if (value === null || value === undefined) return undefined;
  const n = Number(value);
  return Number.isFinite(n) ? n : undefined;
}

/** "850 ms", "7.2 s", "7 m 59 s", "2 h 5 m". One rule everywhere. */
export function formatDuration(ms: Numeric): string {
  const value = finite(ms);
  if (value === undefined || value < 0) return MISSING;
  if (value < 1000) return `${Math.round(value)} ms`;
  if (value < 10_000) return `${(value / 1000).toFixed(1)} s`;
  const totalSeconds = Math.round(value / 1000);
  if (totalSeconds < 60) return `${totalSeconds} s`;
  const totalMinutes = Math.floor(totalSeconds / 60);
  if (totalMinutes < 60) return `${totalMinutes} m ${totalSeconds % 60} s`;
  return `${Math.floor(totalMinutes / 60)} h ${totalMinutes % 60} m`;
}

/** "31.0 B", "4.6 M", "850 K", "1,234". Token counters are int64 on the wire. */
export function formatTokens(count: Numeric): string {
  const value = finite(count);
  if (value === undefined) return MISSING;
  const abs = Math.abs(value);
  if (abs >= 1e9) return `${(value / 1e9).toFixed(1)} B`;
  if (abs >= 1e6) return `${(value / 1e6).toFixed(1)} M`;
  if (abs >= 1e4) return `${Math.round(value / 1e3)} K`;
  return value.toLocaleString("en-US");
}

/** A whole count with thousands separators: "1,234". */
export function formatCount(count: Numeric): string {
  const value = finite(count);
  return value === undefined ? MISSING : value.toLocaleString("en-US");
}

/** A 0–1 fraction as a percent: "12.5%". */
export function formatPercent(fraction: Numeric, digits = 1): string {
  const value = finite(fraction);
  return value === undefined ? MISSING : `${(value * 100).toFixed(digits)}%`;
}

/** Quota points (1 pt = 1% of a window): "2.2 pts" under 10, "46 pts" above. */
export function formatPoints(points: Numeric): string {
  const value = finite(points);
  if (value === undefined) return MISSING;
  const text = Math.abs(value) < 10 ? value.toFixed(1) : Math.round(value).toString();
  return `${text} ${text === "1.0" ? "pt" : "pts"}`;
}

/** A machine code read as words: "partial_coverage" → "Partial coverage". */
export function formatCode(code: string | null | undefined): string {
  const words = (code ?? "").replace(/[_-]+/g, " ").trim().toLowerCase();
  return words ? words.charAt(0).toUpperCase() + words.slice(1) : MISSING;
}

const UUID = /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/gi;

/** Text with every UUID cut to its first 8 characters: the short id people recognise, never the raw one. */
export function shortenIds(text: string | null | undefined): string {
  return (text ?? "").replace(UUID, (id) => id.slice(0, 8));
}

/** "1 run", "2 runs". */
export function pluralize(count: number, singular: string, plural = `${singular}s`): string {
  return `${formatCount(count)} ${count === 1 ? singular : plural}`;
}

const RUNNER_LABELS: Record<string, string> = {
  "claude-code": "Claude Code",
  claude: "Claude Code",
  codex: "Codex",
  opencode: "OpenCode",
  grok: "Grok",
};

const RUNNER_TYPE_SLUGS: Partial<Record<RunnerType, string>> = {
  [RunnerType.CLAUDE_CODE]: "claude-code",
  [RunnerType.CODEX]: "codex",
  [RunnerType.OPENCODE]: "opencode",
  [RunnerType.GROK]: "grok",
};

function titleCase(value: string): string {
  return value
    .split(/[-_\s]+/)
    .filter(Boolean)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}

/** Runners are named by tool: "Claude Code", "Codex", "OpenCode". Accepts the enum or a slug. */
export function runnerLabel(runner?: RunnerType | string): string {
  if (runner === undefined || runner === "") return MISSING;
  const slug = typeof runner === "string" ? runner : RUNNER_TYPE_SLUGS[runner];
  if (!slug) return MISSING;
  return RUNNER_LABELS[slug] ?? titleCase(slug);
}

const SUBSCRIPTION_LABELS: Record<string, string> = {
  anthropic: "Claude",
  claude: "Claude",
  "claude-code": "Claude",
  openai: "Codex",
  codex: "Codex",
};

const NAMED_WINDOW_MINUTES: Record<string, number> = { five_hour: 300, seven_day: 10_080 };

/** "weekly", "5-hour", "daily"; an unrecognised window is title-cased as-is. */
export function windowLabel(window: string): string {
  const digits = /^(\d+)m$/.exec(window)?.[1];
  const minutes = digits ? Number(digits) : NAMED_WINDOW_MINUTES[window];
  if (minutes === 10_080) return "weekly";
  if (minutes === 1_440) return "daily";
  if (minutes && minutes % 60 === 0) return `${minutes / 60}-hour`;
  return titleCase(window);
}

/** Subscriptions are named by plan: "Claude · weekly", "Codex · 5-hour". */
export function subscriptionLabel(provider: string, window: string): string {
  return `${SUBSCRIPTION_LABELS[provider] ?? titleCase(provider)} · ${windowLabel(window)}`;
}
