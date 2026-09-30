import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

import { cn } from "../lib/utils";

/**
 * Calm, neutral empty state: an icon, a short title, an optional explanation,
 * and an optional call to action. Used wherever a list or surface has nothing
 * yet — never leave blank space (DESIGN.md feedback contract). Copy is passed
 * pre-translated by the caller.
 */
export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  className,
  "data-testid": testId,
}: {
  icon?: LucideIcon;
  title: string;
  description?: string;
  action?: ReactNode;
  className?: string;
  "data-testid"?: string;
}) {
  return (
    <div
      data-testid={testId}
      className={cn(
        "flex flex-col items-center gap-3 rounded-panel border border-dashed border-app-border bg-app-surface-muted px-6 py-10 text-center",
        className,
      )}
    >
      {Icon && <Icon aria-hidden="true" className="h-8 w-8 text-app-muted-foreground" />}
      <div className="flex flex-col gap-1">
        <p className="text-sm font-medium text-app-foreground">{title}</p>
        {description && (
          <p className="mx-auto max-w-md text-sm text-app-muted-foreground">{description}</p>
        )}
      </div>
      {action}
    </div>
  );
}
