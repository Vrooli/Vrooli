import type { ReactNode } from "react";
import { Panel } from "./Panel";

interface PageProps {
  title: string;
  description?: string;
  eyebrow?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
  /** `fill` pins the header and lets the body own scrolling (conversation layouts). */
  layout?: "scroll" | "fill";
  /**
   * Phone-width full-bleed: drops page padding and hides the title row, so a
   * detail pane owns the whole viewport the way a messaging app does.
   * `EXPERIENCE.md` — "the detail pane is the largest thing on screen".
   */
  mobileBleed?: boolean;
  testId?: string;
  headingId?: string;
}

/**
 * Page frame: compact title row, optional description, actions on the right,
 * body below. Unframed by design — cards are for repeated records, not for
 * wrapping whole page sections.
 */
export function Page({ title, description, eyebrow, actions, children, layout = "scroll", testId, headingId = "page-heading", mobileBleed = false }: PageProps) {
  return (
    <section
      data-testid={testId}
      aria-labelledby={headingId}
      aria-label={mobileBleed ? title : undefined}
      className={[
        "flex min-w-0 flex-1 flex-col",
        mobileBleed ? "p-0 md:p-6" : "p-4 md:p-6",
        layout === "fill" ? "h-full min-h-0" : "min-h-0 gap-5 overflow-auto",
      ].join(" ")}
    >
      <header className={[
        mobileBleed ? "hidden md:flex" : "flex",
        "flex-wrap items-start justify-between gap-3",
        layout === "fill" ? "mb-4 shrink-0" : "",
      ].join(" ")}>
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
    <Panel className="overflow-hidden p-0">
      <dl className="grid grid-cols-2 gap-px bg-app-border sm:grid-cols-4">
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
    </Panel>
  );
}
