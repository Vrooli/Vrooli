/**
 * Dependency Indicator
 *
 * Shows whether a backlog item's dependencies are met or blocking.
 * Distinguishes between workshop-blocking (deps still being planned)
 * and execution-blocking (deps not yet completed).
 */

import { memo } from "react";
import { useBacklogItemLookup } from "./backlog-items-lookup";

const WORKSHOP_BLOCKING_STATUSES = new Set(["backlog", "researching"]);

interface DependencyIndicatorProps {
  dependsOn?: string[];
}

export const DependencyIndicator = memo(function DependencyIndicator({ dependsOn }: DependencyIndicatorProps) {
  const itemsByKey = useBacklogItemLookup();
  if (!dependsOn || dependsOn.length === 0) return null;

  const workshopBlocked = dependsOn.filter((dep) => {
    const item = itemsByKey.get(dep);
    return item && WORKSHOP_BLOCKING_STATUSES.has(item.status);
  });

  const executionUnmet = dependsOn.filter((dep) => {
    const item = itemsByKey.get(dep);
    return !item || item.status !== "completed";
  });

  const allMet = executionUnmet.length === 0;

  // Workshop-blocking takes priority in display (more actionable).
  if (workshopBlocked.length > 0) {
    return (
      <span
        className="inline-flex items-center gap-1 rounded-full bg-amber-500/15 px-2 py-0.5 text-[10px] font-medium text-amber-400"
        title={`Planning blocked by: ${workshopBlocked.join(", ")}`}
      >
        <svg className="h-2.5 w-2.5" viewBox="0 0 10 10" fill="none">
          <circle cx="5" cy="5" r="4" stroke="currentColor" strokeWidth="1.2" />
          <path d="M5 3V5.5" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
          <circle cx="5" cy="7.2" r="0.6" fill="currentColor" />
        </svg>
        {workshopBlocked.length} planning
      </span>
    );
  }

  if (!allMet) {
    return (
      <span
        className="inline-flex items-center gap-1 rounded-full bg-orange-500/15 px-2 py-0.5 text-[10px] font-medium text-orange-400"
        title={`Blocked by: ${executionUnmet.join(", ")}`}
      >
        <svg className="h-2.5 w-2.5" viewBox="0 0 10 10" fill="none">
          <rect x="3" y="1" width="4" height="6" rx="1" stroke="currentColor" strokeWidth="1.2" />
          <path d="M2.5 4H7.5V8.5H2.5V4Z" stroke="currentColor" strokeWidth="1.2" strokeLinejoin="round" />
        </svg>
        {executionUnmet.length} blocked
      </span>
    );
  }

  return (
    <span
      className="inline-flex items-center gap-1 rounded-full bg-emerald-500/15 px-2 py-0.5 text-[10px] font-medium text-emerald-400"
      title={`All ${dependsOn.length} dep${dependsOn.length !== 1 ? "s" : ""} met`}
    >
      <svg className="h-2.5 w-2.5" viewBox="0 0 10 10" fill="none">
        <path d="M2 5.5L4 7.5L8 3" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
      {dependsOn.length} dep{dependsOn.length !== 1 ? "s" : ""}
    </span>
  );
});
