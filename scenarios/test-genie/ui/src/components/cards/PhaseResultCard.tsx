import type { PhaseExecutionResult } from "../../lib/api";
import { cn } from "../../lib/utils";
import { formatDuration } from "../../lib/formatters";

interface PhaseResultCardProps {
  phase: PhaseExecutionResult;
  className?: string;
}

export function PhaseResultCard({ phase, className }: PhaseResultCardProps) {
  return (
    <div className={cn("rounded-xl border border-white/5 bg-black/20 p-3 text-sm", className)}>
      <div className="flex items-center justify-between">
        <span className="font-medium capitalize">{phase.name}</span>
        <span
          className={cn(
            "rounded-full px-2 py-0.5 text-xs uppercase",
            phase.status === "passed"
              ? "bg-emerald-500/20 text-emerald-300"
              : "bg-red-500/20 text-red-300"
          )}
        >
          {phase.status}
        </span>
      </div>
      <p className="mt-1 text-xs text-slate-400">
        {formatDuration(phase.durationSeconds)} · {phase.observations?.length ?? 0} observations
      </p>
      {phase.error && (
        <p className="mt-1 text-xs text-red-300">
          {phase.error.slice(0, 120)}
          {phase.error.length > 120 ? "…" : ""}
        </p>
      )}
    </div>
  );
}
