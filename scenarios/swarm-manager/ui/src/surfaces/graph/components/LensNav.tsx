/**
 * LensNav - Hierarchical lens navigation with Topology as the primary "atlas" tab.
 * Flow and Operations are contextual sub-views that may require a focus node.
 */

import { cn } from "../../../lib/utils";
import { Breadcrumb } from "./Breadcrumb";
import type { GraphLens } from "../stores/graph-data-store";

interface LensNavProps {
  activeLens: GraphLens;
  focusNodeId: string | null;
  focusNodeLabel: string | null;
  onLensChange: (lens: GraphLens) => void;
  onReturnToAtlas: () => void;
}

const LENSES: Array<{
  id: GraphLens;
  label: string;
  shortLabel: string;
  shortcut: string;
  primary?: boolean;
}> = [
  { id: "topology", label: "Topology", shortLabel: "Topo", shortcut: "1", primary: true },
  { id: "flow", label: "History", shortLabel: "Hist", shortcut: "2" },
  { id: "operations", label: "Operations", shortLabel: "Ops", shortcut: "3" },
];

export function LensNav({
  activeLens,
  focusNodeId,
  focusNodeLabel,
  onLensChange,
  onReturnToAtlas,
}: LensNavProps) {
  return (
    <div className="flex w-fit flex-col gap-1" data-testid="lens-nav">
      <div
        className="flex items-center rounded-lg border border-slate-700/80 bg-slate-900/60 p-0.5"
        role="tablist"
        aria-label="Graph lens"
      >
        {LENSES.map((lens) => {
          const isFlowDisabled = lens.id === "flow" && !focusNodeId;
          return (
            <button
              key={lens.id}
              type="button"
              role="tab"
              aria-selected={activeLens === lens.id}
              onClick={() => !isFlowDisabled && onLensChange(lens.id)}
              disabled={isFlowDisabled}
              title={isFlowDisabled ? "Select a node in Topology first" : undefined}
              className={cn(
                "px-2 md:px-3 py-1 md:py-1.5 text-xs md:text-sm rounded-md transition-colors",
                lens.primary ? "font-semibold" : "font-medium",
                activeLens === lens.id
                  ? "bg-slate-700/80 text-cyan-400"
                  : isFlowDisabled
                    ? "text-slate-600 cursor-not-allowed"
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
          );
        })}
      </div>
      <Breadcrumb
        lens={activeLens}
        focusNodeLabel={focusNodeLabel}
        onNavigateHome={onReturnToAtlas}
      />
    </div>
  );
}
