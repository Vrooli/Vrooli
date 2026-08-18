/**
 * @vrooliComponentSource react-component-library:Drawer
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption af7d2d72-40c6-4f05-83f5-5374630bdd5c
 * @vrooliComponentAppliedAt 2026-08-17T23:22:48Z
 * @vrooliComponentSourceSha256 220b576832958f44969f5f09a975f4b265a9586cf335ffc2230000420e7a79e9
 * @vrooliComponentDriftHash 2ab6dd9cb1289997cbe7c5dbda5a05c1ed7b095b247ca95cc7a72afee7a4bba4
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
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
      <button type="button" onClick={onClose} style={{ minHeight: 44, minWidth: 44 }}>
        Close
      </button>
    </section>
  );
}
