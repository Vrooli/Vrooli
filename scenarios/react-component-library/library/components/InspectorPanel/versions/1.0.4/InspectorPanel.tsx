/**
 * @libraryId react-component-library:InspectorPanel
 * @displayName InspectorPanel
 * @description A layered inspection surface that preserves the host context while exposing contextual detail and dismissal.
 * @version 1.0.4
 * @tags ["overlay","layered","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:InspectorPanel */
import { translate } from "../../../../hooks/useLocale/versions/1.0.1/useLocale";
import { withClassName } from "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge";

import type { ReactNode } from "react";
const panel = {
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, .75rem)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  boxShadow: "var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08))",
};
export const InspectorPanel = withClassName(function InspectorPanel({
  open = false,
  onClose,
  children,
}: {
  open?: boolean;
  onClose?: () => void;
  children?: ReactNode;
}) {
  if (!open) return null;
  return (
    <section
      role="dialog"
      aria-label={translate("overlays.inspector-panel.aria-label.1", "Inspector")}
      style={{ ...panel, display: "grid", gap: 16 }}
    >
      {children ?? "Inspector details"}
      <button data-testid="overlays.inspector-panel" type="button" onClick={onClose}>
        {translate("overlays.inspector-panel.text.1", "Close")}
      </button>
    </section>
  );
});
