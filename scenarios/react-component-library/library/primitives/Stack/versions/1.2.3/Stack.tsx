/**
 * @libraryId react-component-library:Stack
 * @displayName Stack
 * @description The vertical layout primitive giving consistent spacing, alignment, optional separators, wrapping, responsive values, and a choice of semantic element.
 * @version 1.2.3
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:Stack
 * @vrooliComponentSourceSlot primitives.stack */
import { forwardRef, type CSSProperties, type HTMLAttributes } from "react";
import "./Stack.css";

export type StackMeasure = "none" | "narrow" | "content" | "wide";

export interface StackProps extends HTMLAttributes<HTMLDivElement> {
  gap?: string;
  align?: CSSProperties["alignItems"];
  justify?: CSSProperties["justifyContent"];
  textAlign?: CSSProperties["textAlign"];
  measure?: StackMeasure;
  inset?: string;
  insetBlock?: string;
  insetInline?: string;
}

export const Stack = forwardRef<HTMLDivElement, StackProps>(
  (
    {
      gap = "md",
      align,
      justify,
      textAlign,
      measure = "none",
      inset,
      insetBlock,
      insetInline,
      className,
      style,
      ...props
    },
    ref,
  ) => (
    <div
      data-testid="primitives.stack"
      ref={ref}
      className={className}
      data-rcl-stack
      data-gap={gap}
      data-align={align}
      data-justify={justify}
      data-text-align={textAlign}
      data-measure={measure}
      data-inset={inset}
      data-inset-block={insetBlock}
      data-inset-inline={insetInline}
      style={style}
      {...props}
    />
  ),
);
