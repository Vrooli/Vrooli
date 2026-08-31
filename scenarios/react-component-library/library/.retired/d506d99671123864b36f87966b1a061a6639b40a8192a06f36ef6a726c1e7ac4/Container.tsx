/**
 * @libraryId react-component-library:Container
 * @displayName Container
 * @description Container constrains readable content widths without hard-coded dimensions.
 * @version 1.1.0
 * @tags ["primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource primitives.container */
import { forwardRef, type CSSProperties, type HTMLAttributes } from "react";

export type ContainerWidth =
  | "compact"
  | "content"
  | "comfortable"
  | "wide"
  | "full";
export type ContainerGutter = "none" | "sm" | "md" | "lg" | "responsive";

const maxWidths: Record<ContainerWidth, string> = {
  compact: "var(--layout-content, 34rem)",
  content: "var(--container-content, 72rem)",
  comfortable: "var(--layout-content-wide, 42rem)",
  wide: "var(--container-wide, 96rem)",
  full: "none",
};

const gutters: Record<Exclude<ContainerGutter, "none">, string> = {
  sm: "var(--space-sm)",
  md: "var(--space-md)",
  lg: "var(--space-lg)",
  responsive: "clamp(var(--space-sm), 4vw, var(--space-lg))",
};

export interface ContainerProps extends HTMLAttributes<HTMLDivElement> {
  width?: ContainerWidth;
  gutter?: ContainerGutter;
}

export const Container = forwardRef<HTMLDivElement, ContainerProps>(
  ({ width = "content", gutter = "none", className, style, ...props }, ref) => (
    <div
      ref={ref}
      className={className}
      data-container-width={width}
      data-container-gutter={gutter}
      style={{
        width: "100%",
        maxWidth: maxWidths[width],
        marginInline: "auto",
        paddingInline: gutter === "none" ? undefined : gutters[gutter],
        ...style,
      }}
      {...props}
    />
  ),
);
