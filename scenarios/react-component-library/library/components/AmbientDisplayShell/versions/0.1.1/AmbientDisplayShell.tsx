/**
 * @libraryId react-component-library:AmbientDisplayShell
 * @displayName AmbientDisplayShell
 * @description Reusable command-display primitive
 * @version 0.1.1
 * @tags ["ambient-display","command-center"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import * as React from "react";
export interface AmbientDisplayShellProps {
  title?: string;
  children?: React.ReactNode;
}
export default function AmbientDisplayShell({ title, children }: AmbientDisplayShellProps) {
  return (
    <section className="rcl-ambient-shell" aria-label={title}>
      {title && <h1>{title}</h1>}
      {children}
    </section>
  );
}
