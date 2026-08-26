/**
 * @libraryId react-component-library:Inline
 * @displayName Inline
 * @description Inline lays out content on the inline axis with wrapping and alignment.
 * @version 1.1.0
 * @tags ["primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource primitives.inline */
import { forwardRef, type CSSProperties, type HTMLAttributes } from "react";

export interface InlineProps extends HTMLAttributes<HTMLDivElement> {
  gap?: string;
  align?: CSSProperties["alignItems"];
  justify?: CSSProperties["justifyContent"];
  fullWidth?: boolean;
}

export const Inline = forwardRef<
  HTMLDivElement,
  InlineProps
>(({ gap = "sm", align = "center", justify, fullWidth = false, className, style, ...props }, ref) => (
  <div
    ref={ref}
    className={className}
    style={{
      display: "flex",
      flexWrap: "wrap",
      alignItems: align,
      justifyContent: justify,
      gap: `var(--space-${gap})`,
      inlineSize: fullWidth ? "100%" : undefined,
      ...style,
    }}
    {...props}
  />
));