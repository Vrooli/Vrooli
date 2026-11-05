/**
 * @libraryId react-component-library:Surface
 * @displayName Surface
 * @description The canonical background and elevation primitive for panels, menus, overlays, and grouped content, pairing surface colour, border, and shadow as one elevation token.
 * @version 1.0.2
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:Surface
 * @vrooliComponentSourceSlot primitives.surface */
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
