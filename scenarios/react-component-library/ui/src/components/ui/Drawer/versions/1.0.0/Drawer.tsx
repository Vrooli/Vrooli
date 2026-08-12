/**
 * @vrooliComponentSource react-component-library:Drawer
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption af7d2d72-40c6-4f05-83f5-5374630bdd5c
 * @vrooliComponentAppliedAt 2026-08-12T12:59:53Z
 * @vrooliComponentSourceSha256 b39466e766e591d8b18589eacdde3db9d554d0f28e89dac4a39e3e89d9227bae
 * @vrooliComponentDriftHash 7f61007cce2fec767148aa2c31df4d1ade4442b7acb31ba981effef31cedc0af
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
