/** @vrooliComponentSource primitives.badge */
import type { HTMLAttributes } from "react";

const badgeTones = {
  neutral: "var(--color-muted-foreground)",
  info: "var(--color-info)",
  success: "var(--color-success)",
  warning: "var(--color-warning)",
  danger: "var(--color-danger)",
} as const;

export function Badge({
  tone = "neutral",
  children,
  ...props
}: HTMLAttributes<HTMLSpanElement> & { tone?: keyof typeof badgeTones }) {
  return (
    <span
      role="status"
      data-tone={tone}
      style={{
        color: badgeTones[tone],
        border: "var(--badge-border)",
        borderRadius: "var(--radius-pill)",
        paddingInline: "var(--space-sm)",
        paddingBlock: "var(--space-3xs)",
        ...props.style,
      }}
      {...props}
    >
      {children}
    </span>
  );
}
