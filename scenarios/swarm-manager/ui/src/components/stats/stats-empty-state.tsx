import type { ReactNode } from "react";
import { cn } from "../../lib/utils";

interface StatsEmptyStateProps {
  children: ReactNode;
  className?: string;
  testId?: string;
}

/**
 * Consistent inline empty-state for stats sections that have no data yet.
 * Replaces the ad-hoc `<p className="text-sm text-slate-500">…</p>` strings
 * that were scattered across the stats tabs so every "nothing here yet"
 * message reads the same.
 */
export function StatsEmptyState({ children, className, testId }: StatsEmptyStateProps) {
  return (
    <p
      className={cn(
        "rounded-lg border border-dashed border-slate-700/50 bg-slate-900/30 px-3 py-4 text-center text-sm text-slate-500",
        className,
      )}
      data-testid={testId}
    >
      {children}
    </p>
  );
}
