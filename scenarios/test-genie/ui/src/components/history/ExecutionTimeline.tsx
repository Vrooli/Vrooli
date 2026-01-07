import { useState } from "react";
import {
  CheckCircle2,
  XCircle,
  ChevronDown,
  ChevronRight,
  Clock,
  RefreshCw,
  Loader2,
} from "lucide-react";
import { Button } from "../ui/button";
import { cn } from "../../lib/utils";
import { formatRelative, formatDuration } from "../../lib/formatters";
import type { SuiteExecutionResult } from "../../lib/api";

interface ExecutionTimelineProps {
  executions: SuiteExecutionResult[];
  isLoading?: boolean;
  onRefresh?: () => void;
  isRefreshing?: boolean;
  onRerun?: (execution: SuiteExecutionResult) => void;
}

export function ExecutionTimeline({
  executions,
  isLoading,
  onRefresh,
  isRefreshing,
  onRerun,
}: ExecutionTimelineProps) {
  const [expandedId, setExpandedId] = useState<string | null>(null);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-blue-400" />
      </div>
    );
  }

  if (executions.length === 0) {
    return (
      <div className="rounded-2xl border border-white/10 bg-white/[0.02] p-6">
        {/* Header with refresh */}
        {onRefresh && (
          <div className="flex justify-end mb-4">
            <button
              onClick={onRefresh}
              disabled={isRefreshing}
              className={cn(
                "flex items-center gap-2 px-3 py-1.5 rounded-lg border border-white/10",
                "hover:bg-white/5 transition-colors text-sm font-medium",
                isRefreshing && "opacity-50 cursor-not-allowed"
              )}
            >
              {isRefreshing ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <RefreshCw className="h-4 w-4" />
              )}
              Refresh
            </button>
          </div>
        )}
        <div className="flex flex-col items-center justify-center py-8 text-slate-400">
          <Clock className="h-12 w-12 mb-4 opacity-50" />
          <p className="text-sm">No previous runs</p>
          <p className="text-xs text-slate-500 mt-1">
            Run tests to start tracking history
          </p>
        </div>
      </div>
    );
  }

  // Group executions by date
  const groupedExecutions = executions.reduce((acc, execution) => {
    const date = new Date(execution.completedAt).toLocaleDateString();
    if (!acc[date]) {
      acc[date] = [];
    }
    acc[date].push(execution);
    return acc;
  }, {} as Record<string, SuiteExecutionResult[]>);

  const toggleExpanded = (executionId: string | undefined) => {
    if (!executionId) return;
    setExpandedId(expandedId === executionId ? null : executionId);
  };

  return (
    <div className="rounded-2xl border border-white/10 bg-white/[0.02] p-6">
      {/* Header with refresh */}
      {onRefresh && (
        <div className="flex justify-end mb-4">
          <button
            onClick={onRefresh}
            disabled={isRefreshing}
            className={cn(
              "flex items-center gap-2 px-3 py-1.5 rounded-lg border border-white/10",
              "hover:bg-white/5 transition-colors text-sm font-medium",
              isRefreshing && "opacity-50 cursor-not-allowed"
            )}
          >
            {isRefreshing ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <RefreshCw className="h-4 w-4" />
            )}
            Refresh
          </button>
        </div>
      )}

      {/* Timeline content */}
      <div className="space-y-6">
        {Object.entries(groupedExecutions).map(([date, dateExecutions]) => (
          <div key={date}>
            {/* Date header */}
            <div className="flex items-center gap-3 mb-4">
              <div className="h-px flex-1 bg-white/10" />
              <span className="text-xs font-medium text-slate-400 px-2">{date}</span>
              <div className="h-px flex-1 bg-white/10" />
            </div>

            {/* Executions for this date */}
            <div className="relative">
              {/* Timeline line */}
              <div className="absolute left-[9px] top-2 bottom-2 w-px bg-white/10" />

              {/* Executions */}
              <div className="space-y-2">
                {dateExecutions.map((execution) => {
                  const isExpanded = expandedId === execution.executionId;
                  const hasPhases = execution.phases && execution.phases.length > 0;

                  return (
                    <div
                      key={execution.executionId || execution.completedAt}
                      className="relative pl-7"
                    >
                      {/* Timeline dot */}
                      <div
                        className={cn(
                          "absolute left-0 top-3 w-[18px] h-[18px] rounded-full flex items-center justify-center",
                          execution.success
                            ? "bg-emerald-500/20 border border-emerald-500/30"
                            : "bg-red-500/20 border border-red-500/30"
                        )}
                      >
                        {execution.success ? (
                          <CheckCircle2 className="h-3 w-3 text-emerald-400" />
                        ) : (
                          <XCircle className="h-3 w-3 text-red-400" />
                        )}
                      </div>

                      {/* Execution row */}
                      <div
                        className={cn(
                          "p-3 rounded-lg border transition-colors",
                          execution.success
                            ? "border-white/10 bg-slate-900/50"
                            : "border-red-500/20 bg-red-500/5",
                          hasPhases && "cursor-pointer hover:bg-slate-800/50"
                        )}
                        onClick={() => hasPhases && toggleExpanded(execution.executionId)}
                      >
                        <div className="flex items-center justify-between gap-2">
                          <div className="flex items-center gap-3 flex-1 min-w-0">
                            {/* Preset/custom */}
                            <span className={cn(
                              "text-sm font-medium",
                              execution.success ? "text-white" : "text-red-300"
                            )}>
                              {execution.preset ? `${execution.preset} preset` : "Custom"}
                            </span>

                            {/* Phase summary */}
                            <span className="text-sm text-slate-400">
                              {execution.phaseSummary.passed}/{execution.phaseSummary.total} passed
                            </span>

                            {/* Duration */}
                            <span className="text-sm text-emerald-400">
                              {formatDuration(execution.phaseSummary.durationSeconds)}
                            </span>

                            {/* Relative time */}
                            <span className="text-xs text-slate-500">
                              {formatRelative(execution.completedAt)}
                            </span>
                          </div>

                          {/* Expand chevron */}
                          {hasPhases && (
                            <div className="flex-shrink-0 text-slate-500">
                              {isExpanded ? (
                                <ChevronDown className="h-4 w-4" />
                              ) : (
                                <ChevronRight className="h-4 w-4" />
                              )}
                            </div>
                          )}
                        </div>

                        {/* Expanded phase details */}
                        {isExpanded && hasPhases && (
                          <div className="mt-3 pt-3 border-t border-white/10 space-y-2">
                            {execution.phases.map((phase) => (
                              <div
                                key={phase.name}
                                className="flex items-center gap-3 text-sm"
                              >
                                {/* Phase status */}
                                {phase.status === "passed" ? (
                                  <CheckCircle2 className="h-4 w-4 text-emerald-400 flex-shrink-0" />
                                ) : (
                                  <XCircle className="h-4 w-4 text-red-400 flex-shrink-0" />
                                )}

                                {/* Phase name */}
                                <span className={cn(
                                  "flex-1 min-w-0",
                                  phase.status === "passed" ? "text-slate-300" : "text-red-300"
                                )}>
                                  {phase.name}
                                </span>

                                {/* Phase duration */}
                                <span className="text-slate-500 text-xs">
                                  {formatDuration(phase.durationSeconds)}
                                </span>

                                {/* Error message (truncated) */}
                                {phase.error && (
                                  <span className="text-red-400 text-xs truncate max-w-[200px]" title={phase.error}>
                                    {phase.error}
                                  </span>
                                )}
                              </div>
                            ))}

                            {/* Rerun button */}
                            {onRerun && (
                              <div className="pt-2 flex justify-end">
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    onRerun(execution);
                                  }}
                                >
                                  <RefreshCw className="mr-2 h-3 w-3" />
                                  Rerun tests
                                </Button>
                              </div>
                            )}
                          </div>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
