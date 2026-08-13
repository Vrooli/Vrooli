/**
 * @vrooliComponentSource react-component-library:EmptyState
 * @vrooliComponentVersion 1.1.0
 * @vrooliComponentAdoption ec7c47cb-92bc-4ff1-a45b-97dfa2531994
 * @vrooliComponentAppliedAt 2026-07-15T03:15:14Z
 * @vrooliComponentSourceSha256 06dbe7eb1804132d8f57871747deb1109d730373fc32383980deeecdeb2990b3
 * @vrooliComponentDriftHash 06dbe7eb1804132d8f57871747deb1109d730373fc32383980deeecdeb2990b3
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import type { ReactNode } from "react";

export interface EmptyStateProps {
  title: string;
  description?: string;
  icon?: ReactNode;
  action?: ReactNode;
  className?: string;
}

const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

export function EmptyState({ title, description, icon, action, className }: EmptyStateProps) {
  return (
    <div
      className={cn(
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

