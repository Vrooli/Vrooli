import * as React from "react";
import { cn } from "../../lib/utils";

export interface EmptyStateProps {
  icon?: React.ReactNode;
  title: React.ReactNode;
  description?: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
  tone?: "neutral" | "error";
}

export function EmptyState({ icon, title, description, action, className, tone = "neutral" }: EmptyStateProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-start gap-2 rounded-panel border border-dashed p-6 text-sm",
        tone === "error"
          ? "border-app-danger/40 bg-app-danger-soft/40 text-app-foreground"
          : "border-app-border bg-app-surface-muted/50 text-app-muted-foreground",
        className,
      )}
      role={tone === "error" ? "alert" : undefined}
    >
      {icon ? <div className="text-app-muted-foreground">{icon}</div> : null}
      <p className="text-sm font-medium text-app-foreground">{title}</p>
      {description ? <p className="text-xs text-app-muted-foreground">{description}</p> : null}
      {action ? <div className="mt-2">{action}</div> : null}
    </div>
  );
}
