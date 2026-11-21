import type { ReactNode } from "react";

/** A titled block of a page. Headings are h2 so each page has one outline. */
export function Section({ title, actions, children, id, note }: { title: ReactNode; actions?: ReactNode; children: ReactNode; id?: string; note?: ReactNode }) {
  return (
    <section className="rd-section" aria-labelledby={id ? `${id}-title` : undefined} id={id}>
      <div className="flex flex-wrap items-center justify-between gap-space-xs">
        <h2 id={id ? `${id}-title` : undefined} className="text-subheading font-semibold">{title}</h2>
        {actions}
      </div>
      {note ? <p className="text-body-sm text-app-muted-foreground">{note}</p> : null}
      {children}
    </section>
  );
}

/** A definition list laid out for a phone: label above value, two columns on wide screens. */
export function Facts({ children }: { children: ReactNode }) {
  return <dl className="rd-facts">{children}</dl>;
}
