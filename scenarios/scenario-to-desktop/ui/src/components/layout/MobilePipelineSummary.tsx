/**
 * Compact pipeline info bar shown at the top of the generator page on mobile.
 * Replaces the full sidebar — tapping it opens the drawer.
 */

import { Menu, Loader2, CheckCircle2, XCircle, Circle } from "lucide-react";
import { usePipelineStore, selectProgress, selectCurrentStage, selectIsRunning } from "../../store";
import { cn } from "../../lib/utils";
import { formatStageName } from "../../lib/status-display";

interface MobilePipelineSummaryProps {
  onOpenDrawer: () => void;
}

export function MobilePipelineSummary({ onOpenDrawer }: MobilePipelineSummaryProps) {
  const scenarioName = usePipelineStore((s) => s.scenarioName);
  const pipelineStatus = usePipelineStore((s) => s.pipelineStatus);
  const runStatus = usePipelineStore((s) => s.runStatus);
  const progress = usePipelineStore(selectProgress);
  const currentStage = usePipelineStore(selectCurrentStage);
  const isRunning = usePipelineStore(selectIsRunning);

  const status = pipelineStatus?.status ?? runStatus;
  const progressPercent = Math.round(progress * 100);

  const StatusIcon =
    status === "completed" ? CheckCircle2 :
    status === "failed" ? XCircle :
    isRunning ? Loader2 :
    Circle;

  const statusColor =
    status === "completed" ? "text-green-400" :
    status === "failed" ? "text-red-400" :
    isRunning ? "text-blue-400" :
    "text-slate-500";

  return (
    <button
      type="button"
      onClick={onOpenDrawer}
      className="flex w-full items-center gap-3 rounded-lg border border-white/10 bg-slate-900/60 p-3 text-left"
      aria-label="Open pipeline sidebar"
    >
      <Menu className="h-5 w-5 shrink-0 text-slate-400" />

      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <StatusIcon
            className={cn("h-4 w-4 shrink-0", statusColor, isRunning && "animate-spin")}
          />
          <span className="text-sm font-medium text-slate-200 truncate">
            {scenarioName || "No scenario selected"}
          </span>
        </div>

        {/* Progress bar when running */}
        {isRunning && (
          <div className="mt-1.5 flex items-center gap-2">
            <div className="h-1.5 flex-1 rounded-full bg-slate-800 overflow-hidden">
              <div
                className="h-full bg-blue-500 transition-all duration-300 ease-out"
                style={{ width: `${progressPercent}%` }}
              />
            </div>
            <span className="text-xs text-slate-500 shrink-0">
              {currentStage ? formatStageName(currentStage) : `${progressPercent}%`}
            </span>
          </div>
        )}
      </div>
    </button>
  );
}
