import type { ReactNode } from "react";

/**
 * Compact page header: a title, an optional one-line subtitle, and an optional
 * actions cluster. On mobile the actions wrap below the title; on wider
 * viewports they sit to the right. Operational, not hero-scale (DESIGN.md).
 */
export function PageHeader({
  title,
  subtitle,
  actions,
}: {
  title: string;
  subtitle?: string;
  actions?: ReactNode;
}) {
  return (
    <header className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div className="flex flex-col gap-1">
        <h1 className="text-xl font-semibold text-app-foreground">{title}</h1>
        {subtitle && <p className="max-w-2xl text-sm text-app-muted-foreground">{subtitle}</p>}
      </div>
      {actions && <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div>}
    </header>
  );
}
