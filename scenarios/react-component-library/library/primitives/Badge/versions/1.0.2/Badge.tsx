/**
 * @libraryId react-component-library:Badge
 * @displayName Badge
 * @description
 * @version 1.0.2
 * @tags ["primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource primitives.badge */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { HTMLAttributes } from "react";
import "./Badge.css";

export const Badge = withClassName(function Badge({
  tone = "neutral",
  children,
  style,
  ...props
}: HTMLAttributes<HTMLSpanElement> & {
  tone?: "neutral" | "info" | "success" | "warning" | "danger";
}) {
  return (
    <span
      data-testid="primitives.badge"
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
