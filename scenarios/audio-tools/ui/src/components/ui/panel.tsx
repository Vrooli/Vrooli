import * as React from "react";
import { cn } from "../../lib/utils";

export interface PanelProps extends Omit<React.HTMLAttributes<HTMLElement>, "title"> {
  title: React.ReactNode;
  description?: React.ReactNode;
  actions?: React.ReactNode;
  /** When true, body has no internal padding (caller controls layout). */
  bodyless?: boolean;
}

/**
 * Panel — titled operational section. Header (title + actions) is sticky-friendly,
 * body is a content slot. Use for dense info-density surfaces; do NOT wrap whole
 * pages in a Panel.
 */
export const Panel = React.forwardRef<HTMLElement, PanelProps>(
  ({ className, title, description, actions, bodyless, children, ...props }, ref) => (
    <section
      ref={ref}
      className={cn(
        "rounded-panel border border-app-border bg-app-surface text-app-foreground",
        className,
      )}
      {...props}
    >
      <header className="flex flex-wrap items-center justify-between gap-3 border-b border-app-border px-4 py-3">
        <div className="min-w-0">
          <h2 className="text-sm font-semibold leading-tight text-app-foreground">{title}</h2>
          {description ? (
            <p className="mt-0.5 text-xs text-app-muted-foreground">{description}</p>
          ) : null}
        </div>
        {actions ? <div className="flex shrink-0 items-center gap-2">{actions}</div> : null}
      </header>
      <div className={cn(bodyless ? "" : "p-4")}>{children}</div>
    </section>
  ),
);
Panel.displayName = "Panel";
