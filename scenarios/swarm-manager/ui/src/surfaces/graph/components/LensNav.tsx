/**
 * LensNav - Three-tab lens navigation: Plan (the default kanban board), Focus
 * (the attention-filtered graph neighborhood), and Topology (the full graph
 * projection).
 */

import { cn } from "../../../lib/utils";
import type { AppGraphLens } from "../../../app/routes/route-paths";
import type { GraphLens } from "../stores/graph-data-store";

interface LensNavProps {
  activeLens: GraphLens;
  onLensChange: (lens: AppGraphLens) => void;
}

const LENSES: Array<{
  id: AppGraphLens;
  label: string;
  shortLabel: string;
  shortcut: string;
  primary?: boolean;
}> = [
  { id: "plan", label: "Plan", shortLabel: "Plan", shortcut: "1", primary: true },
  { id: "focus", label: "Focus", shortLabel: "Focus", shortcut: "2" },
  { id: "topology", label: "Topology", shortLabel: "Topo", shortcut: "3" },
];

export function LensNav({ activeLens, onLensChange }: LensNavProps) {
  return (
    <div className="flex w-fit flex-col gap-1" data-testid="lens-nav">
      <div
        className="flex items-center rounded-lg border border-slate-700/80 bg-slate-900/60 p-0.5"
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
                "px-2 md:px-3 py-1 md:py-1.5 text-xs md:text-sm rounded-md transition-colors",
                lens.primary ? "font-semibold" : "font-medium",
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
    </div>
  );
}
