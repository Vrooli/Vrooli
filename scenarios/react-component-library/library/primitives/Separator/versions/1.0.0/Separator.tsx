/** @vrooliComponentSource primitives.separator */
import type { HTMLAttributes } from "react";

export function Separator({
  orientation = "horizontal",
  ...props
}: HTMLAttributes<HTMLHRElement> & {
  orientation?: "horizontal" | "vertical";
}) {
  return (
    <hr
      aria-orientation={orientation}
      data-orientation={orientation}
      style={{
        border: 0,
        background: "var(--color-border)",
        ...(orientation === "vertical"
          ? { width: "var(--separator-thickness)", height: "100%" }
          : { height: "var(--separator-thickness)", width: "100%" }),
        ...props.style,
      }}
      {...props}
    />
  );
}
