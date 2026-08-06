/** @vrooliComponentSource primitives.skeleton */
import type { HTMLAttributes } from "react";

export function Skeleton({
  label = "Loading",
  ...props
}: HTMLAttributes<HTMLDivElement> & { label?: string }) {
  return (
    <div
      role="status"
      aria-label={label}
      data-skeleton="true"
      style={{
        background: "var(--app-surface-muted)",
        borderRadius: "var(--radius-sm)",
        minHeight: "var(--skeleton-line-height)",
        ...props.style,
      }}
      {...props}
    />
  );
}
