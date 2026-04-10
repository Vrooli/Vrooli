/**
 * Breadcrumb - Navigation trail for lens drill-down.
 * Shows: Topology > [Focus Node] > History/Operations
 */

import { ChevronRight, Home } from "lucide-react";
import type { GraphLens } from "../stores/graph-data-store";

interface BreadcrumbProps {
  lens: GraphLens;
  focusNodeLabel: string | null;
  onNavigateHome: () => void;
}

function lensDisplayName(lens: GraphLens): string {
  switch (lens) {
    case "operations":
      return "Operations";
    default:
      return "";
  }
}

export function Breadcrumb({ lens, focusNodeLabel, onNavigateHome }: BreadcrumbProps) {
  if (lens === "focus" || lens === "topology") return null;

  return (
    <nav
      className="flex items-center gap-1 text-xs text-slate-400"
      aria-label="Graph navigation"
      data-testid="graph-breadcrumb"
    >
      <button
        type="button"
        onClick={onNavigateHome}
        className="flex items-center gap-1 rounded px-1.5 py-0.5 text-slate-300 hover:bg-slate-800/60 hover:text-slate-100 transition-colors"
        data-testid="breadcrumb-home"
      >
        <Home className="h-3 w-3" />
        <span>Topology</span>
      </button>
      {focusNodeLabel && (
        <>
          <ChevronRight className="h-3 w-3 text-slate-600" />
          <span className="max-w-[160px] truncate text-slate-300" title={focusNodeLabel}>
            {focusNodeLabel}
          </span>
        </>
      )}
      <ChevronRight className="h-3 w-3 text-slate-600" />
      <span className="text-cyan-400 font-medium">{lensDisplayName(lens)}</span>
    </nav>
  );
}
