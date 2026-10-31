import type { ReactNode } from "react";
import { cn } from "../../lib/utils";

export interface EmptyStateProps {
  title: ReactNode;
  description?: ReactNode;
  icon?: ReactNode;
  action?: ReactNode;
  className?: string;
  testId?: string;
}

/**
 * Standard empty-state surface: icon (optional) + title + description + action.
 * Use for any "list with zero items" surface (no goldens, no skills, etc.).
 */
export function EmptyState({ title, description, icon, action, className, testId }: EmptyStateProps) {
  return (
    <div
      data-testid={testId}
      className={cn(
        "flex flex-col items-center justify-center gap-3 rounded-panel border border-dashed border-app-border bg-app-surface-muted/40 px-6 py-10 text-center",
        className,
      )}
    >
      {icon ? <div className="text-app-muted-foreground">{icon}</div> : null}
      <div className="space-y-1">
        <p className="text-sm font-medium text-app-foreground">{title}</p>
        {description ? (
          <p className="text-xs text-app-muted-foreground">{description}</p>
        ) : null}
      </div>
      {action}
    </div>
  );
}
