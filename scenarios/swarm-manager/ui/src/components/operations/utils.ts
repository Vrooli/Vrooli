/**
 * Operations Center display helpers.
 *
 * Lane → palette and lane → label maps live here so every component that
 * touches a lane (header bar, activity row, view toggle) stays in sync.
 * Future lane additions land in `OPERATIONS_LANES` first; the maps below
 * use defensive defaults so unknown lanes still render a chip.
 */

import type {
  ActivityRow,
  LaneStatus,
  OperationsLane,
} from "../../types/operations";
import { OPERATIONS_LANES } from "../../types/operations";

/**
 * Tailwind color tokens per lane. The Ops Center sticks to the existing
 * slate palette plus four hue accents (cyan / amber / sky / violet) so
 * the four bars are distinguishable without re-using statuses' palette.
 */
export interface LanePalette {
  /** Solid bar fill (used when the lane is below the warning threshold). */
  bar: string;
  /** Bar fill when at or above the warning threshold (≥80% capacity). */
  barWarning: string;
  /** Empty-bar background. */
  track: string;
  /** Subtle border / accent for the chip text. */
  text: string;
}

const LANE_PALETTES: Record<OperationsLane, LanePalette> = {
  investigate: {
    bar: "bg-cyan-500",
    barWarning: "bg-amber-500",
    track: "bg-cyan-500/10",
    text: "text-cyan-300",
  },
  execute: {
    bar: "bg-emerald-500",
    barWarning: "bg-amber-500",
    track: "bg-emerald-500/10",
    text: "text-emerald-300",
  },
  review: {
    bar: "bg-sky-500",
    barWarning: "bg-amber-500",
    track: "bg-sky-500/10",
    text: "text-sky-300",
  },
  reconcile: {
    bar: "bg-violet-500",
    barWarning: "bg-amber-500",
    track: "bg-violet-500/10",
    text: "text-violet-300",
  },
};

const FALLBACK_LANE_PALETTE: LanePalette = {
  bar: "bg-slate-500",
  barWarning: "bg-amber-500",
  track: "bg-slate-700/40",
  text: "text-slate-300",
};

const LANE_LABELS: Record<OperationsLane, string> = {
  investigate: "Investigate",
  execute: "Execute",
  review: "Review",
  reconcile: "Reconcile",
};

export function isCanonicalLane(lane: string): lane is OperationsLane {
  return (OPERATIONS_LANES as readonly string[]).includes(lane);
}

export function laneLabel(lane: string): string {
  return isCanonicalLane(lane) ? LANE_LABELS[lane] : lane;
}

export function lanePalette(lane: string): LanePalette {
  return isCanonicalLane(lane) ? LANE_PALETTES[lane] : FALLBACK_LANE_PALETTE;
}

/** ≥80% utilization triggers the warning palette on bars. */
export const LANE_WARNING_THRESHOLD = 0.8;

/**
 * Ensures the four canonical lanes are always present in the order
 * Investigate → Execute → Review → Reconcile, even if the backend
 * returned fewer (e.g. an older API or a partial response). Lanes the
 * backend returned but aren't in the canonical set are appended after.
 */
export function orderLanes(lanes: LaneStatus[]): LaneStatus[] {
  const byName = new Map<string, LaneStatus>();
  for (const lane of lanes) byName.set(lane.lane, lane);
  const ordered: LaneStatus[] = [];
  for (const name of OPERATIONS_LANES) {
    ordered.push(
      byName.get(name) ?? { lane: name, active: 0, capacity: 0, queue: 0 },
    );
  }
  for (const lane of lanes) {
    if (!isCanonicalLane(lane.lane)) ordered.push(lane);
  }
  return ordered;
}

/** Format a duration in seconds as "<1m" / "Nm" / "Hh Mm". */
export function formatRuntime(seconds: number | undefined): string {
  if (!seconds || seconds < 0) return "—";
  if (seconds < 60) return "<1m";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  const remMinutes = minutes % 60;
  if (remMinutes === 0) return `${hours}h`;
  return `${hours}h ${remMinutes}m`;
}

/**
 * Display label for an activity. Falls back to OwnerName / RunID so the
 * row never shows a blank label for legacy ledger entries that lack
 * `OwnerTitle`.
 */
export function activityDisplayName(row: ActivityRow): string {
  return row.ownerTitle || row.ownerName || row.runId || row.activityId;
}

/**
 * Short subtitle for the row. Operating-mode runs surface
 * `mode · phase · round`; backlog spawns surface their `purpose`.
 */
export function activitySubtitle(row: ActivityRow): string {
  if (row.mode && row.phase) {
    const round = row.round ? ` · R${row.round}` : "";
    return `${row.mode} · ${row.phase}${round}`;
  }
  return row.purpose;
}

/**
 * Group activities by initiative. Initiative-owned activities cluster
 * under their `initiativeName` (or `ownerName` when ownerType is
 * "initiative"); everything else (item-level work, scenario/capture
 * spawns, sessions) lands in a single "standalone" bucket the
 * by-initiative view renders at the bottom.
 */
export interface InitiativeGroup {
  /** Stable key — initiative name, or "" for the standalone bucket. */
  key: string;
  /** Display label — derived from the first row that supplied it. */
  label: string;
  rows: ActivityRow[];
  /** True when this group is the catch-all standalone bucket. */
  standalone: boolean;
}

export function groupByInitiative(rows: ActivityRow[]): InitiativeGroup[] {
  const groups = new Map<string, InitiativeGroup>();
  const standaloneKey = "";
  for (const row of rows) {
    const initiative =
      row.initiativeName ||
      (row.ownerType === "initiative" ? row.ownerName : "");
    const key = initiative || standaloneKey;
    const existing = groups.get(key);
    if (existing) {
      existing.rows.push(row);
      if (!existing.label && initiative) existing.label = initiative;
    } else {
      groups.set(key, {
        key,
        label: initiative,
        rows: [row],
        standalone: !initiative,
      });
    }
  }
  // Order: initiative groups first (alphabetical by label), standalone last.
  const initiativeGroups = Array.from(groups.values())
    .filter((g) => !g.standalone)
    .sort((a, b) => a.label.localeCompare(b.label));
  const standalone = groups.get(standaloneKey);
  return standalone ? [...initiativeGroups, standalone] : initiativeGroups;
}
