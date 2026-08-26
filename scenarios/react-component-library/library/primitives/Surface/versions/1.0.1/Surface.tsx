/**
 * @libraryId react-component-library:Surface
 * @displayName Surface
 * @description Surface maps the four elevation levels to the kit's elevation ramps.
 * @version 1.0.1
 * @tags ["primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource primitives.surface */
import { forwardRef, type HTMLAttributes } from "react";
import "./Surface.css";

export const Surface = forwardRef<
  HTMLDivElement,
  HTMLAttributes<HTMLDivElement> & {
    elevation?: "flat" | "raised" | "floating" | "overlay";
  }
>(({ elevation = "flat", className, ...props }, ref) => (
  <div
    data-testid="primitives.surface"
    ref={ref}
    className={className}
    data-elevation={elevation}
    data-rcl-surface
    {...props}
  />
));
