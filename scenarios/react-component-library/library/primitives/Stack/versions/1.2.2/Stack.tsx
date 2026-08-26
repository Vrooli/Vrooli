/**
 * @libraryId react-component-library:Stack
 * @displayName Stack
 * @description Stack lays out content on the block axis with a token-backed gap.
 * @version 1.2.2
 * @tags ["primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource primitives.stack */
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
