/**
 * @libraryId react-component-library:PreviewDock
 * @displayName PreviewDock
 * @description Overlay controls that keep preview navigation visible without taking space from the specimen.
 * @version 1.0.2
 * @tags ["preview","chrome","responsive"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { ReactNode } from "react";

export const PreviewDock = withClassName(function PreviewDock({
  children,
  label = "Preview controls",
}: {
  children?: ReactNode;
  label?: string;
}) {
  return (
    <div
      data-testid="preview.preview-dock"
      data-preview-dock
      role="toolbar"
      aria-label={label}
      style={{
        position: "relative",
        zIndex: 1,
        display: "flex",
        flexWrap: "wrap",
        alignItems: "center",
        gap: "var(--space-2xs)",
        minWidth: 0,
        padding: "var(--space-2xs)",
        border:
          "var(--border-hairline) solid color-mix(in srgb, var(--color-border) 80%, transparent)",
        borderRadius: "var(--radius-panel)",
        background: "color-mix(in srgb, var(--color-surface) 94%, transparent)",
        boxShadow: "var(--elev-raised)",
        backdropFilter: "blur(14px)",
      }}
    >
      {children}
    </div>
  );
});
