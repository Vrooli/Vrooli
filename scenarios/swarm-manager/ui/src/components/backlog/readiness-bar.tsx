/**
 * Compact readiness indicator bar for backlog list views.
 *
 * Renders 5 labeled circles (P, S, A, T, R) colored by readiness score
 * with a round badge. Score-0 uses a ring outline instead of a fill
 * to distinguish "scored zero" from "no data".
 *
 * Returns null when no workshop rounds have been completed.
 */
import { memo } from "react";
import { cn } from "../../lib";
import {
  READINESS_DIMENSIONS,
  DIMENSION_LABELS,
  DIMENSION_SHORT_LABELS,
} from "../../lib/maturity";
import type { ReadinessIndicatorData } from "../../lib/maturity";

interface ReadinessBarProps {
  data: ReadinessIndicatorData;
  className?: string;
}

const FILLED_CLASSES: Record<number, string> = {
  1: "bg-rose-500 text-white",
  2: "bg-amber-500 text-white",
  3: "bg-emerald-500 text-white",
};

const ZERO_CLASS = "ring-1 ring-slate-500 text-slate-500";

export const ReadinessBar = memo(function ReadinessBar({ data, className }: ReadinessBarProps) {
  if (data.roundsCompleted === 0) return null;

  return (
    <div className={cn("flex items-center gap-1.5", className)}>
      <div className="flex gap-0.5">
        {READINESS_DIMENSIONS.map((dim) => {
          const score = data.effectiveScores[dim] ?? 0;
          return (
            <div
              key={dim}
              title={`${DIMENSION_LABELS[dim]}: ${score}/3`}
              className={cn(
                "flex h-3.5 w-3.5 items-center justify-center rounded-full text-[7px] font-bold leading-none",
                score === 0 ? ZERO_CLASS : FILLED_CLASSES[score],
              )}
            >
              {DIMENSION_SHORT_LABELS[dim]}
            </div>
          );
        })}
      </div>
      <span className="text-[10px] font-medium text-slate-400">
        R{data.roundsCompleted}
      </span>
      {data.ready && (
        <span className="text-[10px] font-medium text-emerald-400">Ready</span>
      )}
    </div>
  );
});
