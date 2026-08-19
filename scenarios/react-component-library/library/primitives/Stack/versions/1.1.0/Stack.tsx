/**
 * @libraryId react-component-library:Stack
 * @displayName Stack
 * @description Stack lays out content on the block axis with a token-backed gap.
 * @version 1.1.0
 * @tags ["primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource primitives.stack */
import { forwardRef, type CSSProperties, type HTMLAttributes } from "react";

export type StackMeasure = "none" | "narrow" | "content" | "wide";

const measureWidths: Record<Exclude<StackMeasure, "none">, string> = {
  narrow: "var(--layout-content-compact, 28rem)",
  content: "var(--layout-content, 34rem)",
  wide: "var(--layout-content-wide, 42rem)",
};

export interface StackProps extends HTMLAttributes<HTMLDivElement> {
  gap?: string;
  align?: CSSProperties["alignItems"];
  justify?: CSSProperties["justifyContent"];
  textAlign?: CSSProperties["textAlign"];
  measure?: StackMeasure;
  inset?: string;
}

export const Stack = forwardRef<
  HTMLDivElement,
  StackProps
>(({ gap = "md", align, justify, textAlign, measure = "none", inset, className, style, ...props }, ref) => (
  <div
    ref={ref}
    className={className}
    style={{
      display: "flex",
      flexDirection: "column",
      gap: `var(--space-${gap})`,
      alignItems: align,
      justifyContent: justify,
      textAlign,
      inlineSize: measure === "none" ? undefined : "100%",
      maxInlineSize:
        measure === "none" ? undefined : measureWidths[measure],
      marginInline: measure === "none" ? undefined : "auto",
      padding: inset ? `var(--space-${inset})` : undefined,
      ...style,
    }}
    {...props}
  />
));