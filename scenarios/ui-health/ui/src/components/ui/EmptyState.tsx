import { type LucideIcon } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";

export interface EmptyStateProps extends React.HTMLAttributes<HTMLDivElement> {
  icon?: LucideIcon;
  title: string;
  description?: React.ReactNode;
  action?: React.ReactNode;
}

export function EmptyState({ icon: Icon, title, description, action, className, ...props }: EmptyStateProps) {
  return (
    <div
      role="status"
      className={cn(
        "flex flex-col items-center justify-center gap-3 rounded-panel border border-dashed border-app-border bg-app-surface p-10 text-center",
        className,
      )}
      {...props}
    >
      {Icon ? (
        <div className="rounded-pill bg-app-surface-muted p-3 text-app-muted-foreground">
          <Icon aria-hidden className="h-6 w-6" />
        </div>
      ) : null}
      <h3 className="text-base font-semibold text-app-foreground">{title}</h3>
      {description ? (
        <p className="max-w-md text-sm text-app-muted-foreground">{description}</p>
      ) : null}
      {action ? <div className="pt-2">{action}</div> : null}
    </div>
  );
}
