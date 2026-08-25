/** @vrooliComponentSource primitives.skeleton */
import { translate } from "../../../../hooks/useLocale/versions/1.0.0/useLocale";

import type { HTMLAttributes } from "react";

export function Skeleton({
  label = translate("primitives.skeleton.label.1", "Loading"),
  ...props
}: HTMLAttributes<HTMLDivElement> & { label?: string }) {
  return (
    <div
      role="status"
      aria-label={label}
      data-skeleton="true"
      style={{
        background: "var(--color-surface-muted)",
        borderRadius: "var(--radius-sm)",
        minHeight: "var(--skeleton-line-height)",
        ...props.style,
      }}
      {...props}
    />
  );
}
