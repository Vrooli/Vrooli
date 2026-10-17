/**
 * @libraryId react-component-library:Inline
 * @displayName Inline
 * @description The horizontal layout primitive handling wrapping, alignment, distribution, responsive spacing, and bidirectional reading order.
 * @version 1.1.2
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:Inline
 * @vrooliComponentSourceSlot primitives.inline */
import { forwardRef, type CSSProperties, type HTMLAttributes } from "react";
import "./Inline.css";

export interface InlineProps extends HTMLAttributes<HTMLDivElement> {
  gap?: string;
  align?: CSSProperties["alignItems"];
  justify?: CSSProperties["justifyContent"];
  fullWidth?: boolean;
}

export const Inline = forwardRef<HTMLDivElement, InlineProps>(
  (
    { gap = "sm", align = "center", justify, fullWidth = false, className, style, ...props },
    ref,
  ) => (
    <div
      data-testid="primitives.inline"
      ref={ref}
      className={className}
      data-rcl-inline
      data-gap={gap}
      data-align={align}
      data-justify={justify}
      data-full-width={fullWidth || undefined}
      style={style}
      {...props}
    />
  ),
);
