/**
 * @libraryId react-component-library:ResizableSidebar
 * @displayName ResizableSidebar
 * @description A navigation sidebar that preserves a usable minimum while supporting responsive width changes.
 * @version 1.0.3
 * @tags ["navigation","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:ResizableSidebar */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import type { ReactNode } from "react";
const panel = {
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, .75rem)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  boxShadow: "var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08))",
};
export const ResizableSidebar = withClassName(function ResizableSidebar({
  children,
}: {
  children?: ReactNode;
}) {
  return (
    <aside
      data-testid="navigation.resizable-sidebar"
      data-resizable-sidebar
      style={{ ...panel, minInlineSize: 260, maxInlineSize: 480 }}
    >
      {children}
    </aside>
  );
});
