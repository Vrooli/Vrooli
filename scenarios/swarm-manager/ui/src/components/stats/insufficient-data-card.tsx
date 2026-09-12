/**
 * InsufficientDataCard — renders in place of a metric value when the sample
 * size is below the meaningful threshold. Used by StatsMetricCard.
 *
 * Why this exists: the Stats panel previously rendered "<1 min" for
 * empty-sample timing metrics and "0% / 100%" for zero-denominator rate
 * metrics, which looked identical to real values and triggered false alarms.
 */

import { Info } from "lucide-react";
import { cn } from "../../lib/utils";

interface InsufficientDataCardProps {
  label: string;
  reason: string;
  have?: number;
  required?: number;
  testId?: string;
  compact?: boolean;
}

export function InsufficientDataCard({ label, reason, have, required, testId, compact }: InsufficientDataCardProps) {
  return (
    <div
      className={cn(
        "rounded-lg border border-slate-700/50 bg-slate-900/40 p-3",
        compact ? "" : "min-h-[72px]",
      )}
      data-testid={testId}
    >
      <p className="text-xs text-slate-400">{label}</p>
      <div className="mt-1 flex items-start gap-1.5 text-sm text-slate-400">
        <Info className="mt-0.5 h-3.5 w-3.5 shrink-0 text-slate-500" />
        <div>
          <p className="text-slate-300">Not enough data yet</p>
          <p className="text-xs text-slate-500">
            {reason}
            {typeof have === "number" && typeof required === "number" && (
              <span className="ml-1 text-slate-600">
                ({have} of {required} needed)
              </span>
            )}
          </p>
        </div>
      </div>
    </div>
  );
}
