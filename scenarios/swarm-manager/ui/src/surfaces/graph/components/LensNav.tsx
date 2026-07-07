/**
 * LensNav - Operator navigation between the Plan board and the single Graph
 * surface. Rendered as an icon + short-label segmented control so it reads
 * as one unit and matches the icon language of the workspace header's
 * trailing controls. Focus is graph mode state inside Graph, not a separate
 * tab.
 */

import { Columns3, Network, type LucideIcon } from "lucide-react";
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
  icon: LucideIcon;
  shortcut: string;
  primary?: boolean;
}> = [
  { id: "plan", label: "Plan", icon: Columns3, shortcut: "1", primary: true },
  { id: "graph", label: "Graph", icon: Network, shortcut: "2" },
];

export function LensNav({ activeLens, onLensChange }: LensNavProps) {
  const activeSurface: AppGraphLens = activeLens === "plan" ? "plan" : "graph";

  return (
    <div
      className="flex items-center rounded-lg border border-slate-700/80 bg-slate-900/60 p-0.5"
      role="tablist"
      aria-label="Workspace view"
      data-testid="lens-nav"
    >
      {LENSES.map((lens) => {
        const Icon = lens.icon;
        const isActive = activeSurface === lens.id;
        return (
          <button
            key={lens.id}
            type="button"
            role="tab"
            aria-selected={isActive}
            onClick={() => onLensChange(lens.id)}
            title={`${lens.label} (${lens.shortcut})`}
            className={cn(
              "flex items-center gap-1.5 rounded-md px-2.5 py-1 text-sm transition-colors",
              lens.primary ? "font-semibold" : "font-medium",
              isActive
                ? "bg-slate-700/80 text-cyan-400"
                : "text-slate-400 hover:bg-slate-800/50 hover:text-slate-200",
            )}
            data-testid={`lens-${lens.id}`}
          >
            <Icon className="h-4 w-4 shrink-0" aria-hidden />
            <span className="hidden sm:inline">{lens.label}</span>
          </button>
        );
      })}
    </div>
  );
}
