/** @vrooliComponentSource primitives.scroll-area */
import { forwardRef, type HTMLAttributes } from "react";

export const ScrollArea = forwardRef<
  HTMLDivElement,
  HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={className}
    role="region"
    aria-label="Scrollable content"
    style={{
      overflow: "auto",
      WebkitOverflowScrolling: "touch",
      ...props.style,
    }}
    {...props}
  />
));
