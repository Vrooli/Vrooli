/** @vrooliComponentSource primitives.cluster */
import { forwardRef, type CSSProperties, type HTMLAttributes } from "react";

const layoutStyle = (
  gap: string | undefined,
  extra?: CSSProperties,
): CSSProperties => ({
  gap: gap ? `var(--space-${gap})` : undefined,
  ...extra,
});

export const Cluster = forwardRef<
  HTMLDivElement,
  HTMLAttributes<HTMLDivElement> & { gap?: string }
>(({ gap = "sm", className, ...props }, ref) => (
  <div data-testid="primitives.cluster"
    ref={ref}
    className={className}
    style={layoutStyle(gap, {
      display: "flex",
      flexWrap: "wrap",
      alignItems: "baseline",
    })}
    {...props}
  />
));
