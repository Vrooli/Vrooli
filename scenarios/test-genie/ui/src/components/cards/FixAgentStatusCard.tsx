import { useState } from "react";
import { Bot, X, ChevronDown, ChevronUp, Square, Loader2 } from "lucide-react";
import type { FixRecord } from "../../lib/api";
import { cn } from "../../lib/utils";
import { Button } from "../ui/button";
import { selectors } from "../../consts/selectors";

interface FixAgentStatusCardProps {
  fix: FixRecord;
  onStop?: () => void;
  onDismiss?: () => void;
  isStopping?: boolean;
  className?: string;
}

export function FixAgentStatusCard({
  fix,
  onStop,
  onDismiss,
  isStopping,
  className
}: FixAgentStatusCardProps) {
  const [isOutputExpanded, setIsOutputExpanded] = useState(false);

  const isTerminal = fix.status === "completed" || fix.status === "failed" || fix.status === "cancelled";
  const isRunning = fix.status === "pending" || fix.status === "running";

  const statusStyle =
    fix.status === "completed"
      ? "bg-emerald-500/20 text-emerald-300"
      : fix.status === "running"
        ? "bg-cyan-500/20 text-cyan-200"
        : fix.status === "pending"
          ? "bg-amber-400/20 text-amber-200"
          : "bg-red-500/20 text-red-200";

  const statusLabel =
    fix.status === "pending"
      ? "Starting..."
      : fix.status === "running"
        ? "Fixing tests..."
        : fix.status === "completed"
          ? "Completed"
          : fix.status === "failed"
            ? "Failed"
            : "Cancelled";

  return (
    <section
      className={cn(
        "rounded-2xl border border-white/10 bg-white/[0.02] p-6",
        className
      )}
      data-testid={selectors.runs.fixAgentStatus}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className={cn(
            "p-2 rounded-lg",
            isRunning ? "bg-cyan-500/10" : isTerminal && fix.status === "completed" ? "bg-emerald-500/10" : "bg-slate-500/10"
          )}>
            {isRunning ? (
              <Loader2 className="h-5 w-5 text-cyan-400 animate-spin" />
            ) : (
              <Bot className="h-5 w-5 text-slate-400" />
            )}
          </div>
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-slate-400">AI Fix Agent</p>
            <p className="text-sm font-medium">
              Fixing {fix.phases.length} phase{fix.phases.length !== 1 ? "s" : ""}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <span className={cn("rounded-full px-3 py-1 text-xs uppercase", statusStyle)}>
            {statusLabel}
          </span>

          {isRunning && onStop && (
            <Button
              variant="outline"
              size="sm"
              onClick={onStop}
              disabled={isStopping}
              className="text-red-400 hover:text-red-300 hover:border-red-400/50"
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

      {/* Phases being fixed */}
      <div className="mt-4 flex flex-wrap gap-2">
        {fix.phases.map((phase) => (
          <span
            key={phase.name}
            className="rounded-full border border-white/10 px-3 py-1 text-xs capitalize text-slate-300"
          >
            {phase.name}
          </span>
        ))}
      </div>

      {/* Error message */}
      {fix.error && (
        <div className="mt-4 rounded-lg bg-red-500/10 border border-red-500/20 p-3">
          <p className="text-sm text-red-300">{fix.error}</p>
        </div>
      )}

      {/* Output viewer (expandable) */}
      {fix.output && (
        <div className="mt-4">
          <button
            onClick={() => setIsOutputExpanded(!isOutputExpanded)}
            className="flex items-center gap-2 text-sm text-slate-400 hover:text-slate-300 transition-colors"
          >
            {isOutputExpanded ? (
              <ChevronUp className="h-4 w-4" />
            ) : (
              <ChevronDown className="h-4 w-4" />
            )}
            <span>Agent Output</span>
          </button>

          {isOutputExpanded && (
            <div className="mt-2 rounded-lg bg-black/30 border border-white/5 p-4 max-h-96 overflow-auto">
              <pre className="text-xs text-slate-300 whitespace-pre-wrap font-mono">
                {fix.output}
              </pre>
            </div>
          )}
        </div>
      )}

      {/* Timing info */}
      {fix.completedAt && (
        <p className="mt-4 text-xs text-slate-500">
          Completed at {new Date(fix.completedAt).toLocaleTimeString()}
        </p>
      )}
    </section>
  );
}
