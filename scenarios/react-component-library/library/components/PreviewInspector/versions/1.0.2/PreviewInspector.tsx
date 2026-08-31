/**
 * @libraryId react-component-library:PreviewInspector
 * @displayName PreviewInspector
 * @description A dismissible slide-over for props, inspector details, and preview events.
 * @version 1.0.2
 * @tags ["preview","overlay","inspection"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { ReactNode } from "react";

export const PreviewInspector = withClassName(function PreviewInspector({
  title = "Preview inspector",
  description,
  children,
  open = true,
  onClose,
}: {
  title?: string;
  description?: string;
  children?: ReactNode;
  open?: boolean;
  onClose?: () => void;
}) {
  if (!open) return null;
  return (
    <aside
      data-testid="preview.inspector-drawer"
      data-preview-inspector
      aria-label={title}
      style={{
        display: "flex",
        flexDirection: "column",
        gap: "var(--space-sm)",
        width: "min(100%, 30rem)",
        minWidth: 0,
        padding: "var(--space-md)",
        border: "var(--border-hairline) solid var(--color-border)",
        borderRadius: "var(--radius-panel)",
        background: "color-mix(in srgb, var(--color-surface) 98%, transparent)",
        boxShadow: "var(--elev-overlay)",
      }}
    >
      <header
        style={{
          display: "flex",
          alignItems: "flex-start",
          justifyContent: "space-between",
          gap: "var(--space-sm)",
        }}
      >
        <div style={{ display: "grid", gap: "var(--space-3xs)", minWidth: 0 }}>
          <h2 style={{ margin: 0, font: "var(--text-heading)" }}>{title}</h2>
          {description ? (
            <p
              style={{
                margin: 0,
                color: "var(--color-muted-foreground)",
                font: "var(--text-body-sm)",
              }}
            >
              {description}
            </p>
          ) : null}
        </div>
        {onClose ? (
          <button
            type="button"
            onClick={onClose}
            aria-label="Close inspector"
            style={{
              minHeight: "var(--tap-target-min)",
              paddingInline: "var(--space-sm)",
              border: "var(--border-hairline) solid var(--color-border)",
              borderRadius: "var(--radius-control)",
              background: "var(--color-surface)",
              color: "var(--color-foreground)",
            }}
          >
            Close
          </button>
        ) : null}
      </header>
      <div style={{ minWidth: 0 }}>{children}</div>
    </aside>
  );
});
