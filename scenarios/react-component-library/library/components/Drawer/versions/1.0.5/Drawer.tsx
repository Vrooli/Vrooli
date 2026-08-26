/**
 * @libraryId react-component-library:Drawer
 * @displayName Drawer
 * @description A layered side surface with predictable dismissal, focus semantics, and token-backed elevation.
 * @version 1.0.5
 * @tags ["overlay","layered","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:Drawer */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
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
export type DrawerSide = "left" | "right" | "top" | "bottom";
export type DrawerPresentation = "modal" | "non-modal";
export const Drawer = withClassName(function Drawer({
  open = false,
  onClose,
  children,
  side = "right",
  presentation = "modal",
}: {
  open?: boolean;
  onClose?: () => void;
  children?: ReactNode;
  side?: DrawerSide;
  presentation?: DrawerPresentation;
}) {
  const strings = useStrings();
  if (!open) return null;
  return (
    <section
      role="dialog"
      aria-label={strings("overlays.drawer.drawer", "Drawer")}
      data-side={side}
      data-presentation={presentation}
      style={{ ...panel, display: "grid", gap: 16 }}
    >
      {children ?? "Drawer content"}
      <button
        data-testid="overlays.drawer"
        type="button"
        onClick={onClose}
        style={{ minHeight: 44, minWidth: 44 }}
      >
        {strings("overlays.drawer.close", "Close")}
      </button>
    </section>
  );
});
