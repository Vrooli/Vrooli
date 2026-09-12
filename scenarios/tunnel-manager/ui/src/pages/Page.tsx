import type { ReactNode } from "react";

import { useDocumentTitle } from "../hooks/useDocumentTitle";

interface PageProps {
  testId: string;
  /** Stable id for the <h1>; aria-labelledby points the section at it. */
  headingId: string;
  title: string;
  description?: string;
  experienceSurface?: string;
  children: ReactNode;
}

/**
 * Page is the shared surface shell: a labelled <section> with a heading,
 * optional description, and a document title. Every routed page composes it so
 * headings, titles, and landmark labelling stay consistent across surfaces.
 */
export function Page({ testId, headingId, title, description, experienceSurface, children }: PageProps) {
  useDocumentTitle(title);

  return (
    <section data-testid={testId} data-experience-surface={experienceSurface} aria-labelledby={headingId} className="flex flex-col gap-4">
      <header className="flex flex-col gap-1">
        <h1 id={headingId} className="text-2xl font-semibold">
          {title}
        </h1>
        {description && <p className="text-app-muted-foreground">{description}</p>}
      </header>
      {children}
    </section>
  );
}
