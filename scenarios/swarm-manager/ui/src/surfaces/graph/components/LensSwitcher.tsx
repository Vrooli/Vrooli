/**
 * LensSwitcher - Segmented control for switching between graph lenses.
 */

import { cn } from "../../../lib/utils";
import type { GraphLens } from "../stores/graph-data-store";

interface LensSwitcherProps {
  activeLens: GraphLens;
  onLensChange: (lens: GraphLens) => void;
}

const LENSES: Array<{ id: GraphLens; label: string; shortLabel: string; shortcut: string }> = [
  { id: "topology", label: "Topology", shortLabel: "Topo", shortcut: "1" },
  { id: "flow", label: "Flow", shortLabel: "Flow", shortcut: "2" },
  { id: "operations", label: "Operations", shortLabel: "Ops", shortcut: "3" },
];

export function LensSwitcher({ activeLens, onLensChange }: LensSwitcherProps) {
  return (
    <div
      className="flex items-center rounded-lg border border-slate-700/80 bg-slate-900/60 p-0.5"
      data-testid="lens-switcher"
      role="tablist"
      aria-label="Graph lens"
    >
      {LENSES.map((lens) => (
        <button
          key={lens.id}
          type="button"
          role="tab"
          aria-selected={activeLens === lens.id}
          onClick={() => onLensChange(lens.id)}
          className={cn(
            "px-2 md:px-3 py-1 md:py-1.5 text-xs md:text-sm font-medium rounded-md transition-colors",
            activeLens === lens.id
              ? "bg-slate-700/80 text-cyan-400"
              : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/50",
          )}
          data-testid={`lens-${lens.id}`}
        >
          <span className="md:hidden">{lens.shortLabel}</span>
          <span className="hidden md:inline">{lens.label}</span>
          <span className="hidden lg:inline text-xs ml-1 text-slate-500">
            ({lens.shortcut})
          </span>
        </button>
      ))}
    </div>
  );
}
