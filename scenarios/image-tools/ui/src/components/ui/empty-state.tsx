import type { ReactNode } from "react";
import type { LucideIcon } from "lucide-react";

import { cn } from "../../lib/utils";

export interface EmptyStateProps {
  /** Line-art icon for the state (lucide). */
  Icon: LucideIcon;
  title: string;
  description?: string;
  /** Primary action(s) — typically a button or a "try a sample" control. */
  action?: ReactNode;
  className?: string;
  testId?: string;
}

/**
 * Crafted empty state — a calm line-art icon, a one-line title, an optional
 * description, and a primary action. Used on every surface so a screen is never
 * a blank void (Home recent rail, Library, Activity, the Workspace canvas).
 * Tokens only; the icon picks up the Lume gold accent.
 */
export function EmptyState({ Icon, title, description, action, className, testId }: EmptyStateProps) {
  return (
    <div
      data-testid={testId}
      className={cn(
        "flex flex-col items-center justify-center gap-3 rounded-panel border border-dashed border-app-border bg-app-surface-muted px-6 py-10 text-center",
        className,
      )}
    >
      <span className="flex h-12 w-12 items-center justify-center rounded-full bg-app-surface text-app-brand">
        <Icon aria-hidden="true" className="h-6 w-6" />
      </span>
      <div className="flex flex-col gap-1">
        <p className="text-sm font-medium text-app-foreground">{title}</p>
        {description ? <p className="text-xs text-app-muted-foreground">{description}</p> : null}
      </div>
      {action ? <div className="mt-1 flex flex-wrap items-center justify-center gap-2">{action}</div> : null}
    </div>
  );
}
