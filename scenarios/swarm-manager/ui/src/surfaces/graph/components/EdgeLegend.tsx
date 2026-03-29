/**
 * EdgeLegend - Compact floating legend for edge type visual differentiation.
 *
 * Fixed in bottom-left corner (opposite MiniMap in bottom-right).
 * Shows colored line sample with arrowhead + edge type name.
 * Collapsible via toggle.
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
              <svg width="28" height="10" className="shrink-0">
                <defs>
                  <marker
                    id={`legend-arrow-${type}`}
                    viewBox="0 0 6 6"
                    refX="5"
                    refY="3"
                    markerWidth="5"
                    markerHeight="5"
                    orient="auto-start-reverse"
                  >
                    <path d="M 0 0 L 6 3 L 0 6 z" fill={config.stroke} />
                  </marker>
                </defs>
                <line
                  x1="0"
                  y1="5"
                  x2="22"
                  y2="5"
                  stroke={config.stroke}
                  strokeWidth="3"
                  strokeDasharray={config.strokeDasharray === "none" ? undefined : config.strokeDasharray}
                  markerEnd={`url(#legend-arrow-${type})`}
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
