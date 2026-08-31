/**
 * @libraryId react-component-library:CanvasFrame
 * @displayName CanvasFrame
 * @description The shared preview substrate that owns fit, background, and specimen placement for focus and canvas views.
 * @version 1.0.2
 * @tags ["preview","layout","canvas"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { ReactNode } from "react";

export const CanvasFrame = withClassName(function CanvasFrame({
  children,
  mode = "focus",
  title = "Preview canvas",
}: {
  children?: ReactNode;
  mode?: "focus" | "canvas";
  title?: string;
}) {
  return (
    <section
      data-testid="preview.canvas-frame"
      data-canvas-frame
      data-mode={mode}
      aria-label={title}
      style={{
        minWidth: 0,
        minHeight: "18rem",
        overflow: "hidden",
        border: "var(--border-hairline) solid var(--color-border)",
        borderRadius: "var(--radius-panel)",
        background: "var(--color-surface-muted)",
      }}
    >
      <header
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          gap: "var(--space-sm)",
          padding: "var(--space-sm) var(--space-md)",
          borderBottom: "var(--border-hairline) solid var(--color-border)",
          background: "var(--color-surface)",
        }}
      >
        <h2 style={{ margin: 0, font: "var(--text-heading)" }}>{title}</h2>
        <span
          role="status"
          style={{ color: "var(--color-muted-foreground)", font: "var(--text-caption)" }}
        >
          {mode === "focus" ? "One specimen" : "Spatial canvas"}
        </span>
      </header>
      <div style={{ minHeight: "14rem", padding: "var(--space-lg)" }}>{children}</div>
    </section>
  );
});
