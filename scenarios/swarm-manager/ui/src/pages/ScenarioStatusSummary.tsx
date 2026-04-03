/**
 * ScenarioStatusSummary - Quick status filter buttons for scenario health overview.
 *
 * Extracted from ScenariosPage.tsx (Phase 29 iteration 4).
 * Shows running/error/stopped counts as toggle-able filter chips.
 */

import { selectors } from "../consts/selectors";
import type { ScenarioStatus } from "../types";

export interface StatusSummary {
  running: number;
  stopped: number;
  error: number;
}

export interface ScenarioStatusSummaryProps {
  summary: StatusSummary;
  activeFilter: ScenarioStatus | "";
  onFilterToggle: (status: ScenarioStatus | "") => void;
}

export function ScenarioStatusSummary({
  summary,
  activeFilter,
  onFilterToggle,
}: ScenarioStatusSummaryProps) {
  return (
    <div className="flex items-center gap-1" data-testid={selectors.scenarios.statusSummary}>
      {summary.running > 0 && (
        <button
          onClick={() => onFilterToggle(activeFilter === "running" ? "" : "running")}
          className={`flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium transition-colors ${
            activeFilter === "running"
              ? "bg-green-500/20 text-green-300 ring-1 ring-green-500/50"
              : "text-green-400 hover:bg-green-500/10"
          }`}
          data-testid={selectors.scenarios.runningCount}
          title="Click to filter by running status"
        >
          {summary.running} running
        </button>
      )}
      {summary.stopped > 0 && (
        <button
          onClick={() => onFilterToggle(activeFilter === "stopped" ? "" : "stopped")}
          className={`flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium transition-colors ${
            activeFilter === "stopped"
              ? "bg-slate-500/30 text-slate-200 ring-1 ring-slate-400/50"
              : "text-slate-400 hover:bg-slate-500/10"
          }`}
          data-testid={selectors.scenarios.stoppedCount}
          title="Click to filter by stopped status"
        >
          {summary.stopped} stopped
        </button>
      )}
      {summary.error > 0 && (
        <button
          onClick={() => onFilterToggle(activeFilter === "error" ? "" : "error")}
          className={`flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium transition-colors ${
            activeFilter === "error"
              ? "bg-red-500/20 text-red-300 ring-1 ring-red-500/50"
              : "text-red-400 hover:bg-red-500/10"
          }`}
          data-testid={selectors.scenarios.errorCount}
          title="Click to filter by error status"
        >
          {summary.error} error{summary.error !== 1 ? "s" : ""}
        </button>
      )}
    </div>
  );
}
