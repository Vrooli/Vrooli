/**
 * @libraryId react-component-library:page-section
 * @displayName Page Section
 * @description Semantic section with heading, optional note and actions for page content
 * @version 0.1.0
 * @tags ["layout","content"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import type { ReactNode } from "react";
import "./PageSectionDraftSource.css";

/** Semantic content section for a page. Leaves the child's surface to its caller. */
export interface PageSectionProps {
  title: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
  id?: string;
  note?: ReactNode;
}

export function PageSection({ title, actions, children, id, note }: PageSectionProps) {
  return (
    <section data-rcl-page-section="" aria-labelledby={id ? `${id}-title` : undefined} id={id}>
      <div data-rcl-page-section-heading="">
        <h2 id={id ? `${id}-title` : undefined}>{title}</h2>
        {actions}
      </div>
      {note ? <p data-rcl-page-section-note="">{note}</p> : null}
      {children}
    </section>
  );
}

export default PageSection;
