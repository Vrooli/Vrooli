/**
 * Status Legend Component
 *
 * Displays a collapsible legend explaining status colors and their meanings.
 * Helps new users understand the visual coding system at a glance.
 *
 * Experience Architecture (Phase 29 Iteration 5):
 * - Addresses F1-DISCOVERABILITY: status meaning wasn't immediately obvious
 * - Reduces cognitive load by explaining the visual language
 */

import { useState } from "react";
import { HelpCircle, ChevronDown, ChevronUp } from "lucide-react";
import type { StatusItem } from "./status-legend.types";

interface StatusLegendProps {
  items: StatusItem[];
  title?: string;
  "data-testid"?: string;
}

export function StatusLegend({
  items,
  title = "Status Guide",
  "data-testid": testId,
}: StatusLegendProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  return (
    <div className="relative" data-testid={testId}>
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="flex items-center gap-1.5 text-xs text-slate-300 hover:text-slate-200 transition-colors"
        aria-expanded={isExpanded}
        aria-label={`${isExpanded ? "Hide" : "Show"} status legend`}
      >
        <HelpCircle className="h-3.5 w-3.5" />
        <span>{title}</span>
        {isExpanded ? (
          <ChevronUp className="h-3 w-3" />
        ) : (
          <ChevronDown className="h-3 w-3" />
        )}
      </button>

      {isExpanded && (
        <div className="absolute top-full left-0 z-20 mt-2 w-64 rounded-lg border border-white/10 bg-slate-800 p-3 shadow-xl">
          <div className="space-y-2">
            {items.map((item) => (
              <div key={item.status} className="flex items-start gap-2">
                <span
                  className={`mt-1 inline-block h-2 w-2 rounded-full flex-shrink-0 ${item.colorClass}`}
                />
                <div className="flex-1 min-w-0">
                  <span className="text-xs font-medium text-slate-200">{item.label}</span>
                  <p className="text-xs text-slate-400">{item.description}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
