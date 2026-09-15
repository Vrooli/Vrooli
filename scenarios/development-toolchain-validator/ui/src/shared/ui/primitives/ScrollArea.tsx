import { forwardRef, type HTMLAttributes } from "react";
import { cn } from "../../lib/utils";

/**
 * ScrollArea primitive — native overflow scroll wrapped with the design-tokens
 * scrollbar styling. The scrollbar palette comes from `design-tokens.css`
 * (`::-webkit-scrollbar-*` + `scrollbar-color`); this wrapper just defines
 * the container and applies `overflow-auto`.
 */
export interface ScrollAreaProps extends HTMLAttributes<HTMLDivElement> {
  /** Maximum height — accepts any valid CSS height token via Tailwind. */
  maxHeight?: string;
}

export const ScrollArea = forwardRef<HTMLDivElement, ScrollAreaProps>(
  ({ className, maxHeight, style, ...props }, ref) => (
    <div
      ref={ref}
      className={cn("overflow-auto", className)}
      style={maxHeight ? { ...style, maxHeight } : style}
      {...props}
    />
  ),
);
ScrollArea.displayName = "ScrollArea";
