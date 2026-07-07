/**
 * LensNav - Operator navigation between the Plan board and the single Graph
 * surface. Rendered as lightweight icon + short-label tabs so it reads as
 * navigation, while Focus stays graph mode state inside Graph rather than a
 * separate tab.
 */

import { BarChart3, Columns3, Network, type LucideIcon } from "lucide-react";
import { cn } from "../../../lib/utils";
import type { AppGraphLens } from "../../../app/routes/route-paths";
import type { GraphLens } from "../stores/graph-data-store";

interface LensNavProps {
  activeLens: AppGraphLens | GraphLens;
  onLensChange: (lens: AppGraphLens) => void;
  badges?: Partial<Record<AppGraphLens, number>>;
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
  { id: "stats", label: "Stats", icon: BarChart3, shortcut: "3" },
];

export function LensNav({ activeLens, onLensChange, badges = {} }: LensNavProps) {
  const activeSurface: AppGraphLens = activeLens === "plan" ? "plan" : activeLens === "stats" ? "stats" : "graph";

  return (
    <div
      className="flex items-center gap-1"
      role="tablist"
      aria-label="Workspace view"
      data-testid="lens-nav"
    >
      {LENSES.map((lens) => {
        const Icon = lens.icon;
        const isActive = activeSurface === lens.id;
        const badge = badges[lens.id] ?? 0;
        return (
          <button
            key={lens.id}
            type="button"
            role="tab"
            aria-selected={isActive}
            onClick={() => onLensChange(lens.id)}
            title={`${lens.label} (${lens.shortcut})`}
            className={cn(
              "relative flex items-center gap-1.5 px-2.5 py-1.5 text-sm transition-colors",
              "after:absolute after:inset-x-2 after:bottom-0 after:h-0.5 after:rounded-full after:transition-colors",
              lens.primary ? "font-semibold" : "font-medium",
              isActive
                ? "text-cyan-400 after:bg-cyan-400"
                : "text-slate-400 after:bg-transparent hover:text-slate-200",
            )}
            data-testid={`lens-${lens.id}`}
          >
            <Icon className="h-4 w-4 shrink-0" aria-hidden />
            <span className="hidden sm:inline">{lens.label}</span>
            {badge > 0 && (
              <span
                className={cn(
                  "ml-0.5 rounded-full px-1.5 py-0.5 text-[11px] leading-none",
                  isActive
                    ? "bg-cyan-500/20 text-cyan-100"
                    : "bg-slate-700/70 text-slate-300",
                )}
                data-testid={`lens-${lens.id}-badge`}
              >
                {badge}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}
