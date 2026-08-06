/** @vrooliComponentSource primitives.surface */
import { forwardRef, type HTMLAttributes } from "react";

export const Surface = forwardRef<
  HTMLDivElement,
  HTMLAttributes<HTMLDivElement> & {
    elevation?: "flat" | "raised" | "floating" | "overlay";
  }
>(({ elevation = "flat", className, ...props }, ref) => (
  <div
    ref={ref}
    className={className}
    data-elevation={elevation}
    style={{
      background: "var(--app-surface)",
      border: "var(--surface-border)",
      borderRadius: "var(--panel-radius)",
      boxShadow: `var(--elev-${elevation})`,
      ...props.style,
    }}
    {...props}
  />
));
