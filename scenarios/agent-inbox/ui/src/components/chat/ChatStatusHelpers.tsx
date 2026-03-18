/**
 * Helper components and constants for ChatStatusIcon.
 *
 * Extracted from ChatStatusIcon.tsx for modularity.
 */
import { Zap } from "lucide-react";
import type { AgentRunStatus } from "../../lib/api";
import type { AgentMetric } from "./agent/AgentEventList";

/* ── Circular-progress constants ─────────────────────────────────── */
export const RING_SIZE = 28;
export const STROKE_WIDTH = 2.5;
export const RADIUS = (RING_SIZE - STROKE_WIDTH) / 2;
export const CIRCUMFERENCE = 2 * Math.PI * RADIUS;

export const RING_COLORS: Record<string, string> = {
  pending: "stroke-blue-400",
  starting: "stroke-blue-400",
  running: "stroke-blue-500",
  needs_review: "stroke-yellow-500",
  complete: "stroke-green-500",
  failed: "stroke-red-500",
  cancelled: "stroke-slate-500",
};

/* ── Helpers ──────────────────────────────────────────────────────── */

export function statusTextColor(status: AgentRunStatus): string {
  switch (status) {
    case "running": case "starting": case "pending": return "text-blue-400";
    case "complete": return "text-green-400";
    case "failed": case "cancelled": return "text-red-400";
    case "needs_review": return "text-yellow-400";
    default: return "text-slate-400";
  }
}

export function statusLabel(status: AgentRunStatus): string {
  switch (status) {
    case "pending": return "Pending";
    case "starting": return "Starting";
    case "running": return "Running";
    case "needs_review": return "Needs Review";
    case "complete": return "Completed";
    case "failed": return "Failed";
    case "cancelled": return "Stopped";
    default: return status;
  }
}

export function formatValue(value: number, unit: string): string {
  if (unit === "tokens" || unit === "bytes") {
    if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
    if (value >= 1_000) return `${(value / 1_000).toFixed(1)}k`;
  }
  if (unit === "ms") {
    if (value >= 1_000) return `${(value / 1_000).toFixed(1)}s`;
  }
  if (unit === "usd" || unit === "USD" || unit === "$") {
    return `$${value.toFixed(4)}`;
  }
  return value % 1 === 0 ? String(value) : value.toFixed(2);
}

/* ── Metric chips (mirrors AgentStatusIndicator logic) ────────── */

export function MetricsDisplay({ metrics }: { metrics: AgentMetric[] }) {
  const totals = new Map<string, { value: number; unit: string }>();
  for (const m of metrics) {
    const existing = totals.get(m.name);
    if (existing) existing.value += m.value;
    else totals.set(m.name, { value: m.value, unit: m.unit });
  }

  if (totals.size === 0) return null;

  const chips: { key: string; label: string; tooltip: string }[] = [];
  for (const [name, { value, unit }] of totals) {
    chips.push({
      key: name,
      label: `${formatValue(value, unit)} ${unit || name}`,
      tooltip: `${name}: ${value} ${unit}`,
    });
  }

  return (
    <div className="flex items-center gap-1.5 flex-wrap">
      <Zap className="h-3 w-3 text-zinc-500 flex-shrink-0" />
      {chips.map((c) => (
        <span key={c.key} className="px-1.5 py-0.5 text-xs rounded bg-zinc-700/50 text-zinc-400" title={c.tooltip}>
          {c.label}
        </span>
      ))}
    </div>
  );
}
