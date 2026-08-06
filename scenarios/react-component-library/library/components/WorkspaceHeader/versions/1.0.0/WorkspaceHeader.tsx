/**
 * @libraryId react-component-library:WorkspaceHeader
 * @version 1.0.0
 * @status released
 * @deps {"react":"^18"}
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
export function WorkspaceHeader({
  title,
  description,
  leading,
  primaryAction,
  actions,
  children,
  className,
}: WorkspaceHeaderProps) {
  return (
    <header
      data-testid="workspace-header"
      className={[
        "w-full min-w-0 max-w-full shrink-0 overflow-hidden border-b border-app-border bg-app-surface",
        className,
      ]
        .filter(Boolean)
        .join(" ")}
    >
      <div className="flex min-h-14 items-center gap-3 px-3 py-2 sm:px-4">
        {leading ? (
          <div className="flex shrink-0 items-center">{leading}</div>
        ) : null}
        <div className="min-w-0 flex-1">
          <h1 className="truncate text-base font-semibold text-app-foreground">
            {title}
          </h1>
          {description ? (
            <p className="mt-0.5 truncate text-xs text-app-muted-foreground">
              {description}
            </p>
          ) : null}
        </div>
        {primaryAction || actions ? (
          <div className="flex shrink-0 items-center gap-2">
            {primaryAction}
            {actions}
          </div>
        ) : null}
      </div>
      {children ? (
        <div className="border-t border-app-border px-3 sm:px-4">
          {children}
        </div>
      ) : null}
    </header>
  );
}
