import * as React from "react";
import { cn } from "../lib/utils";
import { selectors } from "../consts/selectors";

export interface EmptyStateProps {
  /** Pre-translated title. Use `t(strings.…)` at the call site. */
  title: string;
  /** Pre-translated description. Optional. */
  description?: string;
  /** Optional action slot (e.g., a primary button). */
  action?: React.ReactNode;
  /** Optional icon slot (e.g., a lucide icon). */
  icon?: React.ReactNode;
  className?: string;
}

export function EmptyState({ title, description, action, icon, className }: EmptyStateProps) {
  return (
    <div
      data-testid={selectors.shared.emptyState.root}
      role="status"
      className={cn(
        "flex flex-col items-center justify-center gap-2 rounded-panel border border-dashed border-app-border bg-app-surface p-8 text-center text-app-muted-foreground backdrop-blur-sm",
        className,
      )}
    >
      {icon ? <div aria-hidden="true">{icon}</div> : null}
      <p data-testid={selectors.shared.emptyState.title} className="text-sm font-semibold text-app-foreground">
        {title}
      </p>
      {description ? (
        <p data-testid={selectors.shared.emptyState.description} className="max-w-sm text-sm">
          {description}
        </p>
      ) : null}
      {action ? <div data-testid={selectors.shared.emptyState.action}>{action}</div> : null}
    </div>
  );
}
