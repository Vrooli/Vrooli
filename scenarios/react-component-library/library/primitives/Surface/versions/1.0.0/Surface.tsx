/** @vrooliComponentSource primitives.surface */
import { forwardRef, type HTMLAttributes } from "react";
import "./Surface.css";

export const Surface = forwardRef<
  HTMLDivElement,
  HTMLAttributes<HTMLDivElement> & {
    elevation?: "flat" | "raised" | "floating" | "overlay";
  }
>(({ elevation = "flat", className, ...props }, ref) => (
  <div data-testid="primitives.surface"
    ref={ref}
    className={className}
    data-elevation={elevation}
    data-rcl-surface
    {...props}
  />
));
