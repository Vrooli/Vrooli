/**
 * @vrooliComponentSource react-component-library:WorkspaceHeader
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 1e61c606-a8fa-49b7-befe-d2d02892770c
 * @vrooliComponentAppliedAt 2026-08-06T03:46:45Z
 * @vrooliComponentSourceSha256 0b3403702b5167508e708e9bfa0764273628a4c4e3020c12fc41f304ce051fc4
 * @vrooliComponentDriftHash fd0f37636060b6d03ad57235ed60fe90cdab5bafcf0d7ad7ac13ba1cffa392f2
 * @vrooliComponentTokenTranslation bg-app-surface->bg-app-surface,border-app-border->border-app-border,text-app-foreground->text-app-foreground,text-app-muted-foreground->text-app-muted-foreground
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { ReactNode } from "react";

export interface WorkspaceHeaderProps {
  title: ReactNode;
  description?: ReactNode;
  leading?: ReactNode;
  primaryAction?: ReactNode;
  actions?: ReactNode;
  children?: ReactNode;
  className?: string;
}

/** A structural header: callers own navigation state and action behavior. */
export function WorkspaceHeader({ title, description, leading, primaryAction, actions, children, className }: WorkspaceHeaderProps) {
  return <header data-testid="workspace-header" className={["w-full min-w-0 max-w-full shrink-0 overflow-hidden border-b border-app-border bg-app-surface", className].filter(Boolean).join(" ")}>
    <div className="flex min-h-14 items-center gap-3 px-3 py-2 sm:px-4">
      {leading ? <div className="flex shrink-0 items-center">{leading}</div> : null}
      <div className="min-w-0 flex-1">
        <h1 className="truncate text-base font-semibold text-app-foreground">{title}</h1>
        {description ? <p className="mt-0.5 truncate text-xs text-app-muted-foreground">{description}</p> : null}
      </div>
      {(primaryAction || actions) ? <div className="flex shrink-0 items-center gap-2">{primaryAction}{actions}</div> : null}
    </div>
    {children ? <div className="border-t border-app-border px-3 sm:px-4">{children}</div> : null}
  </header>;
}
