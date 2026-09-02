import type { ReactNode } from "react";

interface PageProps {
  title: string;
  description?: string;
  eyebrow?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
  /** `fill` pins the header and lets the body own scrolling (conversation layouts). */
  layout?: "scroll" | "fill";
  testId?: string;
  headingId?: string;
}

/**
 * Page frame: compact title row, optional description, actions on the right,
 * body below. Unframed by design — cards are for repeated records, not for
 * wrapping whole page sections.
 */
export function Page({ title, description, eyebrow, actions, children, layout = "scroll", testId, headingId = "page-heading" }: PageProps) {
  return (
    <section
      data-testid={testId}
      aria-labelledby={headingId}
      className={["flex min-w-0 flex-col", layout === "fill" ? "h-full min-h-0" : "gap-5"].join(" ")}
    >
      <header className={["flex flex-wrap items-start justify-between gap-3", layout === "fill" ? "mb-4 shrink-0" : ""].join(" ")}>
        <div className="min-w-0">
          {eyebrow ? <div className="mb-1 text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">{eyebrow}</div> : null}
          <h2 id={headingId} className="text-xl font-semibold leading-tight text-app-foreground md:text-2xl">
            {title}
          </h2>
          {description ? <p className="mt-1 max-w-prose text-sm text-app-muted-foreground">{description}</p> : null}
        </div>
        {actions ? <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div> : null}
      </header>
      {layout === "fill" ? <div className="flex min-h-0 flex-1 flex-col">{children}</div> : children}
    </section>
  );
}

export function StatStrip({ items }: { items: Array<{ label: string; value: string | number; hint?: string; tone?: "neutral" | "warning" | "danger" | "success"; testId?: string }> }) {
  return (
    <dl className="grid grid-cols-2 gap-px overflow-hidden rounded-panel border border-app-border bg-app-border sm:grid-cols-4">
      {items.map((item) => (
        <div key={item.label} data-testid={item.testId} className="flex flex-col gap-0.5 bg-app-surface px-4 py-3">
          <dt className="text-xs font-medium text-app-muted-foreground">{item.label}</dt>
          <dd
            className={[
              "font-mono text-xl font-semibold tabular-nums leading-tight",
              item.tone === "danger" ? "text-app-danger" : item.tone === "warning" ? "text-app-warning" : item.tone === "success" ? "text-app-success" : "text-app-foreground",
            ].join(" ")}
          >
            {item.value}
          </dd>
          {item.hint ? <dd className="text-xs text-app-muted-foreground">{item.hint}</dd> : null}
        </div>
      ))}
    </dl>
  );
}
