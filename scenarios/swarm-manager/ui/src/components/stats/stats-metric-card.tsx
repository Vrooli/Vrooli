/**
 * StatsMetricCard — the canonical stat card for the Stats panel. Renders a
 * label + value with a muted denominator (e.g. "2 of 39 executions"). When
 * sampleSize is below `minSample`, delegates to InsufficientDataCard so the
 * UI never shows a misleading zero or rate.
 */

import type { LucideIcon } from "lucide-react";
import { InsufficientDataCard } from "./insufficient-data-card";
import { StatsCard } from "./stats-card";

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
  icon?: LucideIcon;
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
  icon,
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
    <StatsCard label={label} value={value} valueClassName={valueClassName} icon={icon} testId={testId}>
      <p className="text-xs text-slate-500">
        {subtext ? `${subtext} · ` : null}
        n = {sampleSize}
      </p>
    </StatsCard>
  );
}
