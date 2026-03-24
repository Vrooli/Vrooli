/**
 * Dependency Indicator
 *
 * Shows whether a backlog item's dependencies are met or blocking.
 */

import type { BacklogItem } from "../../types";

interface DependencyIndicatorProps {
  dependsOn?: string[];
  allItems: BacklogItem[];
}

export function DependencyIndicator({ dependsOn, allItems }: DependencyIndicatorProps) {
  if (!dependsOn || dependsOn.length === 0) return null;

  const completedSet = new Set(
    allItems
      .filter((item) => item.status === "completed")
      .map((item) => `${item.kind}/${item.name}`),
  );

  const unmet = dependsOn.filter((dep) => !completedSet.has(dep));
  const allMet = unmet.length === 0;

  const tooltip = allMet
    ? `All ${dependsOn.length} dep${dependsOn.length !== 1 ? "s" : ""} met`
    : `Blocked by: ${unmet.join(", ")}`;

  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${
        allMet ? "bg-emerald-500/15 text-emerald-400" : "bg-orange-500/15 text-orange-400"
      }`}
      title={tooltip}
    >
      {allMet ? (
        <>
          <svg className="h-2.5 w-2.5" viewBox="0 0 10 10" fill="none">
            <path d="M2 5.5L4 7.5L8 3" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
          {dependsOn.length} dep{dependsOn.length !== 1 ? "s" : ""}
        </>
      ) : (
        <>
          <svg className="h-2.5 w-2.5" viewBox="0 0 10 10" fill="none">
            <rect x="3" y="1" width="4" height="6" rx="1" stroke="currentColor" strokeWidth="1.2" />
            <path d="M2.5 4H7.5V8.5H2.5V4Z" stroke="currentColor" strokeWidth="1.2" strokeLinejoin="round" />
          </svg>
          {unmet.length} blocked
        </>
      )}
    </span>
  );
}
