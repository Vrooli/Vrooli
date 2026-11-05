/**
 * @libraryId react-component-library:Cluster
 * @displayName Cluster
 * @description The wrapping group primitive for tags, actions, metadata, and controls whose items must keep useful spacing at any available width.
 * @version 1.0.2
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:Cluster
 * @vrooliComponentSourceSlot primitives.cluster */
import { forwardRef, type CSSProperties, type HTMLAttributes } from "react";

const layoutStyle = (gap: string | undefined, extra?: CSSProperties): CSSProperties => ({
  gap: gap ? `var(--space-${gap})` : undefined,
  ...extra,
});

export const Cluster = forwardRef<
  HTMLDivElement,
  HTMLAttributes<HTMLDivElement> & { gap?: string }
>(({ gap = "sm", className, ...props }, ref) => (
  <div
    data-testid="primitives.cluster"
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
