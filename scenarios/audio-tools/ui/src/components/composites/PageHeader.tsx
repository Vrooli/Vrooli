import * as React from "react";
import { cn } from "../../lib/utils";

export interface PageHeaderProps {
  title: React.ReactNode;
  description?: React.ReactNode;
  actions?: React.ReactNode;
  className?: string;
}

export function PageHeader({ title, description, actions, className }: PageHeaderProps) {
  return (
    <div className={cn("mb-4 flex flex-wrap items-end justify-between gap-3 md:mb-6", className)}>
      <div className="min-w-0">
        <h1 className="truncate text-xl font-semibold text-app-foreground md:text-2xl">{title}</h1>
        {description ? (
          <p className="mt-1 text-sm text-app-muted-foreground">{description}</p>
        ) : null}
      </div>
      {actions ? <div className="flex flex-wrap items-center gap-2">{actions}</div> : null}
    </div>
  );
}
