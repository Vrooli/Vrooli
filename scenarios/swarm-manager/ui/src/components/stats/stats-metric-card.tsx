/**
 * StatsMetricCard — the canonical stat card for the Stats panel. Renders a
 * label + value with a muted denominator (e.g. "2 of 39 executions"). When
 * sampleSize is below `minSample`, delegates to InsufficientDataCard so the
 * UI never shows a misleading zero or rate.
 */

import { cn } from "../../lib/utils";
import { InsufficientDataCard } from "./insufficient-data-card";

interface StatsMetricCardProps {
  label: string;
  value: string;
  sampleSize: number;
  minSample: number;
  sampleNoun?: string;
  insufficientReason?: string;
  valueClassName?: string;
  subtext?: string;
  testId?: string;
}

export function StatsMetricCard({
  label,
  value,
  sampleSize,
  minSample,
  sampleNoun = "samples",
  insufficientReason,
  valueClassName,
  subtext,
  testId,
}: StatsMetricCardProps) {
  if (sampleSize < Math.max(1, minSample)) {
    return (
      <InsufficientDataCard
        label={label}
        reason={insufficientReason ?? `Need at least ${minSample} ${sampleNoun}.`}
        have={sampleSize}
        required={minSample}
        testId={testId}
      />
    );
  }

  return (
    <div className="rounded-lg border border-slate-700/50 bg-slate-900/40 p-3" data-testid={testId}>
      <p className="text-xs text-slate-400">{label}</p>
      <p className={cn("text-lg font-semibold text-slate-100", valueClassName)}>{value}</p>
      <p className="text-xs text-slate-500">
        {subtext ? `${subtext} · ` : null}
        n = {sampleSize}
      </p>
    </div>
  );
}
