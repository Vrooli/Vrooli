/**
 * @libraryId react-component-library:Container
 * @displayName Container
 * @description Container constrains readable content widths without hard-coded dimensions.
 * @version 1.1.2
 * @tags ["primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource primitives.container */
import { forwardRef, type HTMLAttributes } from "react";
import "./Container.css";

export type ContainerWidth = "compact" | "content" | "comfortable" | "wide" | "full";
export type ContainerGutter = "none" | "sm" | "md" | "lg" | "responsive";

export interface ContainerProps extends HTMLAttributes<HTMLDivElement> {
  width?: ContainerWidth;
  gutter?: ContainerGutter;
}

export const Container = forwardRef<HTMLDivElement, ContainerProps>(
  ({ width = "content", gutter = "none", className, style, ...props }, ref) => (
    <div data-testid="primitives.container"
      ref={ref}
      className={className}
      data-container-width={width}
      data-container-gutter={gutter}
      style={style}
      {...props}
    />
  ),
);
