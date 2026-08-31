/**
 * @libraryId react-component-library:NotificationBadge
 * @displayName Notification Badge
 * @description A reusable count or dot indicator anchored to any interactive or visual surface.
 * @version 1.0.2
 * @tags ["feedback","primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import type { HTMLAttributes, ReactNode } from "react";

export type NotificationBadgeTone = "neutral" | "info" | "success" | "warning" | "danger";

export interface NotificationBadgeProps extends HTMLAttributes<HTMLSpanElement> {
  children: ReactNode;
  value?: number | string;
  max?: number;
  dot?: boolean;
  tone?: NotificationBadgeTone;
  badgeLabel?: string;
}

const notificationBadgeStyles = `
[data-rcl-notification-badge] { position: relative; display: inline-flex; min-inline-size: 0; }
[data-rcl-notification-badge-anchor] { display: inline-flex; min-inline-size: 0; }
[data-rcl-notification-badge-indicator] { position: absolute; inset-block-start: calc(var(--space-3xs) * -1); inset-inline-end: calc(var(--space-3xs) * -1); display: inline-flex; min-inline-size: var(--space-sm); block-size: var(--space-sm); align-items: center; justify-content: center; border: var(--border-hairline) solid var(--color-surface-raised); border-radius: var(--radius-pill); background: var(--color-primary); color: var(--color-background); padding-inline: var(--space-3xs); font: 700 var(--text-caption-size) / 1 var(--font-sans); white-space: nowrap; pointer-events: none; }
[data-rcl-notification-badge-indicator][data-tone="neutral"] { background: var(--color-muted-foreground); }
[data-rcl-notification-badge-indicator][data-tone="info"] { background: var(--color-info); }
[data-rcl-notification-badge-indicator][data-tone="success"] { background: var(--color-success); }
[data-rcl-notification-badge-indicator][data-tone="warning"] { background: var(--color-warning); }
[data-rcl-notification-badge-indicator][data-tone="danger"] { background: var(--color-danger); }
[data-rcl-notification-badge-indicator][data-dot="true"] { min-inline-size: var(--space-2xs); inline-size: var(--space-2xs); block-size: var(--space-2xs); padding: 0; }
@media (prefers-reduced-motion: reduce) { [data-rcl-notification-badge-indicator] { transition: none; } }
`;

function formatValue(value: number | string, max: number) {
  if (typeof value !== "number" || !Number.isFinite(value) || value <= max) return String(value);
  return `${max}+`;
}

export const NotificationBadge = withClassName(function NotificationBadge({
  children,
  value,
  max = 99,
  dot = false,
  tone = "neutral",
  badgeLabel,
  ...props
}: NotificationBadgeProps) {
  const visible = dot || (value !== undefined && value !== null && String(value).length > 0);
  return (
    <>
      <StyleSheet name="notification-badge-1-0-0" css={notificationBadgeStyles} />
      <span data-rcl-notification-badge {...props}>
        <span data-rcl-notification-badge-anchor>{children}</span>
        {visible ? (
          <span
            aria-hidden="true"
            data-dot={dot ? "true" : "false"}
            data-rcl-notification-badge-indicator
            data-tone={tone}
            title={badgeLabel}
          >
            {dot ? null : formatValue(value ?? "", max)}
          </span>
        ) : null}
      </span>
    </>
  );
});
