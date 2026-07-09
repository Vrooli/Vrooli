/**
 * @vrooliComponentSource react-component-library:EmptyState
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption template:react-vite:empty-state
 * @vrooliComponentAppliedAt 2026-07-07T00:00:00Z
 * @vrooliComponentSourceSha256 35e4777c7b380571efb456e860d89ef8b3f4a90b8d7ee6e7848322cd135256bf
 * @vrooliComponentDriftHash 35e4777c7b380571efb456e860d89ef8b3f4a90b8d7ee6e7848322cd135256bf
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
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
