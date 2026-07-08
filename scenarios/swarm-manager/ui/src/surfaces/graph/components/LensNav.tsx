/**
 * LensNav - Operator navigation between the Plan board and the single Graph
 * surface. Rendered as lightweight icon + short-label tabs so it reads as
 * navigation, while Focus stays graph mode state inside Graph rather than a
 * separate tab.
 */

import { BarChart3, Columns3, Network, type LucideIcon } from "lucide-react";
import { CompactTabBar, type CompactTabItem } from "../../../components/ui/compact-tab-bar";
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
  const items: CompactTabItem<AppGraphLens>[] = LENSES.map((lens) => {
    const badge = badges[lens.id] ?? 0;
    return {
      value: lens.id,
      label: lens.label,
      icon: lens.icon,
      badge: badge > 0 ? (
        <span
          className="ml-0.5 rounded-full bg-slate-700/70 px-1.5 py-0.5 text-[11px] leading-none text-slate-300 data-[active=true]:bg-cyan-500/20 data-[active=true]:text-cyan-100"
          data-active={activeSurface === lens.id}
          data-testid={`lens-${lens.id}-badge`}
        >
          {badge}
        </span>
      ) : null,
    };
  });

  return (
    <CompactTabBar
      items={items}
      activeValue={activeSurface}
      onValueChange={onLensChange}
      aria-label="Workspace view"
      className="gap-1"
      tabTestIdPrefix="lens"
      data-testid="lens-nav"
    />
  );
}
