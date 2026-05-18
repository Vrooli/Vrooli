import type { ReactNode } from "react";
import { cn } from "../../lib/utils";

export interface PanelHeaderProps {
  title: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
  badge?: ReactNode;
  className?: string;
}

/**
 * Standard panel header: title + optional description on the left,
 * optional badge/status chip beside the title, optional actions on the right.
 */
export function PanelHeader({ title, description, actions, badge, className }: PanelHeaderProps) {
  return (
    <header className={cn("flex items-start justify-between gap-3", className)}>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <h2 className="truncate text-lg font-semibold text-app-foreground">{title}</h2>
          {badge}
        </div>
        {description ? (
          <p className="mt-1 text-xs text-app-muted-foreground">{description}</p>
        ) : null}
      </div>
      {actions ? <div className="flex shrink-0 items-center gap-2">{actions}</div> : null}
    </header>
  );
}
