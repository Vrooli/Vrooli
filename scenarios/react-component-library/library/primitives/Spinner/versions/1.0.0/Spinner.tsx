/** @vrooliComponentSource primitives.spinner */
import type { HTMLAttributes } from "react";

export function Spinner({
  label = "Loading",
  ...props
}: HTMLAttributes<HTMLDivElement> & { label?: string }) {
  return (
    <div
      role="status"
      aria-label={label}
      data-spinner="true"
      style={{
        width: "var(--icon-size-md)",
        height: "var(--icon-size-md)",
        border: "var(--spinner-border)",
        borderTopColor: "var(--color-primary)",
        borderRadius: "var(--radius-pill)",
        animation: "vrooli-spin var(--dur-slow) linear infinite",
        ...props.style,
      }}
      {...props}
    />
  );
}
