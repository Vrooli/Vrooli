import type { ReactNode } from "react";
import { cn } from "../../lib/utils";

export interface MetricStatProps {
  label: ReactNode;
  value: ReactNode;
  delta?: ReactNode;
  className?: string;
}

/**
 * Compact label/value pair with optional delta line — used for run summary
 * displays (duration, tokens, cost) and dashboard tiles.
 */
export function MetricStat({ label, value, delta, className }: MetricStatProps) {
  return (
    <div className={cn("flex flex-col gap-0.5", className)}>
      <span className="text-[10px] uppercase tracking-wide text-app-muted-foreground">{label}</span>
      <span className="text-lg font-semibold tabular-nums text-app-foreground">{value}</span>
      {delta ? <span className="text-xs text-app-muted-foreground">{delta}</span> : null}
    </div>
  );
}
