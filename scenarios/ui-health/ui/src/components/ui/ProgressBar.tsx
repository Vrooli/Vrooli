import * as React from "react";

import { cn } from "../../lib/utils";

export interface ProgressBarProps extends Omit<React.HTMLAttributes<HTMLDivElement>, "role"> {
  value: number;
  max?: number;
  label: string;
  tone?: "default" | "success" | "danger";
  showValue?: boolean;
}

const TONE_CLASS: Record<NonNullable<ProgressBarProps["tone"]>, string> = {
  default: "bg-app-primary",
  success: "bg-app-success",
  danger: "bg-app-danger",
};

export function ProgressBar({
  value,
  max = 100,
  label,
  tone = "default",
  showValue = true,
  className,
  ...props
}: ProgressBarProps) {
  const clamped = Math.max(0, Math.min(max, value));
  const pct = max > 0 ? (clamped / max) * 100 : 0;
  return (
    <div className={cn("flex flex-col gap-1", className)} {...props}>
      <div className="flex items-center justify-between text-xs text-app-muted-foreground">
        <span>{label}</span>
        {showValue ? <span aria-hidden>{Math.round(pct)}%</span> : null}
      </div>
      <div
        role="progressbar"
        aria-label={label}
        aria-valuenow={clamped}
        aria-valuemin={0}
        aria-valuemax={max}
        className="h-2 overflow-hidden rounded-pill bg-app-surface-muted"
      >
        <div
          className={cn("h-full transition-[width]", TONE_CLASS[tone])}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}
