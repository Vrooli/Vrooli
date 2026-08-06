/** @vrooliComponentSource primitives.container */
import { forwardRef, type HTMLAttributes } from "react";

export const Container = forwardRef<
  HTMLDivElement,
  HTMLAttributes<HTMLDivElement> & { width?: "content" | "wide" | "full" }
>(({ width = "content", className, ...props }, ref) => (
  <div
    ref={ref}
    className={className}
    data-container-width={width}
    style={{
      width: "100%",
      maxWidth:
        width === "full"
          ? "none"
          : width === "wide"
            ? "var(--container-wide)"
            : "var(--container-content)",
      marginInline: "auto",
      ...props.style,
    }}
    {...props}
  />
));
