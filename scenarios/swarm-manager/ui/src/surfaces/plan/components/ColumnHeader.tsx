/**
 * ColumnHeader — sticky header for one plan column: name, count chip,
 * optional subtitle line and action slot.
 */

import type { ReactNode } from "react";

export interface ColumnHeaderProps {
  title: string;
  count?: number;
  subtitle?: ReactNode;
  action?: ReactNode;
  testId?: string;
}

export function ColumnHeader({ title, count, subtitle, action, testId }: ColumnHeaderProps) {
  return (
    <div
      className="sticky top-0 z-10 border-b border-slate-800 bg-slate-950/95 px-3 py-2 backdrop-blur"
      data-testid={testId}
    >
      <div className="flex items-center gap-2">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-slate-300">{title}</h2>
        {count !== undefined && (
          <span className="rounded-full bg-slate-800 px-2 py-0.5 text-xs text-slate-400">
            {count}
          </span>
        )}
        {action ? <div className="ml-auto flex items-center gap-1">{action}</div> : null}
      </div>
      {subtitle ? <div className="mt-1 text-xs text-slate-500">{subtitle}</div> : null}
    </div>
  );
}
