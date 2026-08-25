/**
 * @libraryId react-component-library:PreviewInspector
 * @version 1.0.0
 * @status released
 * @deps {"react":"^18"}
 */
import { translate } from "../../../../hooks/useLocale/versions/1.0.0/useLocale";

import type { ReactNode } from "react";

export function PreviewInspector({
  title = translate("preview.inspector-drawer.title.1", "Preview inspector"),
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
      <header style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", gap: "var(--space-sm)" }}>
        <div style={{ display: "grid", gap: "var(--space-3xs)", minWidth: 0 }}>
          <h2 style={{ margin: 0, font: "var(--text-heading)" }}>{title}</h2>
          {description ? <p style={{ margin: 0, color: "var(--color-muted-foreground)", font: "var(--text-body-sm)" }}>{description}</p> : null}
        </div>
        {onClose ? <button type="button" onClick={onClose} aria-label={translate("preview.inspector-drawer.aria-label.2", "Close inspector")} style={{ minHeight: "var(--tap-target-min)", paddingInline: "var(--space-sm)", border: "var(--border-hairline) solid var(--color-border)", borderRadius: "var(--radius-control)", background: "var(--color-surface)", color: "var(--color-foreground)" }}>{translate("preview.inspector-drawer.text.3", "Close")}</button> : null}
      </header>
      <div style={{ minWidth: 0 }}>{children}</div>
    </aside>
  );
}
