import { type ReactNode } from "react";

import { cn } from "../lib/utils";

/**
 * Small set of token-driven layout primitives shared by every console page so
 * spacing, borders, and surface treatment stay coherent with `vrooli-default`.
 * These are presentational only — no domain logic — and intentionally thin.
 */

/** A bordered, raised content card. The console's default content container. */
export function Card({
  className,
  children,
  ...rest
}: { className?: string; children: ReactNode } & React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "rounded-panel border border-app-border bg-app-surface p-4 shadow-sm",
        className,
      )}
      {...rest}
    >
      {children}
    </div>
  );
}

/** A labelled section within a page, with an accessible heading. */
export function SectionPanel({
  title,
  description,
  actions,
  headingId,
  className,
  children,
}: {
  title: string;
  description?: string;
  actions?: ReactNode;
  /** Required so the wrapping <section> has an accessible name. */
  headingId: string;
  className?: string;
  children: ReactNode;
}) {
  return (
    <section
      aria-labelledby={headingId}
      className={cn("flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4", className)}
    >
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="flex flex-col gap-0.5">
          <h3 id={headingId} className="text-sm font-semibold text-app-foreground">
            {title}
          </h3>
          {description ? (
            <p className="text-xs text-app-muted-foreground">{description}</p>
          ) : null}
        </div>
        {actions ? <div className="flex items-center gap-2">{actions}</div> : null}
      </div>
      {children}
    </section>
  );
}

/** A definition-list row: muted term, foreground value. Used for dense metadata. */
export function MetaRow({
  term,
  children,
}: {
  term: string;
  children: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-0.5 sm:flex-row sm:items-baseline sm:gap-3">
      <dt className="shrink-0 text-xs uppercase tracking-wide text-app-muted-foreground sm:w-40">
        {term}
      </dt>
      <dd className="min-w-0 break-words text-sm text-app-foreground">{children}</dd>
    </div>
  );
}
