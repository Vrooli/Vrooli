/**
 * EdgeLegend - Compact floating legend for edge type visual differentiation.
 *
 * Fixed in bottom-left corner (opposite MiniMap in bottom-right).
 * Shows 4 rows: colored line sample + edge type name.
 * Collapsible via toggle. Only visible in topology lens.
 */

import { useState } from "react";
import { ChevronDown, ChevronUp } from "lucide-react";
import { EDGE_STYLES } from "../lib/edge-styles";
import { cn } from "../../../lib/utils";

interface EdgeLegendProps {
  edgeTypes: string[];
}

export function EdgeLegend({ edgeTypes }: EdgeLegendProps) {
  const [collapsed, setCollapsed] = useState(false);
  const legendEntries = edgeTypes
    .map((type) => [type, EDGE_STYLES[type]] as const)
    .filter((entry): entry is [string, (typeof EDGE_STYLES)[string]] => Boolean(entry[1]));

  if (legendEntries.length === 0) {
    return null;
  }

  return (
    <div
      className={cn(
        "absolute bottom-3 left-3 z-20 rounded-lg border border-slate-700/60 bg-slate-900/90 backdrop-blur-sm shadow-lg",
        "transition-all duration-150",
      )}
      data-testid="edge-legend"
    >
      <button
        type="button"
        onClick={() => setCollapsed((v) => !v)}
        className="flex w-full items-center justify-between gap-2 px-3 py-1.5 text-xs font-medium text-slate-300 hover:text-slate-100"
        data-testid="edge-legend-toggle"
      >
        <span>Edges</span>
        {collapsed ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
      </button>
      {!collapsed && (
        <div className="space-y-1 px-3 pb-2" data-testid="edge-legend-items">
          {legendEntries.map(([type, config]) => (
            <div key={type} className="flex items-center gap-2">
              <svg width="24" height="8" className="shrink-0">
                <line
                  x1="0"
                  y1="4"
                  x2="24"
                  y2="4"
                  stroke={config.stroke}
                  strokeWidth="2"
                  strokeDasharray={config.strokeDasharray === "none" ? undefined : config.strokeDasharray}
                />
              </svg>
              <span className="text-[10px] text-slate-400">{config.label}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
