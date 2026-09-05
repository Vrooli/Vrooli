/**
 * @libraryId react-component-library:StatusBadge
 * @displayName Status Badge
 * @version 1.2.2
 * @tags ["feedback","status"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

import type { HTMLAttributes, ReactNode } from "react";
export const statusBadgeStyles = `
[data-rcl-status-badge] { display: inline-flex; min-block-size: calc(var(--text-label-line) + (var(--space-3xs) * 2)); max-inline-size: 100%; align-items: center; gap: var(--space-3xs); box-sizing: border-box; border: var(--border-hairline) solid var(--rcl-status-border, var(--color-border)); border-radius: var(--radius-pill); background: var(--rcl-status-surface, var(--color-surface-muted)); color: var(--rcl-status-accent, var(--color-muted-foreground)); padding: var(--space-3xs) var(--space-xs); font: var(--text-label); white-space: nowrap; }
[data-rcl-status-badge][data-tone="neutral"] { --rcl-status-accent: var(--color-muted-foreground); --rcl-status-surface: var(--color-surface-muted); --rcl-status-border: var(--color-border); }
[data-rcl-status-badge][data-tone="success"] { --rcl-status-accent: var(--color-success); --rcl-status-surface: color-mix(in srgb, var(--color-success) 10%, var(--color-surface)); --rcl-status-border: color-mix(in srgb, var(--color-success) 32%, var(--color-border)); }
[data-rcl-status-badge][data-tone="warning"] { --rcl-status-accent: var(--color-warning); --rcl-status-surface: color-mix(in srgb, var(--color-warning) 11%, var(--color-surface)); --rcl-status-border: color-mix(in srgb, var(--color-warning) 38%, var(--color-border)); }
[data-rcl-status-badge][data-tone="danger"] { --rcl-status-accent: var(--color-danger); --rcl-status-surface: color-mix(in srgb, var(--color-danger) 10%, var(--color-surface)); --rcl-status-border: color-mix(in srgb, var(--color-danger) 34%, var(--color-border)); }
[data-rcl-status-badge][data-tone="info"] { --rcl-status-accent: var(--color-info); --rcl-status-surface: color-mix(in srgb, var(--color-info) 10%, var(--color-surface)); --rcl-status-border: color-mix(in srgb, var(--color-info) 32%, var(--color-border)); }
[data-rcl-status-badge-indicator] { inline-size: var(--space-2xs); block-size: var(--space-2xs); flex: 0 0 auto; border-radius: var(--radius-pill); background: currentColor; box-shadow: 0 0 0 var(--space-3xs) color-mix(in srgb, currentColor 12%, transparent); }
[data-rcl-status-badge-label] { min-inline-size: 0; overflow: hidden; text-overflow: ellipsis; }

`;
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
