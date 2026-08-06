/** @vrooliComponentSource primitives.badge */
import type { HTMLAttributes } from "react";

const badgeTones = {
  neutral: "var(--app-muted-foreground)",
  info: "var(--app-info)",
  success: "var(--app-success)",
  warning: "var(--app-warning)",
  danger: "var(--app-danger)",
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
