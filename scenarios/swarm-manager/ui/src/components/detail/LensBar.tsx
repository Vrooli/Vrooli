/**
 * LensBar
 *
 * Shared component for cross-lens navigation from detail pages.
 * Renders a horizontal row of pill buttons that navigate to different
 * graph lenses (topology, flow/history, operations) focused on the
 * current entity.
 */

import { Activity, History, Network, type LucideIcon } from "lucide-react";
import { cn } from "../../lib/utils";
import type { GraphLens } from "../../surfaces/graph/stores/graph-data-store";

export interface LensOption {
  lens: GraphLens;
  label: string;
  icon: LucideIcon;
  iconColorClass: string;
}

interface LensBarProps {
  nodeId: string;
  lenses: LensOption[];
  onDrillToLens: (nodeId: string, lens: GraphLens) => void;
  className?: string;
}

// ── Lens presets per entity type ──────────────────────────────────────

export const BACKLOG_LENSES: LensOption[] = [
  { lens: "topology", label: "View Topology", icon: Network, iconColorClass: "text-indigo-400" },
  { lens: "flow", label: "View History", icon: History, iconColorClass: "text-cyan-400" },
  { lens: "operations", label: "View Operations", icon: Activity, iconColorClass: "text-amber-400" },
];

export const INITIATIVE_LENSES: LensOption[] = [
  { lens: "topology", label: "View Topology", icon: Network, iconColorClass: "text-indigo-400" },
  { lens: "flow", label: "View History", icon: History, iconColorClass: "text-cyan-400" },
  { lens: "operations", label: "View Operations", icon: Activity, iconColorClass: "text-amber-400" },
];

export const EXECUTION_LENSES: LensOption[] = [
  { lens: "flow", label: "View History", icon: History, iconColorClass: "text-cyan-400" },
  { lens: "operations", label: "View Operations", icon: Activity, iconColorClass: "text-amber-400" },
];

export const SCENARIO_LENSES: LensOption[] = [
  { lens: "topology", label: "View Topology", icon: Network, iconColorClass: "text-indigo-400" },
  { lens: "operations", label: "View Operations", icon: Activity, iconColorClass: "text-amber-400" },
];

// ── Component ─────────────────────────────────────────────────────────

export function LensBar({ nodeId, lenses, onDrillToLens, className }: LensBarProps) {
  if (lenses.length === 0) return null;

  return (
    <div
      className={cn("flex items-center gap-2 px-4 py-2 md:px-6", className)}
      data-testid="lens-bar"
    >
      {lenses.map(({ lens, label, icon: Icon, iconColorClass }) => (
        <button
          key={lens}
          type="button"
          onClick={() => onDrillToLens(nodeId, lens)}
          className="flex items-center gap-1.5 rounded-lg bg-slate-700/50 px-3 py-1.5 text-sm font-medium text-slate-100 transition-colors hover:bg-slate-700/70"
          data-testid={`lens-bar-${lens}`}
        >
          <Icon className={cn("h-3.5 w-3.5", iconColorClass)} />
          {label}
        </button>
      ))}
    </div>
  );
}
