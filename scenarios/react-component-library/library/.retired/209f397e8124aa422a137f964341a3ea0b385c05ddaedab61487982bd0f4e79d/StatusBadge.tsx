/**
 * @libraryId react-component-library:StatusBadge
 * @displayName Status Badge
 * @description Semantic status badge with accessible labels and token-bound color roles.
 * @version 1.2.1
 * @tags ["feedback","status"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

import type { HTMLAttributes, ReactNode } from "react";
import { statusBadgeStyles } from "./styles";

export type StatusTone = "neutral" | "success" | "warning" | "danger" | "info";

export interface StatusBadgeProps extends HTMLAttributes<HTMLSpanElement> {
  children: ReactNode;
  tone?: StatusTone;
}

export const StatusBadge = withClassName(function StatusBadge({
  children,
  className,
  tone = "neutral",
  ...props
}: StatusBadgeProps) {
  useLibraryStyleSheet("status-badge", statusBadgeStyles);
  return (
    <span {...props} className={className} data-rcl-status-badge data-tone={tone}>
      <span data-rcl-status-badge-indicator aria-hidden="true" />
      <span data-rcl-status-badge-label>{children}</span>
    </span>
  );
});
