/**
 * @libraryId react-component-library:Chip
 * @displayName Chip
 * @version 1.0.6
 * @tags ["primitive","control","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:Chip */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

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
        border: "1px solid var(--color-border)",
        borderRadius: "var(--radius-pill)",
        background: selected
          ? "var(--color-primary)"
          : "var(--color-surface-muted)",
        color: selected
          ? "var(--color-primary-foreground)"
          : "var(--color-foreground)",
        paddingInline: 16,
        font: "inherit",
        fontWeight: 650,
      }}
    >
      {children}
    </button>
  );
});
