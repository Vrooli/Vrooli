import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import { cn } from "../../lib/utils";

interface StatsCardProps {
  label: string;
  value: string;
  subtext?: string;
  icon?: LucideIcon;
  children?: ReactNode;
  className?: string;
  valueClassName?: string;
  testId?: string;
}

export function StatsCard({
  label,
  value,
  subtext,
  icon: Icon,
  children,
  className,
  valueClassName,
  testId,
}: StatsCardProps) {
  return (
    <div
      className={cn(
        // Shared min-height keeps StatsCard aligned with InsufficientDataCard
        // when the two sit side-by-side in a grid (e.g. the Dashboard tab).
        "min-h-[72px] rounded-lg border border-slate-700/50 bg-slate-900/40 p-3",
        className,
      )}
      data-testid={testId}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-xs text-slate-400">{label}</p>
          <p className={cn("text-lg font-semibold text-slate-100", valueClassName)}>{value}</p>
          {subtext && <p className="text-xs text-slate-500">{subtext}</p>}
        </div>
        {Icon ? (
          <Icon aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0 text-cyan-400/80" />
        ) : null}
      </div>
      {children}
    </div>
  );
}
