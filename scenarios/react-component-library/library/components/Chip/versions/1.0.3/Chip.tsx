/**
 * @libraryId react-component-library:Chip
 * @displayName Chip
 * @description A compact selectable token that preserves a full touch target while remaining visually quiet in dense collections.
 * @version 1.0.3
 * @tags ["primitive","control","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:Chip */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import type { ButtonHTMLAttributes, ReactNode } from "react";

export interface ChipProps extends Pick<ButtonHTMLAttributes<HTMLButtonElement>, "aria-label"> {
  children: ReactNode;
  selected?: boolean;
  onClick?: () => void;
}

export const Chip = withClassName(function Chip({
  children,
  selected = false,
  onClick,
  "aria-label": ariaLabel,
}: ChipProps) {
  return (
    <button
      data-testid="primitives.chip"
      type="button"
      aria-pressed={selected}
      aria-label={ariaLabel}
      onClick={onClick}
      style={{
        minHeight: 36,
        border: "1px solid var(--color-border, #cbd5e1)",
        borderRadius: "var(--radius-pill, 9999px)",
        background: selected
          ? "var(--color-primary, #2563eb)"
          : "var(--color-surface-muted, #f1f5f9)",
        color: selected
          ? "var(--color-primary-foreground, #fff)"
          : "var(--color-foreground, #0f172a)",
        paddingInline: 16,
        font: "inherit",
        fontWeight: 650,
      }}
    >
      {children}
    </button>
  );
});
