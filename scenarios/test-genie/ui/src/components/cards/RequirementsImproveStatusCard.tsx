import { useState } from "react";
import { Bot, X, ChevronDown, ChevronUp, Square, Loader2, MessageSquare } from "lucide-react";
import type { RequirementsImproveRecord } from "../../lib/api";
import { cn } from "../../lib/utils";
import { Button } from "../ui/button";
import { selectors } from "../../consts/selectors";

interface RequirementsImproveStatusCardProps {
  improve: RequirementsImproveRecord;
  onStop?: () => void;
  onDismiss?: () => void;
  isStopping?: boolean;
  className?: string;
}

const ACTION_TYPE_LABELS: Record<string, string> = {
  write_tests: "Writing/Fixing Tests",
  update_requirements: "Updating Requirements",
  both: "Improving Tests & Requirements"
};

export function RequirementsImproveStatusCard({
  improve,
  onStop,
  onDismiss,
  isStopping,
  className
}: RequirementsImproveStatusCardProps) {
  const [isOutputExpanded, setIsOutputExpanded] = useState(false);
  const [isMessageExpanded, setIsMessageExpanded] = useState(false);

  const isTerminal =
    improve.status === "completed" ||
    improve.status === "failed" ||
    improve.status === "cancelled";
  const isRunning = improve.status === "pending" || improve.status === "running";

  const statusStyle =
    improve.status === "completed"
      ? "bg-emerald-500/20 text-emerald-300"
      : improve.status === "running"
        ? "bg-cyan-500/20 text-cyan-200"
        : improve.status === "pending"
          ? "bg-amber-400/20 text-amber-200"
          : "bg-red-500/20 text-red-200";

  const statusLabel =
    improve.status === "pending"
      ? "Starting..."
      : improve.status === "running"
        ? ACTION_TYPE_LABELS[improve.actionType] || "Improving..."
        : improve.status === "completed"
          ? "Completed"
          : improve.status === "failed"
            ? "Failed"
            : "Cancelled";

  return (
    <section
      className={cn(
        "rounded-2xl border border-white/10 bg-white/[0.02] p-6",
        className
      )}
      data-testid={selectors.requirements.improveStatus}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div
            className={cn(
              "rounded-lg p-2",
              isRunning
                ? "bg-cyan-500/10"
                : isTerminal && improve.status === "completed"
                  ? "bg-emerald-500/10"
                  : "bg-slate-500/10"
            )}
          >
            {isRunning ? (
              <Loader2 className="h-5 w-5 animate-spin text-cyan-400" />
            ) : (
              <Bot className="h-5 w-5 text-slate-400" />
            )}
          </div>
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-slate-400">
              AI Requirements Agent
            </p>
            <p className="text-sm font-medium">
              Improving {improve.requirements.length} requirement
              {improve.requirements.length !== 1 ? "s" : ""}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <span
            className={cn("rounded-full px-3 py-1 text-xs uppercase", statusStyle)}
          >
            {statusLabel}
          </span>

          {isRunning && onStop && (
            <Button
              variant="outline"
              size="sm"
              onClick={onStop}
              disabled={isStopping}
              className="text-red-400 hover:border-red-400/50 hover:text-red-300"
            >
              {isStopping ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Square className="h-4 w-4" />
              )}
              <span className="ml-1">Stop</span>
            </Button>
          )}

          {isTerminal && onDismiss && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onDismiss}
              className="text-slate-400 hover:text-slate-300"
            >
              <X className="h-4 w-4" />
            </Button>
          )}
        </div>
      </div>

      {/* Requirements being improved */}
      <div className="mt-4 flex flex-wrap gap-2">
        {improve.requirements.map((req) => (
          <span
            key={req.id}
            className={cn(
              "rounded-full border px-3 py-1 text-xs",
              req.liveStatus === "failed"
                ? "border-red-500/30 text-red-300"
                : req.liveStatus === "not_run"
                  ? "border-amber-500/30 text-amber-300"
                  : "border-white/10 text-slate-300"
            )}
            title={req.title}
          >
            {req.id}
          </span>
        ))}
      </div>

      {/* User message context */}
      {improve.message && (
        <div className="mt-4">
          <button
            onClick={() => setIsMessageExpanded(!isMessageExpanded)}
            className="flex items-center gap-2 text-sm text-violet-400 transition-colors hover:text-violet-300"
          >
            <MessageSquare className="h-4 w-4" />
            <span>User Context</span>
            {isMessageExpanded ? (
              <ChevronUp className="h-4 w-4" />
            ) : (
              <ChevronDown className="h-4 w-4" />
            )}
          </button>

          {isMessageExpanded && (
            <div className="mt-2 rounded-lg border border-violet-500/20 bg-violet-500/5 p-3">
              <p className="whitespace-pre-wrap text-sm text-slate-300">{improve.message}</p>
            </div>
          )}
        </div>
      )}

      {/* Error message */}
      {improve.error && (
        <div className="mt-4 rounded-lg border border-red-500/20 bg-red-500/10 p-3">
          <p className="text-sm text-red-300">{improve.error}</p>
        </div>
      )}

      {/* Output viewer (expandable) */}
      {improve.output && (
        <div className="mt-4">
          <button
            onClick={() => setIsOutputExpanded(!isOutputExpanded)}
            className="flex items-center gap-2 text-sm text-slate-400 transition-colors hover:text-slate-300"
          >
            {isOutputExpanded ? (
              <ChevronUp className="h-4 w-4" />
            ) : (
              <ChevronDown className="h-4 w-4" />
            )}
            <span>Agent Output</span>
          </button>

          {isOutputExpanded && (
            <div className="mt-2 max-h-96 overflow-auto rounded-lg border border-white/5 bg-black/30 p-4">
              <pre className="whitespace-pre-wrap font-mono text-xs text-slate-300">
                {improve.output}
              </pre>
            </div>
          )}
        </div>
      )}

      {/* Timing info */}
      {improve.completedAt && (
        <p className="mt-4 text-xs text-slate-500">
          Completed at {new Date(improve.completedAt).toLocaleTimeString()}
        </p>
      )}
    </section>
  );
}
