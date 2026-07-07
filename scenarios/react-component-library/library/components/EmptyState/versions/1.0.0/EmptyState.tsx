/**
 * @libraryId react-component-library:EmptyState
 * @version 1.0.0
 * @status released
 * @deps {"react":"^18"}
 */
import type { ReactNode } from "react";

export interface EmptyStateProps {
  title: string;
  description?: string;
  icon?: ReactNode;
  action?: ReactNode;
  className?: string;
}

const joinClasses = (...classes: Array<string | undefined | false>) =>
  classes.filter(Boolean).join(" ");

export function EmptyState({ title, description, icon, action, className }: EmptyStateProps) {
  return (
    <div
      className={joinClasses(
        "flex min-w-0 flex-col items-start gap-3 rounded-panel border border-dashed border-app-border bg-app-surface-muted p-4 text-app-foreground",
        className,
      )}
    >
      {icon && <div className="text-app-muted-foreground">{icon}</div>}
      <div className="min-w-0">
        <h3 className="text-base font-semibold">{title}</h3>
        {description && <p className="mt-1 text-sm text-app-muted-foreground">{description}</p>}
      </div>
      {action && <div className="pt-1">{action}</div>}
    </div>
  );
}

