/** @vrooliComponentSource primitives.stack */
import { forwardRef, type CSSProperties, type HTMLAttributes } from "react";

const layoutStyle = (
  gap: string | undefined,
  extra?: CSSProperties,
): CSSProperties => ({
  gap: gap ? `var(--space-${gap})` : undefined,
  ...extra,
});

export const Stack = forwardRef<
  HTMLDivElement,
  HTMLAttributes<HTMLDivElement> & { gap?: string }
>(({ gap = "md", className, ...props }, ref) => (
  <div
    ref={ref}
    className={className}
    style={layoutStyle(gap, { display: "flex", flexDirection: "column" })}
    {...props}
  />
));
