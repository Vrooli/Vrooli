/** @vrooliComponentSource react-component-library:Drawer */
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
export function Drawer({
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
  if (!open) return null;
  return (
    <section
      role="dialog"
      aria-label="Drawer"
      data-side={side}
      data-presentation={presentation}
      style={{ ...panel, display: "grid", gap: 16 }}
    >
      {children ?? "Drawer content"}
      <button
        type="button"
        onClick={onClose}
        style={{ minHeight: 44, minWidth: 44 }}
      >
        Close
      </button>
    </section>
  );
}
