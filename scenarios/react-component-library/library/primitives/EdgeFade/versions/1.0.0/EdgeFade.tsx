/** @vrooliComponentSource primitives.edge-fade */
import type { HTMLAttributes } from "react";

export function EdgeFade({
  side = "inline-end",
  ...props
}: HTMLAttributes<HTMLDivElement> & { side?: "inline-start" | "inline-end" }) {
  return (
    <div
      aria-hidden
      style={{
        pointerEvents: "none",
        [side === "inline-start" ? "insetInlineStart" : "insetInlineEnd"]: 0,
        position: "absolute",
        insetBlock: 0,
        width: "var(--edge-fade-width)",
        background: "var(--edge-fade)",
        ...props.style,
      }}
      {...props}
    />
  );
}
