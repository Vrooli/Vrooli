/** @vrooliComponentSource primitives.badge */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import type { HTMLAttributes } from "react";
import "./Badge.css";

export const Badge = withClassName(function Badge({
  tone = "neutral",
  children,
  style,
  ...props
}: HTMLAttributes<HTMLSpanElement> & { tone?: "neutral" | "info" | "success" | "warning" | "danger" }) {
  return (
    <span data-testid="primitives.badge"
      role="status"
      data-rcl-badge
      data-tone={tone}
      style={style}
      {...props}
    >
      {children}
    </span>
  );
});
